// Package bank is a small banking library — Account, Bank, and a currency
// parser — written specifically to be thoroughly tested. See bank_test.go,
// bank_bench_test.go, amount_fuzz_test.go, and handler_test.go for the
// full test suite exercising every technique from Module 14's guide.
package bank

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// --- Amount parsing (the Fuzz Testing target) -----------------------------

type Amount float64

func (a Amount) String() string {
	return fmt.Sprintf("$%.2f", float64(a))
}

// ParseAmount handles arbitrary, possibly-untrusted input strings — exactly
// the kind of function fuzz testing is best at: strconv.ParseFloat never
// panics on garbage input, but that doesn't mean every edge case (leading
// zeros, "+"/"-" signs, scientific notation, unicode whitespace...) is
// necessarily handled the way we'd want.
func ParseAmount(s string) (Amount, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "$")
	if s == "" {
		return 0, fmt.Errorf("empty amount")
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q: %w", s, err)
	}
	return Amount(v), nil
}

// --- TransactionLogger (the Mocking seam) -----------------------------

// TransactionLogger is a small interface specifically so tests can swap in
// a fake — see bank_test.go's FakeLogger and TestWithdraw_LogsTransaction.
type TransactionLogger interface {
	Log(accountID string, amount float64)
}

// NoopLogger is the default when no logger is supplied — satisfies
// TransactionLogger by doing nothing, so callers never need a nil check.
type NoopLogger struct{}

func (NoopLogger) Log(accountID string, amount float64) {}

// --- Account -----------------------------------------------------

type Account struct {
	ID string

	mu      sync.Mutex
	balance float64
	logger  TransactionLogger
}

func NewAccount(id string, initialBalance float64, logger TransactionLogger) *Account {
	if logger == nil {
		logger = NoopLogger{}
	}
	return &Account{ID: id, balance: initialBalance, logger: logger}
}

func (a *Account) Balance() float64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.balance
}

func (a *Account) Deposit(amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("deposit amount must be positive, got %.2f", amount)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.balance += amount
	a.logger.Log(a.ID, amount)
	return nil
}

func (a *Account) Withdraw(amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("withdrawal amount must be positive, got %.2f", amount)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if amount > a.balance {
		return fmt.Errorf("insufficient funds: balance %.2f, requested %.2f", a.balance, amount)
	}
	a.balance -= amount
	a.logger.Log(a.ID, -amount)
	return nil
}

// --- Bank -----------------------------------------------------

type Bank struct {
	mu       sync.RWMutex
	accounts map[string]*Account
	logger   TransactionLogger
}

func NewBank(logger TransactionLogger) *Bank {
	if logger == nil {
		logger = NoopLogger{}
	}
	return &Bank{accounts: make(map[string]*Account), logger: logger}
}

func (b *Bank) OpenAccount(id string, initialBalance float64) (*Account, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, exists := b.accounts[id]; exists {
		return nil, fmt.Errorf("account %q already exists", id)
	}
	acc := NewAccount(id, initialBalance, b.logger)
	b.accounts[id] = acc
	return acc, nil
}

func (b *Bank) Account(id string) (*Account, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	acc, ok := b.accounts[id]
	if !ok {
		return nil, fmt.Errorf("no account %q", id)
	}
	return acc, nil
}

// Transfer withdraws from one account and deposits to another, rolling back
// the withdrawal if the deposit somehow fails partway through — see
// bank_test.go's TestTransfer_RollsBackOnFailure for this exact scenario.
func (b *Bank) Transfer(fromID, toID string, amount float64) error {
	from, err := b.Account(fromID)
	if err != nil {
		return fmt.Errorf("transfer source: %w", err)
	}
	to, err := b.Account(toID)
	if err != nil {
		return fmt.Errorf("transfer destination: %w", err)
	}

	if err := from.Withdraw(amount); err != nil {
		return fmt.Errorf("transfer withdrawal: %w", err)
	}
	if err := to.Deposit(amount); err != nil {
		from.Deposit(amount) // roll back the withdrawal — best-effort, ignoring this error
		return fmt.Errorf("transfer deposit: %w", err)
	}
	return nil
}
