// Project 3: Email Worker
//
// Sends a batch of emails concurrently, rate-limited to a configurable
// number in flight at once (a buffered-channel semaphore), retrying
// transient failures with backoff. Demonstrates sync.Once for a lazily
// created, shared "expensive" client, atomic counters for stats under heavy
// concurrent access, and an RWMutex-protected Config that's read constantly
// but only occasionally written — including a live change mid-run.
package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// --- Lazily-initialized shared client, via sync.Once -----------------------

type EmailClient struct {
	connectionID int
}

var (
	clientOnce sync.Once
	client     *EmailClient
)

// getClient is called from every sending goroutine, but the expensive
// "connect" work inside once.Do runs EXACTLY ONCE, no matter how many
// goroutines call getClient() concurrently — the rest just get the
// already-built client once it's ready.
func getClient() *EmailClient {
	clientOnce.Do(func() {
		fmt.Println("[client] establishing connection (this should print ONCE)...")
		time.Sleep(50 * time.Millisecond) // pretend this is a real, slow handshake
		client = &EmailClient{connectionID: rand.Intn(100000)}
	})
	return client
}

func (c *EmailClient) send(to, subject string) error {
	// Simulate a transient failure ~20% of the time.
	if rand.Intn(100) < 20 {
		return fmt.Errorf("transient send failure to %s", to)
	}
	time.Sleep(time.Duration(10+rand.Intn(30)) * time.Millisecond)
	return nil
}

// --- RWMutex-protected, live-updatable config -----------------------------

type Config struct {
	mu            sync.RWMutex
	maxRetries    int
	fromAddress   string
}

func NewConfig() *Config {
	return &Config{maxRetries: 2, fromAddress: "noreply@example.com"}
}

func (c *Config) MaxRetries() int {
	c.mu.RLock() // many sending goroutines call this concurrently — all allowed at once
	defer c.mu.RUnlock()
	return c.maxRetries
}

func (c *Config) FromAddress() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fromAddress
}

// SetMaxRetries is called ONCE, mid-run, from main below — it needs the
// EXCLUSIVE lock, which waits for any in-progress reads to finish and
// blocks new reads until it completes.
func (c *Config) SetMaxRetries(n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxRetries = n
}

// --- Stats, via atomics -----------------------------------------------

type Stats struct {
	sent    atomic.Int64
	failed  atomic.Int64
	retried atomic.Int64
}

func (s *Stats) String() string {
	return fmt.Sprintf("sent=%d failed=%d retried=%d", s.sent.Load(), s.failed.Load(), s.retried.Load())
}

// --- Sending, with rate limiting and retry -----------------------------

// sendWithRetry retries transient failures with a short backoff, up to
// cfg.MaxRetries() — read fresh each call, so a live config change (see
// main) takes effect for emails sent AFTER the change, without needing to
// restart anything.
func sendWithRetry(client *EmailClient, cfg *Config, stats *Stats, to string) {
	maxRetries := cfg.MaxRetries()
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			stats.retried.Add(1)
			time.Sleep(time.Duration(attempt) * 20 * time.Millisecond) // simple linear backoff
		}
		if err := client.send(to, "Your receipt"); err != nil {
			lastErr = err
			continue
		}
		stats.sent.Add(1)
		return
	}
	stats.failed.Add(1)
	_ = lastErr // in a real system: log lastErr here
}

func main() {
	recipients := make([]string, 25)
	for i := range recipients {
		recipients[i] = fmt.Sprintf("user%d@example.com", i+1)
	}

	cfg := NewConfig()
	stats := &Stats{}

	const maxConcurrentSends = 4
	semaphore := make(chan struct{}, maxConcurrentSends) // buffered channel AS a rate limiter

	var wg sync.WaitGroup
	fmt.Printf("Sending %d emails, max %d concurrent, from %s\n\n", len(recipients), maxConcurrentSends, cfg.FromAddress())

	for i, to := range recipients {
		// Halfway through, tighten retry policy live — every send AFTER this
		// point picks it up automatically via cfg.MaxRetries() re-reading it.
		// Called directly here (not in its own goroutine) — Lock() still
		// coordinates correctly against every concurrent RLock() in flight
		// from sender goroutines; no extra goroutine is needed to prove that.
		if i == len(recipients)/2 {
			fmt.Println("\n[config] reducing max retries from 2 to 1, mid-run\n")
			cfg.SetMaxRetries(1)
		}

		wg.Add(1)
		go func(to string) {
			defer wg.Done()

			semaphore <- struct{}{}        // ACQUIRE — blocks if maxConcurrentSends already in flight
			defer func() { <-semaphore }() // RELEASE — always runs, even on early return

			c := getClient() // sync.Once ensures the real connection setup happens once, ever
			sendWithRetry(c, cfg, stats, to)
		}(to)
	}

	wg.Wait()

	fmt.Println("\n=== Final stats ===")
	fmt.Println(stats.String())
	fmt.Printf("client connection ID used throughout: %d (proves ONE client was shared)\n", client.connectionID)
}
