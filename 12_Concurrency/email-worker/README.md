# Project 3 — Email Worker

```bash
cd email-worker
go run main.go
```

Sends 25 simulated emails, at most 4 concurrently, with retries on
transient failures — and tightens the retry policy live, partway through.

## What's Demonstrated Here

- **`sync.Once` for a shared, lazily-created client** — every sending
  goroutine calls `getClient()`, but "establishing connection..." prints
  exactly once. The final line confirms it directly: every send used the
  *same* `client.connectionID`.
- **A buffered channel as a semaphore** — `semaphore := make(chan struct{}, 4)`;
  acquiring is `semaphore <- struct{}{}` (blocks once 4 are already
  in-flight), releasing is `<-semaphore` (deferred, so it always runs). This
  is a genuinely common Go idiom for "limit concurrency to N" that doesn't
  need a worker-pool structure at all — just a channel used purely for its
  blocking behavior, not to carry any real data (`struct{}` takes zero
  bytes, which is why it's the conventional choice for a "just a signal"
  channel).
- **Atomic counters (`Stats`)** — `sent`, `failed`, and `retried` are all
  `atomic.Int64`, updated from up to 4 concurrent goroutines with no mutex
  needed, because each is a single independent counter (exactly the
  "atomics for simple counters" guidance from the guide).
- **`RWMutex`-protected, *live* config** — `cfg.MaxRetries()` is called by
  every send (many concurrent readers, via `RLock`), while `SetMaxRetries`
  is called once, mid-run, needing the exclusive `Lock`. Sends already in
  flight when the change happens keep using whatever they'd already read;
  every *subsequent* call to `MaxRetries()` sees the new value immediately.

```
┌──────────────────────────────────────────────────────────┐
│   25 goroutines, one per recipient                                 │
│        │                                                              │
│        ▼                                                                │
│   semaphore <- struct{}{}   ◀── blocks once 4 are already in flight       │
│        │                                                                     │
│        ▼                                                                       │
│   getClient()   ◀── sync.Once: real work happens on the FIRST call only,          │
│        │             every other call just returns the already-built client          │
│        ▼                                                                                 │
│   sendWithRetry()   ◀── reads cfg.MaxRetries() (RLock) fresh each call                      │
│        │                                                                                       │
│        ▼                                                                                          │
│   stats.sent/failed/retried.Add(1)   ◀── atomic, no lock needed                                      │
│        │                                                                                                 │
│        ▼                                                                                                    │
│   <-semaphore   ◀── release (deferred — always runs)                                                          │
└──────────────────────────────────────────────────────────┘
```

## Case Study: Why the Semaphore Uses `struct{}`, Not `bool` or `int`

```go
semaphore := make(chan struct{}, maxConcurrentSends)
semaphore <- struct{}{} // acquire
<-semaphore              // release
```

The channel's *value* is never actually used or inspected anywhere — only
its **blocking behavior** (send blocks when the buffer is full; receive
unblocks a waiting sender) matters. `struct{}{}` is Go's idiomatic
"zero-information, zero-size value" — it occupies no memory at all, unlike
even a single `bool`. Using `chan bool` or `chan int` here would work
identically in practice, but `chan struct{}` communicates the *intent*
clearly to anyone reading the code: "this channel carries no data, it's
purely a synchronization signal" — the same spirit as an unbuffered channel
used purely for its rendezvous behavior (Section 3 of the guide), just with
a capacity this time.

## Try It Yourself
- Change `maxConcurrentSends` to `1` and to `25`, and watch total run time
  change — with `25`, the semaphore never actually blocks anyone, which is a
  good way to see when a "rate limiter" set too high stops doing anything
  useful at all
- Add a `Stats.RetriedByAttempt() map[int]int64` — you'll find this is
  *exactly* the kind of "more than one related value updated together" case
  the guide says atomics alone can't cleanly handle, and that a `Mutex`
  around a plain map is the better fit (same reasoning as Job Queue's status
  map)
- Run with `go run -race main.go` — should be clean. Then swap
  `stats.sent.Add(1)` for a plain `stats.sentPlain++` on a new, non-atomic
  `int` field instead, called from the same concurrent sends, and run
  `-race` again to see it flagged
