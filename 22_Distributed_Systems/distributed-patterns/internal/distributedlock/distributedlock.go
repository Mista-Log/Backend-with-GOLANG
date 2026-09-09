// Package distributedlock implements a lease-based lock — the guide's
// Distributed Locks section. A sync.Mutex only coordinates goroutines
// within one process (Module 12); this simulates the SHARED, external
// store (Redis, etcd) that lets separate PROCESSES coordinate, via a
// single Mutex standing in for "a store every process can reach."
package distributedlock

import (
	"sync"
	"time"
)

type lease struct {
	holderID  string
	expiresAt time.Time
}

// LockStore simulates the shared store — in production this IS Redis or
// etcd; the Mutex here plays the role their own internal atomicity plays
// for a real SETNX-with-expiry operation.
type LockStore struct {
	mu     sync.Mutex
	leases map[string]*lease // resource key -> current lease, if any
}

func NewLockStore() *LockStore {
	return &LockStore{leases: make(map[string]*lease)}
}

// TryLock acquires the lock for `key` if it's free OR expired — mirroring
// a real Redis SETNX-with-TTL: this is the ONE atomic check-and-set
// operation (Module 12's LoadOrStore instinct, applied here) that makes
// the lock safe against two callers racing to acquire it simultaneously.
func (s *LockStore) TryLock(key, holderID string, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, exists := s.leases[key]
	if exists && time.Now().Before(current.expiresAt) && current.holderID != holderID {
		return false // someone else holds a still-valid lease
	}
	s.leases[key] = &lease{holderID: holderID, expiresAt: time.Now().Add(ttl)}
	return true
}

// Unlock releases the lock — but ONLY if the caller still actually holds
// it. This check-then-delete must be atomic (the guide's Lua-script note)
// specifically to prevent a caller whose lease already EXPIRED (and was
// reacquired by someone else) from accidentally deleting that OTHER
// holder's now-valid lease.
func (s *LockStore) Unlock(key, holderID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, exists := s.leases[key]
	if !exists || current.holderID != holderID {
		return false // we don't hold it (anymore) — nothing to release
	}
	delete(s.leases, key)
	return true
}

// IsLocked reports whether `key` currently has a live (unexpired) holder
// — useful for the demo to show a lock's state without racing to acquire it.
func (s *LockStore) IsLocked(key string) (holderID string, locked bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.leases[key]
	if !exists || time.Now().After(current.expiresAt) {
		return "", false
	}
	return current.holderID, true
}
