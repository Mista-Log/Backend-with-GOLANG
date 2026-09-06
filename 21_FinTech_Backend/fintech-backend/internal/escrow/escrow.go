// Package escrow implements Project — Escrow System: funds held by the
// platform (a neutral third party, in ledger terms a Liability account
// the platform owes to WHOMEVER ends up entitled to it) until a release
// or refund decision is made — the guide's Escrow section, made real.
package escrow

import (
	"fmt"
	"sync"

	"fintechbackend/internal/ledger"
	"fintechbackend/internal/wallet"
)

const escrowAccountID = "escrow-holding"

type Status string

const (
	Held      Status = "held"
	Released  Status = "released"
	Refunded  Status = "refunded"
)

type Escrow struct {
	ID       string
	FromUser string
	Amount   int64
	Currency string
	Status   Status
}

type Service struct {
	ledger  *ledger.Ledger
	wallets *wallet.Service

	mu      sync.Mutex
	escrows map[string]*Escrow
	nextID  int
}

func New(l *ledger.Ledger, w *wallet.Service, currency string) *Service {
	// ASSET type — matching wallet.go's convention (wallets are modeled as
	// Asset accounts, the user's own point of view, throughout this
	// codebase) so escrow entries use the exact same debit/credit shape
	// as a direct wallet-to-wallet transfer: debit increases, credit
	// decreases, on both sides of every entry below.
	l.OpenAccount(escrowAccountID, "Escrow Holding", ledger.Asset, currency)
	return &Service{ledger: l, wallets: w, escrows: make(map[string]*Escrow)}
}

// Create moves funds OUT of the payer's wallet and INTO the shared escrow
// account, tagged with this specific escrow's ID — the money is fully,
// physically (in ledger terms) held by the platform now, not just marked
// "pending" on the payer's own wallet.
func (s *Service) Create(fromUserID string, amount int64, currency string) (*Escrow, error) {
	fromWallet, err := s.wallets.Get(fromUserID)
	if err != nil {
		return nil, err
	}

	// Same defensive check as wallet.Withdraw, transfer.Send, and
	// fx.Convert: the ledger's balance invariant only guarantees
	// debits==credits, never that an account has enough to give.
	balance, err := s.ledger.Balance(fromWallet.AccountID)
	if err != nil {
		return nil, err
	}
	if balance < amount {
		return nil, fmt.Errorf("insufficient funds: balance %d, requested %d", balance, amount)
	}

	s.mu.Lock()
	id := fmt.Sprintf("escrow_%d", s.nextID+1)
	s.nextID++
	e := &Escrow{ID: id, FromUser: fromUserID, Amount: amount, Currency: currency, Status: Held}
	s.escrows[id] = e
	s.mu.Unlock()

	_, err = s.ledger.Post("escrow", "escrow-create-"+id, fmt.Sprintf("escrow hold %s", id), []ledger.Entry{
		{AccountID: fromWallet.AccountID, Direction: ledger.Credit, Amount: amount},
		{AccountID: escrowAccountID, Direction: ledger.Debit, Amount: amount},
	})
	if err != nil {
		return nil, err
	}
	return e, nil
}

// Release pays the held funds out to whoever the buyer/platform has
// decided is entitled to them (e.g., a seller, once an order ships).
func (s *Service) Release(escrowID, toUserID string) error {
	s.mu.Lock()
	e, ok := s.escrows[escrowID]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("no escrow %q", escrowID)
	}
	if e.Status != Held {
		return fmt.Errorf("escrow %q is not held (status: %s) — cannot release", escrowID, e.Status)
	}

	toWallet, err := s.wallets.Get(toUserID)
	if err != nil {
		return err
	}

	_, err = s.ledger.Post("escrow", "escrow-release-"+escrowID, fmt.Sprintf("escrow release %s", escrowID), []ledger.Entry{
		{AccountID: escrowAccountID, Direction: ledger.Credit, Amount: e.Amount},
		{AccountID: toWallet.AccountID, Direction: ledger.Debit, Amount: e.Amount},
	})
	if err != nil {
		return err
	}

	s.mu.Lock()
	e.Status = Released
	s.mu.Unlock()
	return nil
}

// Refund returns held funds back to the original payer — the deal fell
// through, or was cancelled before the release condition was met.
func (s *Service) Refund(escrowID string) error {
	s.mu.Lock()
	e, ok := s.escrows[escrowID]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("no escrow %q", escrowID)
	}
	if e.Status != Held {
		return fmt.Errorf("escrow %q is not held (status: %s) — cannot refund", escrowID, e.Status)
	}

	fromWallet, err := s.wallets.Get(e.FromUser)
	if err != nil {
		return err
	}

	_, err = s.ledger.Post("escrow", "escrow-refund-"+escrowID, fmt.Sprintf("escrow refund %s", escrowID), []ledger.Entry{
		{AccountID: escrowAccountID, Direction: ledger.Credit, Amount: e.Amount},
		{AccountID: fromWallet.AccountID, Direction: ledger.Debit, Amount: e.Amount},
	})
	if err != nil {
		return err
	}

	s.mu.Lock()
	e.Status = Refunded
	s.mu.Unlock()
	return nil
}

func (s *Service) Get(escrowID string) (*Escrow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.escrows[escrowID]
	if !ok {
		return nil, fmt.Errorf("no escrow %q", escrowID)
	}
	return e, nil
}
