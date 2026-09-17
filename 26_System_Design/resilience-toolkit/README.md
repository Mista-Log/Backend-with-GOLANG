# Resilience Toolkit

Four small, genuinely reusable packages implementing the guide's Circuit
Breaker, Rate Limiter, Retry, and Backpressure sections.

```bash
cd resilience-toolkit
go run ./cmd/demo
```

## What Each Section Shows

- **Circuit Breaker**: 3 failures trip it OPEN — calls 4 and 5 fail
  instantly with `ErrOpen`, never actually invoking the failing function.
  After the reset timeout, one HALF-OPEN test call succeeds and the
  breaker fully recovers to CLOSED.
- **Rate Limiter**: a bucket of capacity 3 allows the first 3 requests,
  rejects the next 2 (no time to refill yet at 1 token/sec).
- **Retry**: fails twice, succeeds on the third attempt, with the delay
  between attempts doubling each time.
- **Backpressure**: 2 workers, a queue of depth 2 — the first 4 job
  submissions return instantly (2 running + 2 queued), but submissions 5
  and 6 **block** until a worker frees up. Watch the "submitting job N at
  ..." timestamps: jobs 5-6 print noticeably later than 1-4, proving the
  block is real, not simulated.

## Try It Yourself
- Wrap `retry.Do` around a call that's ALSO guarded by a `circuitbreaker.Breaker`
  — exactly the guide's "complementary, not competing" pairing — and watch
  the breaker eventually stop the retries from even attempting once it opens
- Add jitter to `retry.Do`'s backoff (Module 08's "Try It Yourself" suggestion)
- Change `backpressure.Pool` to reject (instead of block) once the queue is
  full, returning `false` from `Submit` — a "shed load" policy instead of a
  "slow down" one, and think through which real scenarios want which
