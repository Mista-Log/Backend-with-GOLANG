// Package savings implements Project — Savings App: moving funds from a
// customer's spendable wallet into a locked savings account, which
// accrues interest over time — two ledger accounts per customer instead
// of one, connected by transfers exactly like any other double-entry
// movement.
package savings

import (
	"fmt"
	"sync"

	"fintechbackend/internal/ledger"
	"fintechbackend/internal/wallet"
)

const interestPayableAccountID = "interest-payable"

type Account struct {
	UserID    string
	AccountID string // the ledger account backing this user's LOCKED savings balance
	Currency  string
}

type Service struct {
	ledger  *ledger.Ledger
	wallets *wallet.Service

	mu       sync.Mutex
	accounts map[string]*Account // by UserID
}

func New(l *ledger.Ledger, w *wallet.Service, currency string) *Service {
	// LIABILITY, not Expense — the platform now OWES this interest to the
	// customer (increases on Credit), which is what correctly balances
	// against the customer's savings Asset account increasing (on Debit)
	// in AccrueInterest below. An earlier draft of this file used an
	// Expense account here, which increases on Debit too — making it
	// mathematically impossible to balance against the savings account
	// increasing in the same direction. This is exactly the kind of
	// mistake the ledger's own balance check (ledger.go's ErrUnbalanced)
	// is designed to catch before it can ever reach production.
	l.OpenAccount(interestPayableAccountID, "Interest Payable", ledger.Liability, currency)
	return &Service{ledger: l, wallets: w, accounts: make(map[string]*Account)}
}

func (s *Service) OpenSavingsAccount(userID, currency string) (*Account, error) {
	if _, err := s.wallets.Get(userID); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.accounts[userID]; exists {
		return nil, fmt.Errorf("user %q already has a savings account", userID)
	}
	accountID := "savings-" + userID
	s.ledger.OpenAccount(accountID, "Savings ("+userID+")", ledger.Asset, currency)
	acc := &Account{UserID: userID, AccountID: accountID, Currency: currency}
	s.accounts[userID] = acc
	return acc, nil
}

func (s *Service) get(userID string) (*Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, ok := s.accounts[userID]
	if !ok {
		return nil, fmt.Errorf("no savings account for user %q", userID)
	}
	return acc, nil
}

// Lock moves funds FROM the spendable wallet INTO the locked savings
// account — both Asset accounts, so this is structurally identical to a
// transfer (transfer.go), just between two accounts the SAME user owns.
func (s *Service) Lock(userID string, amount int64) error {
	w, err := s.wallets.Get(userID)
	if err != nil {
		return err
	}
	savingsAcc, err := s.get(userID)
	if err != nil {
		return err
	}
	balance, err := s.ledger.Balance(w.AccountID)
	if err != nil {
		return err
	}
	if balance < amount {
		return fmt.Errorf("insufficient wallet balance: have %d, need %d", balance, amount)
	}
	_, err = s.ledger.Post("savings", fmt.Sprintf("savings-lock-%s-%d", userID, amount),
		fmt.Sprintf("lock %d into savings for %s", amount, userID), []ledger.Entry{
			{AccountID: savingsAcc.AccountID, Direction: ledger.Debit, Amount: amount},
			{AccountID: w.AccountID, Direction: ledger.Credit, Amount: amount},
		})
	return err
}

// Unlock moves funds back the other way — savings to spendable wallet.
func (s *Service) Unlock(userID string, amount int64) error {
	w, err := s.wallets.Get(userID)
	if err != nil {
		return err
	}
	savingsAcc, err := s.get(userID)
	if err != nil {
		return err
	}
	balance, err := s.ledger.Balance(savingsAcc.AccountID)
	if err != nil {
		return err
	}
	if balance < amount {
		return fmt.Errorf("insufficient savings balance: have %d, need %d", balance, amount)
	}
	_, err = s.ledger.Post("savings", fmt.Sprintf("savings-unlock-%s-%d", userID, amount),
		fmt.Sprintf("unlock %d from savings for %s", amount, userID), []ledger.Entry{
			{AccountID: w.AccountID, Direction: ledger.Debit, Amount: amount},
			{AccountID: savingsAcc.AccountID, Direction: ledger.Credit, Amount: amount},
		})
	return err
}

// AccrueInterest posts interest for EVERY savings account at once, at the
// given annual rate — in a real system this runs on a schedule (a daily
// cron job or a Module 12-style background goroutine with a ticker); here
// it's called directly by cmd/demo to keep the demo's timing visible and
// deterministic rather than actually waiting on a real clock.
func (s *Service) AccrueInterest(annualRatePercent float64) error {
	s.mu.Lock()
	accounts := make([]*Account, 0, len(s.accounts))
	for _, acc := range s.accounts {
		accounts = append(accounts, acc)
	}
	s.mu.Unlock()

	for _, acc := range accounts {
		balance, err := s.ledger.Balance(acc.AccountID)
		if err != nil {
			return err
		}
		if balance <= 0 {
			continue
		}
		// Simplified: apply the full annual rate as one accrual, standing
		// in for "one period's worth of interest" — a real system would
		// prorate this per day/month based on the actual accrual schedule.
		interest := int64(float64(balance) * annualRatePercent / 100)
		if interest <= 0 {
			continue
		}
		_, err = s.ledger.Post("savings", fmt.Sprintf("interest-%s-%d", acc.UserID, interest),
			fmt.Sprintf("interest accrual for %s", acc.UserID), []ledger.Entry{
				{AccountID: acc.AccountID, Direction: ledger.Debit, Amount: interest},
				{AccountID: interestPayableAccountID, Direction: ledger.Credit, Amount: interest},
			})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Balance(userID string) (int64, error) {
	acc, err := s.get(userID)
	if err != nil {
		return 0, err
	}
	return s.ledger.Balance(acc.AccountID)
}
