// Package retry re-implements Module 08's exponential-backoff retry as a
// standalone toolkit piece, so the demo can show it working TOGETHER with
// the circuit breaker — retry handles one transient blip; the breaker
// (wrapped around the same call) watches the aggregate failure rate.
package retry

import "time"

func Do(fn func() error, maxAttempts int, baseDelay time.Duration) error {
	var lastErr error
	delay := baseDelay
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := fn(); err != nil {
			lastErr = err
			if attempt < maxAttempts {
				time.Sleep(delay)
				delay *= 2
			}
			continue
		}
		return nil
	}
	return lastErr
}
