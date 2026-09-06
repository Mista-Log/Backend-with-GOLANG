// Package webhook implements the guide's Webhook and Idempotency sections
// as a small, reusable piece: track which idempotency keys have already
// been processed, and refuse to process the same one twice — regardless
// of which service (Payment Gateway, Virtual Accounts, or anything added
// later) is receiving the events.
package webhook

import "sync"

// IdempotencyStore is deliberately its own tiny type, not folded into
// paymentgateway — the same "has this key been seen before?" mechanism is
// needed anywhere an external system might redeliver an event, which in a
// real fintech backend is almost every inbound integration, not just one.
type IdempotencyStore struct {
	mu   sync.Mutex
	seen map[string]bool
}

func NewIdempotencyStore() *IdempotencyStore {
	return &IdempotencyStore{seen: make(map[string]bool)}
}

// CheckAndMark is ONE atomic operation — check AND record — for the exact
// same reason Module 12's Concurrent Web Scraper used sync.Map's
// LoadOrStore instead of a separate Load-then-Store: a gap between
// checking and marking is a real race two concurrent webhook deliveries
// could both slip through.
func (s *IdempotencyStore) CheckAndMark(key string) (alreadyProcessed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seen[key] {
		return true
	}
	s.seen[key] = true
	return false
}
