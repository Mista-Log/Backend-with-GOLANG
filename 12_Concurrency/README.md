# Go for Beginners — Module 12: Concurrency (Most Important)

## Contents

1. **[12-concurrency.md](./12-concurrency.md)** — Goroutines, channels
   (buffered/unbuffered), `WaitGroup`, `Mutex`/`RWMutex`, `sync/atomic`,
   `sync.Once`, `sync.Map`, `sync.Cond`, `select` (including the
   `time.After` timeout and `context.Done()` cancellation patterns), worker
   pools, pipelines, and fan-out/fan-in — then an **Advanced** section on
   race conditions (with the `-race` detector), deadlocks (the classic
   two-mutex case, and the "consistent lock ordering" fix), Go's memory
   model (the happens-before guarantees channels and `sync` actually
   provide), and the M:N goroutine scheduler. This is the longest guide in
   the series — diagrams throughout every section.

2. **[concurrent-web-scraper/](./concurrent-web-scraper)** — A worker pool
   (fan-out) scraping URLs against a local, offline `httptest` server, with
   results fanned back **in** to one channel, `sync.Map` for safe
   concurrent deduplication (`LoadOrStore`, with a case study on exactly why
   it has to be one atomic call), and two nested layers of `context` timeout.

3. **[job-queue/](./job-queue)** — A worker pool tracking per-job status in a
   **Mutex**-protected map (not `sync.Map` — the README explains exactly
   why), `select`-based cancellation racing the next job against
   `ctx.Done()`, and a `WaitGroup`-coordinated clean shutdown that guarantees
   no worker sends on an already-closed channel.

4. **[email-worker/](./email-worker)** — `sync.Once` for a shared, lazily
   built client (proven single-instance via a shared connection ID),
   `atomic.Int64` counters under heavy concurrent access, a buffered-channel
   semaphore for rate-limiting concurrency, and an `RWMutex`-protected config
   changed **live**, mid-run.

5. **[image-processor/](./image-processor)** — A genuine 3-stage pipeline
   (load → resize → save) where the middle, CPU-bound stage is *itself* a
   fan-out/fan-in across 3 workers — fan-out/fan-in nested inside a pipeline
   stage, not just a standalone pattern. Cancellation via `select` at every
   channel send, and a `sync.Cond`-based bounded log demonstrating real,
   visible backpressure.

## Suggested Order

```
Concurrency guide (core + Advanced)
        │
        ▼
Concurrent Web Scraper ──▶ Job Queue ──▶ Email Worker ──▶ Image Processor
  (fan-out/fan-in,           (worker pool,    (sync.Once,        (full pipeline,
   sync.Map, context)         Mutex map,       atomics,           fan-out/fan-in
                               select cancel)   RWMutex,           AS a stage,
                                                 semaphore)         sync.Cond)
```

Each project deliberately uses a *different* combination of the guide's
primitives, so by the end you've seen every one of them applied in a real,
runnable program — not just described in isolation.

## Quick Reference: Running Any Project

```bash
cd <project-folder>
go run main.go

# ALWAYS worth running concurrent code through the race detector:
go run -race main.go
```

All four projects are fully self-contained and offline — no real network,
no external services, no setup beyond `go run`.

*Note: this module builds on Modules 00–11 — start there first if you
haven't already. This is the densest module in the series; if any single
project's code is hard to follow in one pass, re-read its section of the
guide alongside it rather than pushing through — concurrency bugs are
genuinely subtle, and it's worth being deliberate here.*
