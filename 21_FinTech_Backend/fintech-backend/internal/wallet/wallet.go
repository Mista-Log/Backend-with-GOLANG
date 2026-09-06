// Package wallet implements Project — Wallet System. A wallet is a
// product-facing wrapper around a ledger.Account: deposit, withdraw, and
// balance, translated into properly-balanced double-entry postings. This
// package NEVER touches a balance directly — every operation is a
// ledger.Post call.
package wallet

import (
	"fmt"
	"sync"

	"fintechbackend/internal/ledger"
)

// externalFundingAccountID is a single platform-level liability account
// representing "money that entered the system from outside" (a card
// payment, a bank deposit) — every user deposit's OTHER leg posts here,
// so the whole ledger always stays balanced even though deposits look
// one-sided from a single user's point of view.
const externalFundingAccountID = "external-funding"

type Wallet struct {
	UserID    string
	AccountID string
	Currency  string
}

type Service struct {
	mu      sync.Mutex
	ledger  *ledger.Ledger
	wallets map[string]*Wallet // by UserID
}

func New(l *ledger.Ledger) *Service {
	l.OpenAccount(externalFundingAccountID, "External Funding Source", ledger.Liability, "USD")
	return &Service{ledger: l, wallets: make(map[string]*Wallet)}
}

// CreateWallet opens a new ledger account (type Asset — a wallet is money
// the USER can spend, which is an asset from their own point of view; see
// the guide's note on user-facing framing vs. accounting direction) for a
// brand-new user.
func (s *Service) CreateWallet(userID, currency string) (*Wallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.wallets[userID]; exists {
		return nil, fmt.Errorf("user %q already has a wallet", userID)
	}
	accountID := "wallet-" + userID
	s.ledger.OpenAccount(accountID, "Wallet ("+userID+")", ledger.Asset, currency)
	w := &Wallet{UserID: userID, AccountID: accountID, Currency: currency}
	s.wallets[userID] = w
	return w, nil
}

func (s *Service) Get(userID string) (*Wallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.wallets[userID]
	if !ok {
		return nil, fmt.Errorf("no wallet for user %q", userID)
	}
	return w, nil
}

// Deposit credits money INTO a user's wallet — an asset increases on
// DEBIT (the guide's table), so the wallet leg is a debit; the offsetting
// credit lands on the shared external-funding liability account.
func (s *Service) Deposit(actor, idempotencyKey, userID string, amount int64) (*ledger.Transaction, error) {
	w, err := s.Get(userID)
	if err != nil {
		return nil, err
	}
	if amount <= 0 {
		return nil, fmt.Errorf("deposit amount must be positive")
	}
	return s.ledger.Post(actor, idempotencyKey, fmt.Sprintf("deposit to %s", userID), []ledger.Entry{
		{AccountID: w.AccountID, Direction: ledger.Debit, Amount: amount},
		{AccountID: externalFundingAccountID, Direction: ledger.Credit, Amount: amount},
	})
}

// Withdraw reverses that: credit reduces an asset, so the wallet leg is a
// credit here. Checks sufficient balance FIRST — the ledger's balance
// check would reject an unbalanced posting, but a NEGATIVE wallet balance
// is a business rule this layer enforces, not something double-entry
// bookkeeping alone prevents (the ledger would happily balance a
// transaction that overdraws an account; only application logic stops it).
func (s *Service) Withdraw(actor, idempotencyKey, userID string, amount int64) (*ledger.Transaction, error) {
	w, err := s.Get(userID)
	if err != nil {
		return nil, err
	}
	if amount <= 0 {
		return nil, fmt.Errorf("withdrawal amount must be positive")
	}
	balance, err := s.ledger.Balance(w.AccountID)
	if err != nil {
		return nil, err
	}
	if balance < amount {
		return nil, fmt.Errorf("insufficient funds: balance %d, requested %d", balance, amount)
	}
	return s.ledger.Post(actor, idempotencyKey, fmt.Sprintf("withdrawal from %s", userID), []ledger.Entry{
		{AccountID: externalFundingAccountID, Direction: ledger.Debit, Amount: amount},
		{AccountID: w.AccountID, Direction: ledger.Credit, Amount: amount},
	})
}

func (s *Service) Balance(userID string) (int64, error) {
	w, err := s.Get(userID)
	if err != nil {
		return 0, err
	}
	return s.ledger.Balance(w.AccountID)
}
