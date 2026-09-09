// Package replication simulates leader-follower replication with
// artificial network latency, making the guide's synchronous-vs-
// asynchronous trade-off (and the CAP theorem / Consistency sections it
// connects to) directly observable rather than just described.
package replication

import (
	"sync"
	"time"
)

type Follower struct {
	Name    string
	Latency time.Duration // simulated network delay to THIS follower

	mu  sync.Mutex
	log []string
}

func (f *Follower) receive(entry string) {
	time.Sleep(f.Latency) // stand-in for real network latency
	f.mu.Lock()
	defer f.mu.Unlock()
	f.log = append(f.log, entry)
}

func (f *Follower) Read() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.log))
	copy(out, f.log)
	return out
}

type Leader struct {
	mu        sync.Mutex
	log       []string
	followers []*Follower
}

func NewLeader(followers ...*Follower) *Leader {
	return &Leader{followers: followers}
}

// WriteSync waits for EVERY follower to confirm before returning — strong
// consistency (any read, anywhere, after this returns, sees the write),
// at the cost of taking as long as the SLOWEST follower.
func (l *Leader) WriteSync(entry string) time.Duration {
	start := time.Now()
	l.mu.Lock()
	l.log = append(l.log, entry)
	l.mu.Unlock()

	var wg sync.WaitGroup
	for _, f := range l.followers {
		wg.Add(1)
		go func(f *Follower) {
			defer wg.Done()
			f.receive(entry)
		}(f)
	}
	wg.Wait() // BLOCKS until every follower has the entry
	return time.Since(start)
}

// WriteAsync returns IMMEDIATELY after committing locally — available and
// fast, but a reader hitting a follower right after this returns may see
// STALE data (the write hasn't arrived there yet) — eventual consistency,
// made directly observable in the demo by reading a follower immediately
// after this call returns.
func (l *Leader) WriteAsync(entry string) time.Duration {
	start := time.Now()
	l.mu.Lock()
	l.log = append(l.log, entry)
	l.mu.Unlock()

	for _, f := range l.followers {
		go f.receive(entry) // fire-and-forget — the leader does NOT wait
	}
	return time.Since(start)
}

func (l *Leader) Read() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.log))
	copy(out, l.log)
	return out
}
