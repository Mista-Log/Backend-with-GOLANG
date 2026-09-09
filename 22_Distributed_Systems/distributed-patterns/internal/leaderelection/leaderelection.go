// Package leaderelection is, deliberately, almost the exact same
// mechanism as distributedlock — the guide calls this out directly: a
// distributed lock IS a lease, just scoped to "who's the leader" instead
// of "who has exclusive access to one resource." Kept as its own package
// because the vocabulary and usage pattern (continuous renewal, automatic
// failover on a missed renewal) is distinct enough to be worth naming
// separately, even though the underlying primitive is identical.
package leaderelection

import (
	"sync"
	"time"
)

type Lease struct {
	HolderID  string
	ExpiresAt time.Time
}

type LeaseStore struct {
	mu    sync.Mutex
	lease *Lease
}

func NewLeaseStore() *LeaseStore {
	return &LeaseStore{}
}

// TryAcquireOrRenew is the guide's function, implemented for real: it
// succeeds if the lease is free, expired, or ALREADY held by this exact
// node (a renewal) — and fails if a DIFFERENT node currently holds a
// still-valid lease.
func (s *LeaseStore) TryAcquireOrRenew(nodeID string, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lease == nil || time.Now().After(s.lease.ExpiresAt) || s.lease.HolderID == nodeID {
		s.lease = &Lease{HolderID: nodeID, ExpiresAt: time.Now().Add(ttl)}
		return true
	}
	return false
}

func (s *LeaseStore) CurrentLeader() (nodeID string, isValid bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lease == nil || time.Now().After(s.lease.ExpiresAt) {
		return "", false
	}
	return s.lease.HolderID, true
}

// Node simulates one participant in the cluster — it repeatedly tries to
// become (or remain) leader, on its own schedule, exactly like a real
// process would poll/renew against etcd or Consul in a background goroutine.
type Node struct {
	ID       string
	store    *LeaseStore
	ttl      time.Duration
	stopCh   chan struct{}
	stopOnce sync.Once
}

func NewNode(id string, store *LeaseStore, ttl time.Duration) *Node {
	return &Node{ID: id, store: store, ttl: ttl, stopCh: make(chan struct{})}
}

// Run renews (or attempts to acquire) leadership on every tick — a real
// implementation would renew well before the TTL expires, exactly as
// modeled here by ticking at a fraction of the TTL.
func (n *Node) Run(tickInterval time.Duration, onBecomeLeader func(nodeID string), onLoseLeadership func(nodeID string)) {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	wasLeader := false
	for {
		select {
		case <-ticker.C:
			isLeader := n.store.TryAcquireOrRenew(n.ID, n.ttl)
			if isLeader && !wasLeader {
				onBecomeLeader(n.ID)
			}
			if !isLeader && wasLeader {
				onLoseLeadership(n.ID)
			}
			wasLeader = isLeader
		case <-n.stopCh:
			return
		}
	}
}

// Stop simulates this node crashing — it simply stops renewing, exactly
// like a real crashed process would, letting its lease expire naturally.
func (n *Node) Stop() {
	n.stopOnce.Do(func() { close(n.stopCh) })
}
