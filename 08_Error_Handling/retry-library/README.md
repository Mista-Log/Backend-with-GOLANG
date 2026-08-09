# Project 2 — Retry Library

```
retry-library/
├── go.mod
├── main.go          package main — demo
└── retry/
    └── retry.go       package retry — the actual library
```

```bash
cd retry-library
go run .
```

## What's Demonstrated Here

- **`panic`/`recover` at a genuine system boundary** — `callSafely` is
  *exactly* the case the guide calls out as legitimate: `Do` cannot control
  what the caller's `fn` does, so it recovers any panic and turns it into an
  ordinary returned error, guaranteeing a flaky (or buggy) operation can
  never crash the whole retry loop.
- **The functional options pattern** — `WithMaxAttempts`, `WithBaseDelay`,
  `WithMaxDelay` are each a function *returning* a function
  (`func(*config)`), passed as variadic arguments to `Do`. This is Module
  03's higher-order functions applied to configuration — a genuinely common
  real-world Go idiom (you'll see this exact shape in popular libraries like
  `grpc-go` and the AWS SDK).
- **`Permanent()` + `errors.As`** — wraps an error to mean "don't retry
  this," and `Do` checks for it every attempt via `errors.As(err, &perm)`.
  This is the `errors.As`-for-custom-types pattern from the guide, used to
  make a *control-flow* decision (stop retrying) rather than just extracting
  data.
- **`RetryError` wraps the last real error via `Unwrap`** — so even after
  every attempt is exhausted, `errors.Is`/`errors.As` can still reach
  whatever actually went wrong on the final try, exactly like
  `validation.FieldError` in the other project.

```
┌──────────────────────────────────────────────────────────┐
│   Do(fn, opts...)                                                │
│        │                                                            │
│        ▼                                                              │
│   for attempt := 1; attempt <= maxAttempts; attempt++ {                  │
│       err := callSafely(fn)   ◀── panic-safe call                          │
│       if err == nil { return nil }              // success                    │
│       if errors.As(err, &permanentError) { return }  // give up NOW              │
│       sleep(delay); delay *= 2                    // backoff, then retry          │
│   }                                                                                    │
│   return &RetryError{Attempts, lastErr}          // out of attempts                     │
└──────────────────────────────────────────────────────────┘
```

## Case Study: Why `Permanent` Exists at All

Without it, `Do` has no way to distinguish "this failed because of a
transient network blip — try again" from "this failed because the API key
is wrong, and it will fail identically on every single retry." Retrying the
second kind of failure is worse than doing nothing: it wastes `maxAttempts`
tries and the full backoff delay on something guaranteed to keep failing,
delaying the moment the caller finds out. `retry.Permanent(err)` lets the
*function being retried* — which is the only thing that actually knows
whether a given failure is transient or not — communicate that back to `Do`
without `Do` needing any special-case logic about HTTP status codes,
specific error messages, or anything else caller-specific. This is a clean
separation of concerns: `Do` only knows "retry, unless told not to";
deciding *when* not to is entirely up to the code calling it.

## Try It Yourself
- Add jitter to the backoff (a small random amount added to `delay` before
  each sleep) — a standard technique to avoid many retrying clients all
  hammering a recovering service at the exact same moments
- Add a `WithOnRetry(func(attempt int, err error))` option — another
  functional option, called between attempts, useful for logging
- Change `Do` to accept a `context.Context` and stop retrying (returning the
  context's error) if it's cancelled mid-backoff — a natural next step once
  cancellation is covered in a later module
