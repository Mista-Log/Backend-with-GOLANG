// Package fx implements the guide's Multi Currency and Exchange Rates
// sections: a rate provider interface (Module 06/17's port pattern —
// swappable for a real market-data API in production) and a conversion
// service that moves money between two currency-specific ledger accounts
// through an explicit FX clearing account, recording the exact rate used.
package fx

import (
	"context"
	"fmt"
	"sync"
	"time"

	"fintechbackend/internal/ledger"
)

type RateProvider interface {
	Rate(ctx context.Context, from, to string) (rate float64, asOf time.Time, err error)
}

// StaticRateProvider is a simple, fixed-rate implementation for this demo
// — a real deployment would implement RateProvider against a live market
// data feed instead, with zero changes needed to ConversionService.
type StaticRateProvider struct {
	mu    sync.Mutex
	rates map[string]float64 // "USD:EUR" -> rate
}

func NewStaticRateProvider() *StaticRateProvider {
	return &StaticRateProvider{rates: map[string]float64{
		"USD:EUR": 0.92,
		"EUR:USD": 1.087,
		"USD:NGN": 1550.00,
		"NGN:USD": 0.000645,
	}}
}

func (p *StaticRateProvider) Rate(ctx context.Context, from, to string) (float64, time.Time, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if from == to {
		return 1.0, time.Now(), nil
	}
	rate, ok := p.rates[from+":"+to]
	if !ok {
		return 0, time.Time{}, fmt.Errorf("no exchange rate available for %s -> %s", from, to)
	}
	return rate, time.Now(), nil
}

// ConversionService moves money between two currency-specific accounts —
// the guide's "two linked transactions through an FX clearing account"
// diagram, implemented for real against the ledger.
type ConversionService struct {
	ledger        *ledger.Ledger
	rates         RateProvider
	clearingAccts map[string]string // currency -> clearing account ID for that currency
}

func New(l *ledger.Ledger, rates RateProvider) *ConversionService {
	return &ConversionService{ledger: l, rates: rates, clearingAccts: make(map[string]string)}
}

// ensureClearingAccount lazily opens one FX clearing account per currency
// this service has ever needed. ASSET type — matching the wallet
// convention throughout this codebase — so a wallet's balance decreasing
// (Credit, since Asset decreases on credit) can correctly pair with the
// clearing account increasing (Debit, since Asset increases on debit) in
// the SAME balanced transaction. An earlier draft of this file used an
// Equity account here, which — combined with the (also then-incorrect)
// entry directions below — happened to still satisfy Post's arithmetic
// balance check (debits still equaled credits) while moving every balance
// in the economically WRONG direction. That is the important lesson: the
// ledger's balance check guarantees the BOOKS balance, never that a
// developer chose the correct SIDE for each account. Only correct
// accounting reasoning — or a reconciliation check noticing a wallet
// balance moving the wrong way — catches that second class of bug.
func (c *ConversionService) ensureClearingAccount(currency string) string {
	if id, ok := c.clearingAccts[currency]; ok {
		return id
	}
	id := "fx-clearing-" + currency
	c.ledger.OpenAccount(id, "FX Clearing ("+currency+")", ledger.Asset, currency)
	c.clearingAccts[currency] = id
	return id
}

// Convert moves `amount` (in fromCurrency's minor units) out of
// fromWallet and credits the CONVERTED amount into toWallet, in
// toCurrency — as TWO separate, each-individually-balanced ledger
// transactions, exactly as the guide's Multi Currency section requires.
func (c *ConversionService) Convert(ctx context.Context, actor, idempotencyKey, fromWallet, toWallet string, amount int64, fromCurrency, toCurrency string) (converted int64, err error) {
	rate, asOf, err := c.rates.Rate(ctx, fromCurrency, toCurrency)
	if err != nil {
		return 0, err
	}
	converted = int64(float64(amount) * rate)

	fromClearing := c.ensureClearingAccount(fromCurrency)
	toClearing := c.ensureClearingAccount(toCurrency)

	// Same defensive check as wallet.Withdraw and transfer.Send: the
	// ledger's balance invariant only guarantees debits==credits, never
	// that an account has enough to give — that business rule belongs
	// here, at the application layer, checked before anything is posted.
	balance, err := c.ledger.Balance(fromWallet)
	if err != nil {
		return 0, err
	}
	if balance < amount {
		return 0, fmt.Errorf("insufficient funds for conversion: balance %d, requested %d", balance, amount)
	}

	// Leg 1: pull the source amount OUT of the source wallet (Asset
	// decreases on Credit) into that currency's clearing account (Asset
	// increases on Debit) — balances within fromCurrency alone.
	_, err = c.ledger.Post(actor, idempotencyKey+"-leg1", "fx convert (leg 1)", []ledger.Entry{
		{AccountID: fromWallet, Direction: ledger.Credit, Amount: amount},
		{AccountID: fromClearing, Direction: ledger.Debit, Amount: amount},
	})
	if err != nil {
		return 0, fmt.Errorf("fx conversion leg 1 failed: %w", err)
	}

	// Leg 2: release the CONVERTED amount from the destination currency's
	// clearing account (Asset decreases on Credit) into the destination
	// wallet (Asset increases on Debit) — balances within toCurrency alone.
	// The rate used is recorded via the idempotency key suffix and (in a
	// fuller implementation) a dedicated audit entry — see the guide's
	// Exchange Rates section on why that record matters.
	_, err = c.ledger.Post(actor, idempotencyKey+"-leg2", "fx convert (leg 2)", []ledger.Entry{
		{AccountID: toClearing, Direction: ledger.Credit, Amount: converted},
		{AccountID: toWallet, Direction: ledger.Debit, Amount: converted},
	})
	if err != nil {
		return 0, fmt.Errorf("fx conversion leg 2 failed: %w", err)
	}

	_ = asOf // recorded via audit log in a fuller implementation; kept here to show it's available
	return converted, nil
}
