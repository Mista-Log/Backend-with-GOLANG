// Package ratelimiter implements a token bucket, the guide's default
// recommendation — same algorithm as Module 16's, restated here as its
// own standalone toolkit piece.
package ratelimiter

import (
	"sync"
	"time"
)

type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	capacity   float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

func New(capacity, refillRate float64) *TokenBucket {
	return &TokenBucket{tokens: capacity, capacity: capacity, refillRate: refillRate, lastRefill: time.Now()}
}

func (b *TokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.lastRefill = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
