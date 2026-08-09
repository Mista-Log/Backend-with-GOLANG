// Demo entry point for the retry package — run with: go run .
package main

import (
	"errors"
	"fmt"
	"time"

	"retrydemo/retry"
)

func main() {
	fmt.Println("=== Succeeds on the 3rd attempt ===")
	attempt := 0
	err := retry.Do(func() error {
		attempt++
		fmt.Printf("  attempt %d...\n", attempt)
		if attempt < 3 {
			return fmt.Errorf("connection refused")
		}
		return nil
	}, retry.WithMaxAttempts(5), retry.WithBaseDelay(10*time.Millisecond))
	fmt.Println("result:", err) // nil — succeeded before running out of attempts

	fmt.Println("\n=== Permanent error — stops immediately, no wasted retries ===")
	tries := 0
	err = retry.Do(func() error {
		tries++
		fmt.Printf("  attempt %d...\n", tries)
		return retry.Permanent(fmt.Errorf("invalid API key"))
	}, retry.WithMaxAttempts(5), retry.WithBaseDelay(10*time.Millisecond))
	fmt.Println("result:", err)
	fmt.Println("total attempts actually made:", tries) // 1 — Permanent stopped it early

	fmt.Println("\n=== Exhausts every attempt, returns a RetryError ===")
	err = retry.Do(func() error {
		return fmt.Errorf("timeout")
	}, retry.WithMaxAttempts(3), retry.WithBaseDelay(10*time.Millisecond))
	fmt.Println("result:", err)

	var retryErr *retry.RetryError
	if errors.As(err, &retryErr) {
		fmt.Printf("structured info -> Attempts: %d, last Err: %v\n", retryErr.Attempts, retryErr.Err)
	}

	fmt.Println("\n=== A panicking function doesn't crash the program ===")
	err = retry.Do(func() error {
		var m map[string]int
		m["this"] = 1 // panics: assignment to entry in nil map
		return nil
	}, retry.WithMaxAttempts(2), retry.WithBaseDelay(10*time.Millisecond))
	fmt.Println("result:", err) // a normal error, NOT a crashed program

	fmt.Println("\n=== Custom backoff timing, observed directly ===")
	start := time.Now()
	retry.Do(func() error {
		return fmt.Errorf("still failing")
	}, retry.WithMaxAttempts(4), retry.WithBaseDelay(50*time.Millisecond), retry.WithMaxDelay(200*time.Millisecond))
	fmt.Printf("total time for 4 attempts with doubling backoff: %v\n", time.Since(start))
}
