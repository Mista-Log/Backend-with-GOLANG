# Go for Beginners — Module 13: Context

## Contents

1. **[13-context.md](./13-context.md)** — `context.Background` and
   `context.TODO` (identical behavior, different documentation intent),
   `WithCancel`, `WithTimeout`, `WithDeadline` (and why `defer cancel()`
   matters even on the success path), `WithValue` (including the dedicated
   key-type trick to avoid cross-package collisions, and firm guidance on
   when *not* to use it), and propagation — the convention that `ctx` is
   always an explicit first parameter, never a struct field, with a case
   study on exactly why. Diagrams throughout.

2. **[payment-timeout/](./payment-timeout)** — A 4-layer call chain sharing
   **one** top-level timeout, propagated down unmodified so the budget
   genuinely drains across every layer instead of resetting at each one — a
   case study makes this concrete with a "what if each layer got its own
   fresh timeout instead" comparison. Also carries a request ID via
   `context.WithValue` for correlated logging, and runs several payments
   concurrently to prove contexts are genuinely per-call.

3. **[api-timeout/](./api-timeout)** — Three *real* HTTP hops (client →
   gateway → downstream, all local via `httptest`), comparing a gateway
   handler that correctly derives its context from the inbound request
   against one using `context.Background()` — a real, common, and subtle
   bug. A `sync.WaitGroup` proves the wasted server-side work empirically
   rather than just describing it.

## Suggested Order

```
Context guide ──▶ Payment Timeout ──▶ API Timeout
                    (propagation through    (propagation across REAL
                     plain function calls,    network hops, and what
                     context values)           breaks when it's skipped)
```

Payment Timeout establishes the core propagation habit in the simplest
possible setting (in-process function calls). API Timeout raises the
stakes: the same principle, now across real HTTP boundaries, where getting
it wrong has a genuine resource-leak cost instead of just an incorrect
timing.

## Quick Reference: Running Either Project

```bash
cd payment-timeout && go run main.go
cd api-timeout && go run main.go
```

Both are fully self-contained and offline — no real network, nothing to set
up. Run either one a few times; both include intentionally randomized or
tight timing so you'll see both the "succeeds" and "times out" paths across
a few runs.

*Note: this module builds on Modules 00–12 — start there first if you
haven't already, especially Module 12, which used `context` throughout its
projects without a dedicated explanation. This module is that explanation.*
