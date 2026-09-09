// Package outbox implements the guide's Outbox Pattern: a business change
// and the event describing it are written together, ATOMICALLY, into one
// store — solving the dual-write problem where a database update and a
// separate message-broker publish could otherwise succeed or fail
// independently. A separate Relay then delivers outbox rows to
// "subscribers" (standing in for a real message broker, Module 20-style),
// with genuine at-least-once semantics: a crash between publishing and
// marking a row sent causes a harmless redelivery, not a lost message.
package outbox

import (
	"fmt"
	"sync"
)

type OutboxRow struct {
	ID        int
	EventType string
	Payload   string
	Sent      bool
}

// Store simulates ONE atomic transaction covering both a business table
// and the outbox table — in a real system this is a genuine database
// transaction (Module 17's Begin/Commit/Rollback); here a single Mutex
// covering both maps stands in for that same atomicity guarantee.
type Store struct {
	mu      sync.Mutex
	orders  map[string]string // business data: orderID -> status
	outbox  []*OutboxRow
	nextID  int
}

func NewStore() *Store {
	return &Store{orders: make(map[string]string)}
}

// PlaceOrderWithEvent is the ONE atomic operation the dual-write problem
// requires: the business change (the order record) and the outbox event
// describing it are written together, under the SAME lock — they can
// never succeed or fail independently of each other.
func (s *Store) PlaceOrderWithEvent(orderID, eventPayload string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.orders[orderID]; exists {
		return fmt.Errorf("order %q already exists", orderID)
	}
	s.orders[orderID] = "placed"

	s.nextID++
	s.outbox = append(s.outbox, &OutboxRow{ID: s.nextID, EventType: "order.placed", Payload: eventPayload})
	return nil
}

// unsentRows and markSent are the Relay's only access points into the
// store — deliberately narrow, so the Relay can never touch business data
// directly, only the outbox rows it's responsible for delivering.
func (s *Store) unsentRows() []*OutboxRow {
	s.mu.Lock()
	defer s.mu.Unlock()
	var rows []*OutboxRow
	for _, r := range s.outbox {
		if !r.Sent {
			rows = append(rows, r)
		}
	}
	return rows
}

func (s *Store) markSent(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.outbox {
		if r.ID == id {
			r.Sent = true
			return
		}
	}
}

// Subscriber stands in for a real message broker subscriber (Module 20) —
// Relay.Poll delivers to it exactly like a broker client would.
type Subscriber func(eventType, payload string) error

// Relay is the SEPARATE process (a goroutine here; a standalone service
// in production) that actually delivers outbox rows — decoupled from the
// business logic in PlaceOrderWithEvent entirely.
type Relay struct {
	store      *Store
	subscriber Subscriber
}

func NewRelay(store *Store, subscriber Subscriber) *Relay {
	return &Relay{store: store, subscriber: subscriber}
}

// Poll delivers every currently-unsent row. If delivery succeeds, the row
// is marked sent; if it fails, the row is left unsent, to be retried on
// the NEXT Poll — real at-least-once delivery, not just a description of it.
func (r *Relay) Poll() (delivered int, err error) {
	for _, row := range r.store.unsentRows() {
		if pubErr := r.subscriber(row.EventType, row.Payload); pubErr != nil {
			return delivered, fmt.Errorf("delivering outbox row %d: %w", row.ID, pubErr)
		}
		r.store.markSent(row.ID)
		delivered++
	}
	return delivered, nil
}

// SimulateCrashBeforeMarkingSent delivers a row but DELIBERATELY skips
// marking it sent — reproducing the "relay crashed after publishing but
// before acknowledging" scenario from the guide, so a subsequent Poll can
// be shown redelivering it (a genuine, harmless duplicate an idempotent
// subscriber — Module 21's webhook receiver pattern — handles correctly).
func (r *Relay) SimulateCrashBeforeMarkingSent() (delivered int, err error) {
	rows := r.store.unsentRows()
	if len(rows) == 0 {
		return 0, nil
	}
	row := rows[0]
	if pubErr := r.subscriber(row.EventType, row.Payload); pubErr != nil {
		return 0, pubErr
	}
	// deliberately NOT calling markSent — simulating the crash
	return 1, nil
}
