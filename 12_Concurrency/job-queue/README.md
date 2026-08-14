# Project 2 — Job Queue

```bash
cd job-queue
go run main.go
```

Submits 12 jobs to a 4-worker pool (every 5th job deliberately fails), prints
a mid-flight status snapshot while work is still happening, then shuts down
cleanly and prints final statuses.

## What's Demonstrated Here

- **A worker pool with `select`-based cancellation** — each worker's `select`
  races the next job against `ctx.Done()`, so cancelling the context stops
  workers from picking up *new* jobs immediately, without needing to drain
  whatever's left in the channel first.
- **`WaitGroup` coordinating a clean shutdown** — `Shutdown` closes `jobs`,
  then `wg.Wait()` blocks until every worker has actually returned from its
  loop, and only *then* closes `results` — guaranteeing no worker is still
  trying to send to a channel that's already been closed (which would
  panic).
- **A Mutex-protected `map[int]JobStatus`**, read and written from many
  goroutines safely, with `Status()` proven "live" by the mid-flight
  snapshot showing a genuine mix of pending/running/done statuses rather
  than everything jumping straight to "done."
- **Submitting and collecting concurrently** — the results collector runs in
  its *own* goroutine, reading while `main` is still calling `Submit` in a
  loop, not after — the shape that scales to far more jobs than any
  channel's buffer could hold at once.

```
┌──────────────────────────────────────────────────────────┐
│   Submit(job)  →  setStatus(Pending)  →  jobs <- job              │
│                                                                       │
│   worker:  select {                                                      │
│                case job := <-jobs:  setStatus(Running) → run → setStatus(Done/Failed)│
│                case <-ctx.Done():   return immediately                        │
│            }                                                                       │
│                                                                                          │
│   Shutdown():  close(jobs) → wg.Wait() → close(results)                                    │
└──────────────────────────────────────────────────────────┘
```

## Case Study: Why the Status Map Uses `Mutex`, Not `sync.Map`

This looks, on the surface, like exactly the case `sync.Map` is for — many
goroutines hitting the same map concurrently. The difference is **what kind
of updates happen**: `setStatus` is a single, independent write per call
(genuinely `sync.Map`-friendly), but a *slightly* more featureful version of
this queue — say, one that also wanted to atomically check "is this job
already Done before marking it Failed?" — would need to read-then-write as
ONE indivisible operation, which `sync.Map`'s API doesn't cleanly support
for arbitrary custom logic (it only atomizes a few specific operations like
`LoadOrStore`). A `Mutex` guarding a plain `map[int]JobStatus` stays fully
general: any future logic needing multiple related checks/updates together
just goes inside the same `Lock()`/`Unlock()` pair, with the compiler still
giving full type safety (`JobStatus`, not `any`) the whole way. Compare this
against the Concurrent Web Scraper project's `visited sync.Map` — that one
only ever needed ONE atomic operation (`LoadOrStore`), which is exactly
`sync.Map`'s sweet spot.

## Try It Yourself
- Call `cancel()` (from `context.WithCancel`) partway through submission —
  e.g., after submitting 5 of the 12 jobs — and observe which jobs end up
  `Done`/`Failed` vs. stuck at `Pending` forever (a real consequence of
  cancellation worth seeing directly)
- Add a `Progress() (pending, running, done, failed int)` method that
  locks once and counts every status in one pass — a natural extension now
  that the map is Mutex-protected
- Swap `Job.Task`'s signature to accept a `context.Context` parameter too,
  so long-running jobs can check for cancellation *mid-task*, not just
  between jobs — a meaningfully harder problem than what this project solves
