// Package retry runs a function with exponential backoff, recovering any
// panic inside it so a flaky operation can never crash its caller, and
// offering a Permanent() escape hatch for errors that retrying can never
// fix (bad input, a 404, an auth failure — as opposed to a timeout that
// might succeed on the next attempt).
package retry

import (
	"errors"
	"fmt"
	"time"
)

type config struct {
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
}

func defaultConfig() *config {
	return &config{
		maxAttempts: 3,
		baseDelay:   100 * time.Millisecond,
		maxDelay:    2 * time.Second,
	}
}

// Option is the FUNCTIONAL OPTIONS pattern — each Option is a function that
// mutates a *config, and Do accepts any number of them. This is the same
// "function as a first-class value" idea from Module 03's higher-order
// functions, applied to configuration instead of computation.
type Option func(*config)

func WithMaxAttempts(n int) Option {
	return func(c *config) { c.maxAttempts = n }
}

func WithBaseDelay(d time.Duration) Option {
	return func(c *config) { c.baseDelay = d }
}

func WithMaxDelay(d time.Duration) Option {
	return func(c *config) { c.maxDelay = d }
}

// permanentError marks an error as NOT WORTH RETRYING — Do checks for this
// via errors.As and stops immediately, instead of burning through every
// remaining attempt on something that will never succeed.
type permanentError struct {
	err error
}

func (e *permanentError) Error() string { return e.err.Error() }
func (e *permanentError) Unwrap() error { return e.err }

// Permanent wraps an error to signal "don't retry this." A validation
// failure or a 404 are permanent; a network timeout is not.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &permanentError{err: err}
}

// RetryError is returned when every attempt was exhausted — it wraps the
// LAST underlying error via Unwrap, so callers can still errors.Is/errors.As
// through it to find out what actually went wrong on the final try.
type RetryError struct {
	Attempts int
	Err      error
}

func (e *RetryError) Error() string {
	return fmt.Sprintf("failed after %d attempt(s): %v", e.Attempts, e.Err)
}

func (e *RetryError) Unwrap() error {
	return e.Err
}

// Do runs fn, retrying with exponential backoff on failure, until it
// succeeds, hits a Permanent error, or exhausts maxAttempts.
func Do(fn func() error, opts ...Option) error {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	var lastErr error
	delay := cfg.baseDelay

	for attempt := 1; attempt <= cfg.maxAttempts; attempt++ {
		err := callSafely(fn)
		if err == nil {
			return nil // success — nothing more to do
		}
		lastErr = err

		var perm *permanentError
		if errors.As(err, &perm) {
			return perm.err // stop immediately — retrying would be pointless
		}

		if attempt == cfg.maxAttempts {
			break // out of attempts — fall through to RetryError below
		}

		time.Sleep(delay)
		delay *= 2
		if delay > cfg.maxDelay {
			delay = cfg.maxDelay
		}
	}

	return &RetryError{Attempts: cfg.maxAttempts, Err: lastErr}
}

// callSafely wraps fn in a RECOVER so a PANIC inside the retried function
// becomes an ordinary returned error instead of crashing the whole program
// — this uses the exact defer+recover+named-return pattern from the guide.
func callSafely(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered during attempt: %v", r)
		}
	}()
	return fn()
}
