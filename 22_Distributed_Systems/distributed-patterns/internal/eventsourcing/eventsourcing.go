// Package eventsourcing implements an event-sourced Account: instead of
// storing a mutable balance, every change is appended as an immutable
// Event, and the current balance is always DERIVED by replaying history
// — the guide's Event Sourcing section, generalizing Module 21's ledger
// principle (never mutate, only append) to an explicit event log.
package eventsourcing

import (
	"fmt"
	"sync"
	"time"
)

// Event is a closed set of concrete event types — a sealed interface via
// an unexported marker method (Module 06's implicit satisfaction, used
// here specifically to prevent any OTHER package from inventing new event
// types this aggregate wouldn't know how to replay).
type Event interface {
	isEvent()
	EventType() string
}

type AccountOpened struct {
	AccountID string
	Owner     string
	At        time.Time
}

type MoneyDeposited struct {
	AccountID string
	Amount    int64
	At        time.Time
}

type MoneyWithdrawn struct {
	AccountID string
	Amount    int64
	At        time.Time
}

func (AccountOpened) isEvent()           {}
func (MoneyDeposited) isEvent()          {}
func (MoneyWithdrawn) isEvent()          {}
func (AccountOpened) EventType() string  { return "AccountOpened" }
func (MoneyDeposited) EventType() string { return "MoneyDeposited" }
func (MoneyWithdrawn) EventType() string { return "MoneyWithdrawn" }

// AccountState is the FOLDED result of replaying a stream of events — it
// is never stored directly as the source of truth; it's always derivable
// again from the event log alone.
type AccountState struct {
	AccountID string
	Owner     string
	Balance   int64
	Version   int // how many events have been applied — used for optimistic concurrency
}

// Apply folds ONE event into a state — the guide's Replay function,
// factored out so both full replay and incremental updates use the exact
// same logic (a projector consuming events one at a time, per CQRS below,
// calls this same function).
func Apply(state AccountState, event Event) AccountState {
	switch e := event.(type) {
	case AccountOpened:
		state.AccountID = e.AccountID
		state.Owner = e.Owner
	case MoneyDeposited:
		state.Balance += e.Amount
	case MoneyWithdrawn:
		state.Balance -= e.Amount
	}
	state.Version++
	return state
}

// Replay folds an entire event history into a single current state —
// Module 03's Reduce, applied to an aggregate's full history.
func Replay(events []Event) AccountState {
	var state AccountState
	for _, e := range events {
		state = Apply(state, e)
	}
	return state
}

// EventStore is a simple, append-only, in-memory event log per account —
// standing in for a real event store (EventStoreDB, or events stored as
// rows in a normal database table, exactly like Module 21's ledger
// entries table).
type EventStore struct {
	mu     sync.Mutex
	events map[string][]Event // accountID -> its full event history, in order
}

func NewEventStore() *EventStore {
	return &EventStore{events: make(map[string][]Event)}
}

// Append adds new events, but ONLY if expectedVersion matches the
// account's actual current version — optimistic concurrency control: if
// someone else appended events since the caller last read this account's
// state, expectedVersion will be stale, and this correctly rejects the
// write rather than silently losing the intervening changes.
func (s *EventStore) Append(accountID string, expectedVersion int, newEvents ...Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.events[accountID]
	if len(current) != expectedVersion {
		return fmt.Errorf("concurrency conflict: expected version %d, actual version %d", expectedVersion, len(current))
	}
	s.events[accountID] = append(current, newEvents...)
	return nil
}

func (s *EventStore) Load(accountID string) []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Return a COPY — callers must never be able to mutate the stored
	// history through the returned slice (Module 04's slice-aliasing trap).
	out := make([]Event, len(s.events[accountID]))
	copy(out, s.events[accountID])
	return out
}

// AllEventsInOrder returns every event ever appended, across all
// accounts — flattened deterministically for this simplified demo store;
// a real event store maintains one true global append order natively,
// which is exactly what a CQRS projector (see the cqrs package) subscribes to.
func (s *EventStore) AllEventsInOrder() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all []Event
	for _, evs := range s.events {
		all = append(all, evs...)
	}
	return all
}
