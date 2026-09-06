// Package fraud implements simple, explainable rule-based checks — the
// guide's Fraud Detection section. Real systems layer many such signals
// and often score rather than hard-block; these rules are kept simple and
// fully readable on purpose, so the LOGIC is the point, not sophistication.
package fraud

import (
	"fmt"
	"sync"
	"time"
)

type record struct {
	timestamp time.Time
	amount    int64
}

type Checker struct {
	mu      sync.Mutex
	history map[string][]record // userID -> recent transaction history

	velocityWindow    time.Duration
	velocityMax       int
	anomalyMultiplier int64
}

func New() *Checker {
	return &Checker{
		history:           make(map[string][]record),
		velocityWindow:    5 * time.Minute,
		velocityMax:       10,
		anomalyMultiplier: 10,
	}
}

// Check runs every rule BEFORE the transaction is allowed to proceed —
// called from transfer/withdraw code paths, never after the fact.
func (c *Checker) Check(userID string, amount int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	recent := c.recentLocked(userID, now)

	// VELOCITY: too many transactions in too little time.
	if len(recent) >= c.velocityMax {
		return fmt.Errorf("velocity check failed: %d transactions in the last %v", len(recent), c.velocityWindow)
	}

	// AMOUNT ANOMALY: far larger than this user's own typical transaction.
	if typical := typicalAmount(recent); typical > 0 && amount > typical*c.anomalyMultiplier {
		return fmt.Errorf("amount anomaly: %d is more than %dx this user's typical transaction size (%d)",
			amount, c.anomalyMultiplier, typical)
	}

	c.history[userID] = append(recent, record{timestamp: now, amount: amount})
	return nil
}

func (c *Checker) recentLocked(userID string, now time.Time) []record {
	var kept []record
	for _, r := range c.history[userID] {
		if now.Sub(r.timestamp) <= c.velocityWindow {
			kept = append(kept, r)
		}
	}
	return kept
}

// typicalAmount uses a simple average of recent transactions — a real
// system would likely use a median or a learned model instead, but the
// PATTERN (compare against the user's own history, not a global constant)
// is the part worth internalizing.
func typicalAmount(recent []record) int64 {
	if len(recent) == 0 {
		return 0
	}
	var total int64
	for _, r := range recent {
		total += r.amount
	}
	return total / int64(len(recent))
}
