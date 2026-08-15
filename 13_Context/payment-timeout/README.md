# Project 1 — Payment Timeout

```bash
cd payment-timeout
go run main.go
```

Run it a few times — the 150ms-budget payment (`req-002`) will sometimes
succeed and sometimes time out, since the simulated bank latency is
randomized between 100-500ms. That variability is the point: this is
exactly the kind of real-world timing uncertainty timeouts exist to guard
against.

## What's Demonstrated Here

- **ONE timeout, created ONCE, propagated through every layer** —
  `runPayment` is the only place `context.WithTimeout` is called.
  `ProcessPayment`, `ValidatePayment`, `ChargeCard`, and `callBankAPI` all
  just receive and pass along the same `ctx` — none of them creates its own
  timeout, and none of them needs to know the actual budget value. If the
  overall deadline expires while deep inside `callBankAPI`, that's where
  `ctx.Done()` fires — but it would fire identically no matter which layer
  happened to be running when time ran out.
- **`context.WithValue` for request-scoped correlation** — every `logf`
  call automatically includes the request ID, pulled from `ctx` rather than
  threaded through every function's parameter list. This is exactly the
  "genuinely request-scoped metadata" use case the guide calls appropriate
  for context values — a request ID isn't a parameter any layer's *logic*
  depends on, it's purely for correlating log lines back to one call.
- **`errors.Is(err, context.DeadlineExceeded)`** — distinguishing "this
  failed because of a timeout" from "this failed for a business reason"
  (like `req-003`'s invalid amount), using Module 08's `errors.Is` habit
  against the specific sentinel `context` itself defines.
- **Multiple independent, concurrently-running payments**, each with its
  *own* context, budget, and request ID — proving that contexts really are
  per-call, not shared, ambient state (the guide's case study on why
  `Context` never lives in a struct field, made concrete: if `ProcessPayment`
  somehow shared one `ctx` across calls, `req-004`, `req-005`, and `req-006`
  running concurrently would corrupt each other's deadlines and request IDs).

```
┌──────────────────────────────────────────────────────────┐
│   runPayment("req-002", ..., 150ms)                              │
│        │                                                            │
│        ▼                                                              │
│   ctx := WithTimeout(WithRequestID(background, "req-002"), 150ms)       │
│        │                                                                  │
│        ▼                                                                    │
│   ProcessPayment(ctx, ...)  ──▶  ValidatePayment(ctx, ...)  [50ms]             │
│        │                              (same ctx, same 150ms budget,              │
│        │                               now with only ~100ms left)                   │
│        ▼                                                                               │
│   ChargeCard(ctx, ...)  [80ms]  ──▶  callBankAPI(ctx, ...)  [100-500ms]                   │
│                                          (if the REMAINING budget runs out                   │
│                                           here, ctx.Done() fires and every                      │
│                                           layer above unwinds immediately)                         │
└──────────────────────────────────────────────────────────┘
```

## Case Study: The Budget Is Shared, Not Reset at Each Layer

A tempting-but-wrong design would give each layer its *own* fresh timeout —
`ValidatePayment` gets 150ms, then `ChargeCard` gets its own fresh 150ms,
then `callBankAPI` gets *its own* fresh 150ms. That's not what a "150ms
budget for the whole payment" should mean at all — under that broken
design, three sequential layers could each individually stay within their
own 150ms and yet the *total* elapsed time balloons to 450ms+, blowing
whatever real deadline actually mattered to the caller (an HTTP request
timeout, a user waiting on a screen). Propagating the *same* context means
the 150ms is a **shared, continuously-draining** budget — by the time
`callBankAPI` runs, it's not getting a fresh 150ms; it's getting whatever's
*left* of the original 150ms after validation and the card-network step
already consumed some of it. This is the entire reason context propagation
matters more than each function just accepting *a* `context.Context`
parameter — it has to be *the same one*, flowing down unmodified (aside from
further derivation, like the request ID here) through the whole chain.

## Try It Yourself
- Add a fourth layer (`notifyCustomer(ctx, txID)`) that runs *after*
  `callBankAPI` succeeds — but only if there's still budget left; check
  `ctx.Err()` before starting it and skip it gracefully if the context is
  already done, logging that the notification was skipped rather than
  failing the whole payment over it
- Change `callBankAPI` to retry once on a transient failure (reusing Module
  08's `retry` package ideas) — but make sure the retry still respects the
  *overall* context deadline, not a fresh timeout of its own
- Add a `context.WithDeadline` variant of `runPayment` that takes an
  absolute `time.Time` instead of a `time.Duration`, useful for "this whole
  batch of payments must finish by 5:00 PM" rather than "each one gets X
  seconds"
