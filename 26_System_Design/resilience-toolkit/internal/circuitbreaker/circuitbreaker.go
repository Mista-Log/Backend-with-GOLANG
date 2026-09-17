// Package circuitbreaker implements the guide's three-state machine:
// Closed (normal), Open (failing fast), Half-Open (one test call).
package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

type State int

const (
	Closed State = iota
	Open
	HalfOpen
)

var ErrOpen = errors.New("circuit breaker is open")

type Breaker struct {
	mu              sync.Mutex
	state           State
	failureCount    int
	failureThreshold int
	openedAt        time.Time
	resetTimeout    time.Duration
}

func New(failureThreshold int, resetTimeout time.Duration) *Breaker {
	return &Breaker{failureThreshold: failureThreshold, resetTimeout: resetTimeout}
}

// Call runs fn only if the breaker allows it — rejecting immediately (no
// call to fn at all) while Open, per the guide's "fail fast" definition.
func (b *Breaker) Call(fn func() error) error {
	b.mu.Lock()
	if b.state == Open {
		if time.Since(b.openedAt) < b.resetTimeout {
			b.mu.Unlock()
			return ErrOpen
		}
		b.state = HalfOpen // timeout elapsed — allow ONE test call through
	}
	b.mu.Unlock()

	err := fn()

	b.mu.Lock()
	defer b.mu.Unlock()
	if err != nil {
		b.failureCount++
		if b.state == HalfOpen || b.failureCount >= b.failureThreshold {
			b.state = Open
			b.openedAt = time.Now()
		}
		return err
	}
	// success — recover fully
	b.state = Closed
	b.failureCount = 0
	return nil
}

func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}
