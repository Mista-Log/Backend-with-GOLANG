// Resilience Toolkit demo — run with: go run ./cmd/demo
package main

import (
	"fmt"
	"time"

	"resiliencetoolkit/internal/backpressure"
	"resiliencetoolkit/internal/circuitbreaker"
	"resiliencetoolkit/internal/ratelimiter"
	"resiliencetoolkit/internal/retry"
)

func main() {
	fmt.Println("=== Circuit Breaker ===")
	cb := circuitbreaker.New(3, 200*time.Millisecond)
	failing := func() error { return fmt.Errorf("downstream is down") }

	for i := 1; i <= 5; i++ {
		err := cb.Call(failing)
		fmt.Printf("call %d: err=%v state=%v\n", i, err, cb.State())
	}
	fmt.Println("(after 3 failures, the breaker OPENS — call 4/5 fail FAST, no real call made)")

	time.Sleep(250 * time.Millisecond)
	err := cb.Call(func() error { return nil }) // a test call that SUCCEEDS
	fmt.Println("after reset timeout, one successful test call: err=", err, "state=", cb.State())

	fmt.Println()
	fmt.Println("=== Rate Limiter (token bucket) ===")
	rl := ratelimiter.New(3, 1) // capacity 3, refills 1/sec
	for i := 1; i <= 5; i++ {
		fmt.Printf("request %d allowed: %v\n", i, rl.Allow())
	}

	fmt.Println()
	fmt.Println("=== Retry (exponential backoff) ===")
	attempt := 0
	err = retry.Do(func() error {
		attempt++
		if attempt < 3 {
			return fmt.Errorf("attempt %d failed", attempt)
		}
		return nil
	}, 5, 10*time.Millisecond)
	fmt.Println("succeeded after", attempt, "attempts, err:", err)

	fmt.Println()
	fmt.Println("=== Backpressure (bounded worker pool) ===")
	pool := backpressure.New(2, 2) // 2 workers, queue depth 2
	start := time.Now()
	for i := 1; i <= 6; i++ {
		fmt.Printf("submitting job %d at %v\n", i, time.Since(start).Round(time.Millisecond))
		pool.Submit(func() {
			time.Sleep(100 * time.Millisecond)
		})
	}
	pool.Wait()
	pool.Close()
	fmt.Println("all jobs done at", time.Since(start).Round(time.Millisecond))
	fmt.Println("(later Submit calls BLOCKED once the queue filled — that's backpressure)")
}
