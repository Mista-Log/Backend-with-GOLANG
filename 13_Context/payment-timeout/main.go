// Project 1: Payment Timeout
//
// A 3-layer call chain (ProcessPayment -> ValidatePayment/ChargeCard ->
// callBankAPI), all sharing ONE top-level context. A single WithTimeout at
// the top propagates all the way down — if the bank call is slow enough to
// blow the overall budget, every layer notices via ctx.Done() and unwinds
// immediately, without needing its own separately-configured timeout.
// A request ID travels alongside via context.WithValue, for correlated logging.
package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type contextKey string

const requestIDKey contextKey = "requestID"

func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func requestIDFrom(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return "unknown"
}

func logf(ctx context.Context, format string, args ...any) {
	fmt.Printf("  [%s] "+format+"\n", append([]any{requestIDFrom(ctx)}, args...)...)
}

// ProcessPayment is the entry point — it doesn't create the timeout itself
// (that's main's job, since main knows the caller's actual budget); it just
// PROPAGATES whatever ctx it was handed straight down through every layer.
func ProcessPayment(ctx context.Context, amount float64) (string, error) {
	logf(ctx, "processing payment of $%.2f", amount)

	if err := ValidatePayment(ctx, amount); err != nil {
		return "", fmt.Errorf("validation failed: %w", err)
	}

	txID, err := ChargeCard(ctx, amount)
	if err != nil {
		return "", fmt.Errorf("charge failed: %w", err)
	}

	return txID, nil
}

func ValidatePayment(ctx context.Context, amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	select {
	case <-time.After(50 * time.Millisecond): // simulated validation work
		logf(ctx, "validation passed")
		return nil
	case <-ctx.Done():
		logf(ctx, "validation aborted: %v", ctx.Err())
		return ctx.Err()
	}
}

func ChargeCard(ctx context.Context, amount float64) (string, error) {
	select {
	case <-time.After(80 * time.Millisecond): // simulated card-network work
	case <-ctx.Done():
		logf(ctx, "charge aborted before reaching the bank: %v", ctx.Err())
		return "", ctx.Err()
	}
	return callBankAPI(ctx, amount)
}

// callBankAPI is the deepest, slowest, least predictable layer — exactly
// where a real system's timeout risk usually concentrates. Its latency is
// randomized specifically so SOME runs succeed within budget and some don't.
func callBankAPI(ctx context.Context, amount float64) (string, error) {
	bankLatency := time.Duration(100+rand.Intn(400)) * time.Millisecond

	select {
	case <-time.After(bankLatency):
		logf(ctx, "bank approved after %v", bankLatency.Round(time.Millisecond))
		return fmt.Sprintf("TX-%05d", rand.Intn(100000)), nil
	case <-ctx.Done():
		logf(ctx, "bank call abandoned after budget ran out (would have taken ~%v): %v",
			bankLatency.Round(time.Millisecond), ctx.Err())
		return "", ctx.Err()
	}
}

func runPayment(ctx context.Context, requestID string, amount float64, budget time.Duration) {
	ctx = withRequestID(ctx, requestID)
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel() // ALWAYS — releases the internal timer whether we finish early or not

	start := time.Now()
	txID, err := ProcessPayment(ctx, amount)
	elapsed := time.Since(start)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Printf("[%s] TIMED OUT after %v (budget was %v)\n\n", requestID, elapsed.Round(time.Millisecond), budget)
		} else {
			fmt.Printf("[%s] FAILED after %v: %v\n\n", requestID, elapsed.Round(time.Millisecond), err)
		}
		return
	}
	fmt.Printf("[%s] SUCCESS after %v: transaction %s\n\n", requestID, elapsed.Round(time.Millisecond), txID)
}

func main() {
	background := context.Background()

	fmt.Println("=== Payment with a generous 1s budget ===")
	runPayment(background, "req-001", 49.99, 1*time.Second)

	fmt.Println("=== Payment with a tight 150ms budget (likely to time out) ===")
	runPayment(background, "req-002", 19.99, 150*time.Millisecond)

	fmt.Println("=== Payment with an invalid amount (fails validation, not a timeout) ===")
	runPayment(background, "req-003", -5.00, 1*time.Second)

	fmt.Println("=== Several payments running concurrently, each with its own budget and request ID ===")
	done := make(chan struct{})
	requests := []struct {
		id     string
		amount float64
		budget time.Duration
	}{
		{"req-004", 10.00, 1 * time.Second},
		{"req-005", 25.50, 200 * time.Millisecond},
		{"req-006", 99.00, 1 * time.Second},
	}
	for _, r := range requests {
		go func(id string, amount float64, budget time.Duration) {
			runPayment(background, id, amount, budget)
			done <- struct{}{}
		}(r.id, r.amount, r.budget)
	}
	for range requests {
		<-done
	}
}
