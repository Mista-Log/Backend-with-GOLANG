package ledger

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

type AccountType string

const (
	Asset     AccountType = "asset"
	Liability AccountType = "liability"
	Equity    AccountType = "equity"
	Revenue   AccountType = "revenue"
	Expense   AccountType = "expense"
)

type Direction string

const (
	Debit  Direction = "debit"
	Credit Direction = "credit"
)

type Account struct {
	ID       string
	Name     string
	Type     AccountType
	Currency string
}

// Entry is one leg of a transaction. Amount is ALWAYS a positive int64 in
// minor units (cents) — the guide's "never use float64 for money" rule.
// Direction says which way it moves; Amount is never negative.
type Entry struct {
	AccountID string
	Direction Direction
	Amount    int64
}

type TransactionStatus string

const (
	Completed TransactionStatus = "completed"
	Rejected  TransactionStatus = "rejected"
)

type Transaction struct {
	ID             string
	IdempotencyKey string
	Description    string
	Entries        []Entry
	Status         TransactionStatus
	CreatedAt      time.Time
}

// ErrUnbalanced is returned when a transaction's debits and credits don't
// match, per-currency — the ledger's single most important invariant.
type ErrUnbalanced struct {
	Currency     string
	TotalDebits  int64
	TotalCredits int64
}

func (e *ErrUnbalanced) Error() string {
	return fmt.Sprintf("unbalanced transaction in %s: debits=%d credits=%d", e.Currency, e.TotalDebits, e.TotalCredits)
}

type Ledger struct {
	mu       sync.Mutex
	accounts map[string]*Account
	// entries stores EVERY entry ever posted, per account — a real balance
	// is always the SUM of these, never a separately-stored, directly-
	// editable number. This is what makes the ledger provably correct and
	// reconcilable (the guide's Reconciliation section).
	entries map[string][]Entry
	txns    map[string]*Transaction // by transaction ID
	idempotency map[string]*Transaction // by idempotency key — see Post()
	audit    *AuditLog
	nextTxID int
}

func New(audit *AuditLog) *Ledger {
	return &Ledger{
		accounts:    make(map[string]*Account),
		entries:     make(map[string][]Entry),
		txns:        make(map[string]*Transaction),
		idempotency: make(map[string]*Transaction),
		audit:       audit,
		nextTxID:    1,
	}
}

func (l *Ledger) OpenAccount(id, name string, accType AccountType, currency string) *Account {
	l.mu.Lock()
	defer l.mu.Unlock()
	acc := &Account{ID: id, Name: name, Type: accType, Currency: currency}
	l.accounts[id] = acc
	l.audit.Record("system", "ledger.open_account", id, map[string]any{"type": accType, "currency": currency})
	return acc
}

// Post is THE core operation of this entire project. Every money movement
// in the whole system — deposits, transfers, escrow holds, FX conversions
// — ultimately calls this one function.
func (l *Ledger) Post(actor string, idempotencyKey, description string, entries []Entry) (*Transaction, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// IDEMPOTENCY CHECK FIRST, before any validation or mutation — the
	// guide's Idempotency section. A retried request with a key we've
	// already processed gets the ORIGINAL result, verbatim, no matter what.
	if existing, ok := l.idempotency[idempotencyKey]; ok {
		return existing, nil
	}

	if len(entries) < 2 {
		return nil, fmt.Errorf("a transaction needs at least 2 entries, got %d", len(entries))
	}

	// BALANCE CHECK, per currency — the guide's Double Entry Accounting
	// section. This is computed BEFORE writing anything, so an unbalanced
	// transaction is rejected atomically: nothing is written at all.
	debits := make(map[string]int64)  // currency -> total debits
	credits := make(map[string]int64) // currency -> total credits
	for _, e := range entries {
		acc, ok := l.accounts[e.AccountID]
		if !ok {
			return nil, fmt.Errorf("unknown account %q", e.AccountID)
		}
		switch e.Direction {
		case Debit:
			debits[acc.Currency] += e.Amount
		case Credit:
			credits[acc.Currency] += e.Amount
		default:
			return nil, fmt.Errorf("invalid direction %q", e.Direction)
		}
	}
	currencies := make(map[string]bool)
	for c := range debits {
		currencies[c] = true
	}
	for c := range credits {
		currencies[c] = true
	}
	for currency := range currencies {
		if debits[currency] != credits[currency] {
			l.audit.Record(actor, "ledger.post_rejected", description, map[string]any{"reason": "unbalanced", "currency": currency})
			return nil, &ErrUnbalanced{Currency: currency, TotalDebits: debits[currency], TotalCredits: credits[currency]}
		}
	}

	tx := &Transaction{
		ID:             generateTxID(l.nextTxID),
		IdempotencyKey: idempotencyKey,
		Description:    description,
		Entries:        entries,
		Status:         Completed,
		CreatedAt:      time.Now(),
	}
	l.nextTxID++

	// Write every entry — this is the point a real implementation would
	// wrap in an actual database transaction (Module 17); here, the
	// Ledger's own Mutex plays the same "all or nothing, together" role.
	for _, e := range entries {
		l.entries[e.AccountID] = append(l.entries[e.AccountID], e)
	}
	l.txns[tx.ID] = tx
	l.idempotency[idempotencyKey] = tx

	l.audit.Record(actor, "ledger.post", tx.ID, map[string]any{
		"description": description,
		"entryCount":  len(entries),
	})

	return tx, nil
}

// Balance computes an account's current balance by summing every entry
// ever posted to it — never a stored, separately-editable number. Positive
// means the account's "natural" balance side (per its AccountType) has a
// net positive amount.
func (l *Ledger) Balance(accountID string) (int64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	acc, ok := l.accounts[accountID]
	if !ok {
		return 0, fmt.Errorf("unknown account %q", accountID)
	}

	var balance int64
	for _, e := range l.entries[accountID] {
		delta := e.Amount
		increasesBalance := accountIncreasesOnDebit(acc.Type) == (e.Direction == Debit)
		if !increasesBalance {
			delta = -delta
		}
		balance += delta
	}
	return balance, nil
}

// accountIncreasesOnDebit implements the guide's Double Entry Accounting
// table exactly: Asset and Expense accounts increase on DEBIT; Liability,
// Equity, and Revenue accounts increase on CREDIT.
func accountIncreasesOnDebit(t AccountType) bool {
	switch t {
	case Asset, Expense:
		return true
	default: // Liability, Equity, Revenue
		return false
	}
}

// AccountIDs returns every account ID ever opened — the raw material
// reconciliation's system-wide integrity check needs to sum across the
// whole ledger, not just one account at a time.
func (l *Ledger) AccountIDs() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	ids := make([]string, 0, len(l.accounts))
	for id := range l.accounts {
		ids = append(ids, id)
	}
	return ids
}

func (l *Ledger) Transaction(id string) (*Transaction, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	tx, ok := l.txns[id]
	return tx, ok
}

// EntriesFor returns every entry ever posted to an account, in order —
// the raw material Reconciliation and audit investigations both need.
func (l *Ledger) EntriesFor(accountID string) []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Entry, len(l.entries[accountID]))
	copy(out, l.entries[accountID])
	return out
}

func generateTxID(n int) string {
	return "txn_" + strconv.Itoa(n)
}
