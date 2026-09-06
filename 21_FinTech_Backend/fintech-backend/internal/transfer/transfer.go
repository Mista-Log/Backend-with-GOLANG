// Package transfer implements Project — Transfer API: moving money
// between two wallets, gated by KYC limits and fraud scoring BEFORE the
// ledger ever sees the request. This ordering matters — cheap, fast
// checks run first, so an obviously-bad request never reaches the part of
// the system responsible for correctness (the ledger itself).
package transfer

import (
	"fmt"

	"fintechbackend/internal/fraud"
	"fintechbackend/internal/kyc"
	"fintechbackend/internal/ledger"
	"fintechbackend/internal/wallet"
)

type Service struct {
	ledger  *ledger.Ledger
	wallets *wallet.Service
	kycSvc  *kyc.Service
	fraud   *fraud.Checker
}

func New(l *ledger.Ledger, w *wallet.Service, k *kyc.Service, f *fraud.Checker) *Service {
	return &Service{ledger: l, wallets: w, kycSvc: k, fraud: f}
}

// Send moves `amount` from fromUserID's wallet to toUserID's wallet.
// Order of checks matters: KYC (a hard regulatory limit) before fraud (a
// heuristic risk signal) before the ledger (the source of truth) — cheap,
// definitive checks first, expensive or fuzzy ones next, ground truth last.
func (s *Service) Send(actor, idempotencyKey, fromUserID, toUserID string, amount int64) (*ledger.Transaction, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("transfer amount must be positive")
	}

	if err := s.kycSvc.CheckLimit(fromUserID, amount); err != nil {
		return nil, fmt.Errorf("KYC check failed: %w", err)
	}
	if err := s.fraud.Check(fromUserID, amount); err != nil {
		return nil, fmt.Errorf("fraud check failed: %w", err)
	}

	fromWallet, err := s.wallets.Get(fromUserID)
	if err != nil {
		return nil, fmt.Errorf("sender: %w", err)
	}
	toWallet, err := s.wallets.Get(toUserID)
	if err != nil {
		return nil, fmt.Errorf("recipient: %w", err)
	}
	if fromWallet.Currency != toWallet.Currency {
		return nil, fmt.Errorf("currency mismatch: sender is %s, recipient is %s — use fx.ConversionService instead",
			fromWallet.Currency, toWallet.Currency)
	}

	balance, err := s.ledger.Balance(fromWallet.AccountID)
	if err != nil {
		return nil, err
	}
	if balance < amount {
		return nil, fmt.Errorf("insufficient funds: balance %d, requested %d", balance, amount)
	}

	// A wallet is an ASSET account (wallet.go's design) — crediting one
	// asset and debiting another is exactly how a direct transfer between
	// two "cash-like" accounts balances, with no intermediate account
	// needed at all (contrast this with escrow, which DOES need one).
	return s.ledger.Post(actor, idempotencyKey, fmt.Sprintf("transfer %s -> %s", fromUserID, toUserID), []ledger.Entry{
		{AccountID: toWallet.AccountID, Direction: ledger.Debit, Amount: amount},
		{AccountID: fromWallet.AccountID, Direction: ledger.Credit, Amount: amount},
	})
}
