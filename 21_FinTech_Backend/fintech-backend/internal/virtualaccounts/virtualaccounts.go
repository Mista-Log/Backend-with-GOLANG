// Package virtualaccounts implements Project — Virtual Accounts: each
// customer gets a unique, dedicated account number that an EXTERNAL payer
// (anyone with a real bank account) can send money to — the deposit
// arrives via the same bank-webhook pattern as paymentgateway, but
// identified by account NUMBER instead of a reference this system
// generated itself, since the sender is external and unaware of our
// internal reference scheme entirely.
package virtualaccounts

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"

	"fintechbackend/internal/ledger"
	"fintechbackend/internal/wallet"
	"fintechbackend/internal/webhook"
)

const bankFundingAccountID = "bank-settlement" // shared with paymentgateway — both represent
                                                   // "money that arrived from the real bank rail"

type VirtualAccount struct {
	Number string
	UserID string
}

type Service struct {
	ledger      *ledger.Ledger
	wallets     *wallet.Service
	idempotency *webhook.IdempotencyStore

	mu       sync.Mutex
	accounts map[string]*VirtualAccount // by account number
}

func New(l *ledger.Ledger, w *wallet.Service) *Service {
	// OpenAccount is safe to call more than once for the same ID (it's a
	// simple re-registration, not a create-or-fail) — this means
	// virtualaccounts and paymentgateway can each ensure the SHARED
	// bank-settlement account exists, in whichever order cmd/demo
	// constructs them, with no ordering dependency between the two.
	l.OpenAccount(bankFundingAccountID, "Bank Settlement Account", ledger.Liability, "USD")
	return &Service{
		ledger:      l,
		wallets:     w,
		idempotency: webhook.NewIdempotencyStore(),
		accounts:    make(map[string]*VirtualAccount),
	}
}

// Issue generates a unique account number for a customer — in a real
// system this comes FROM the partner bank via their own API; here it's
// generated locally, since the point being taught is the ROUTING logic
// that happens once a deposit against one of these numbers arrives.
func (s *Service) Issue(userID string) (*VirtualAccount, error) {
	if _, err := s.wallets.Get(userID); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	va := &VirtualAccount{Number: generateAccountNumber(), UserID: userID}
	s.accounts[va.Number] = va
	return va, nil
}

// HandleIncomingDeposit is what a real bank webhook (or, in a fuller
// system, bankapi itself) would call when SOMEONE ELSE sends money to one
// of these account numbers — an idempotency key is required here for the
// exact same reason as paymentgateway's webhook handler.
func (s *Service) HandleIncomingDeposit(accountNumber string, amount int64, idempotencyKey string) error {
	if s.idempotency.CheckAndMark(idempotencyKey) {
		return nil // duplicate delivery — safely ignored
	}

	s.mu.Lock()
	va, ok := s.accounts[accountNumber]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("no virtual account %q on file", accountNumber)
	}

	w, err := s.wallets.Get(va.UserID)
	if err != nil {
		return err
	}

	_, err = s.ledger.Post("virtual-accounts", idempotencyKey,
		fmt.Sprintf("incoming deposit to virtual account %s", accountNumber), []ledger.Entry{
			{AccountID: w.AccountID, Direction: ledger.Debit, Amount: amount},
			{AccountID: bankFundingAccountID, Direction: ledger.Credit, Amount: amount},
		})
	return err
}

func generateAccountNumber() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(9999999999))
	return fmt.Sprintf("%010d", n)
}
