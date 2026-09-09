// Package cqrs separates the WRITE side (commands, validated and appended
// as events via eventsourcing.EventStore) from the READ side (a
// denormalized, pre-computed read model) — the guide's CQRS section. The
// read model is updated by a Projector that must be explicitly told to
// "catch up," making the eventual-consistency lag between write and read
// completely visible rather than hidden behind an automatic background
// process.
package cqrs

import (
	"fmt"
	"sync"
	"time"

	"distributedpatterns/internal/eventsourcing"
)

// CommandHandler is the WRITE side — every command is validated against
// CURRENT state (loaded by replaying events, per Module 21's habit of
// checking balance before posting) before being appended.
type CommandHandler struct {
	store *eventsourcing.EventStore
}

func NewCommandHandler(store *eventsourcing.EventStore) *CommandHandler {
	return &CommandHandler{store: store}
}

func (h *CommandHandler) OpenAccount(accountID, owner string) error {
	existing := h.store.Load(accountID)
	if len(existing) != 0 {
		return fmt.Errorf("account %q already exists", accountID)
	}
	return h.store.Append(accountID, 0, eventsourcing.AccountOpened{
		AccountID: accountID, Owner: owner, At: time.Now(),
	})
}

func (h *CommandHandler) Deposit(accountID string, amount int64) error {
	events := h.store.Load(accountID)
	if len(events) == 0 {
		return fmt.Errorf("no account %q", accountID)
	}
	if amount <= 0 {
		return fmt.Errorf("deposit amount must be positive")
	}
	return h.store.Append(accountID, len(events), eventsourcing.MoneyDeposited{
		AccountID: accountID, Amount: amount, At: time.Now(),
	})
}

func (h *CommandHandler) Withdraw(accountID string, amount int64) error {
	events := h.store.Load(accountID)
	if len(events) == 0 {
		return fmt.Errorf("no account %q", accountID)
	}
	state := eventsourcing.Replay(events) // CURRENT balance, derived fresh from history
	if amount <= 0 {
		return fmt.Errorf("withdrawal amount must be positive")
	}
	if state.Balance < amount {
		return fmt.Errorf("insufficient balance: have %d, requested %d", state.Balance, amount)
	}
	return h.store.Append(accountID, len(events), eventsourcing.MoneyWithdrawn{
		AccountID: accountID, Amount: amount, At: time.Now(),
	})
}

// ReadModel is the QUERY side — a plain, denormalized map, optimized for
// fast lookups, with NO event-replay logic involved at query time at all.
type ReadModel struct {
	mu       sync.RWMutex
	balances map[string]eventsourcing.AccountState
}

func NewReadModel() *ReadModel {
	return &ReadModel{balances: make(map[string]eventsourcing.AccountState)}
}

func (r *ReadModel) GetBalance(accountID string) (eventsourcing.AccountState, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	state, ok := r.balances[accountID]
	return state, ok
}

func (r *ReadModel) apply(state eventsourcing.AccountState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.balances[state.AccountID] = state
}

// Projector consumes events from the store and updates the read model —
// this is the ONE piece of code that bridges the write side to the read
// side, and it's ENTIRELY separate from both CommandHandler and any
// query path. In a real system this runs continuously in the background
// (a goroutine, or a genuine message consumer per Module 20); here it's
// invoked explicitly (CatchUp) so the demo can show the read model BEFORE
// and AFTER catching up, making the "eventually consistent" gap visible.
type Projector struct {
	store         *eventsourcing.EventStore
	readModel     *ReadModel
	lastProcessed map[string]int // accountID -> how many events already folded into the read model
}

func NewProjector(store *eventsourcing.EventStore, readModel *ReadModel) *Projector {
	return &Projector{store: store, readModel: readModel, lastProcessed: make(map[string]int)}
}

// CatchUp folds any events appended since the last call into the read
// model — one account at a time, since this simplified event store keeps
// per-account logs (a real global-stream projector would instead consume
// one ordered stream across every account at once).
func (p *Projector) CatchUp(accountIDs []string) {
	for _, id := range accountIDs {
		events := p.store.Load(id)
		alreadyProcessed := p.lastProcessed[id]
		if alreadyProcessed >= len(events) {
			continue // nothing new for this account
		}

		state, _ := p.readModel.GetBalance(id)
		for _, e := range events[alreadyProcessed:] {
			state = eventsourcing.Apply(state, e)
		}
		p.readModel.apply(state)
		p.lastProcessed[id] = len(events)
	}
}
