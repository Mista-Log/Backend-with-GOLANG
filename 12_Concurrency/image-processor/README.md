# Project 4 — Image Processor

```bash
cd image-processor
go run main.go
```

Generates 10 stand-in "image" files, runs them through a 3-stage pipeline,
and prints progress messages as a background drainer periodically empties a
capacity-4 log — watch for occasional pauses where a stage is visibly
waiting for log room to free up.

## What's Demonstrated Here

- **A real 3-stage pipeline** — `loadStage → resizeStage → saveStage`,
  chained by plain channels, each stage its own goroutine(s), all running
  concurrently: while `saveStage` writes image #1, `resizeStage` can already
  be working on image #3.
- **Fan-out/fan-in *embedded inside one pipeline stage*** — `resizeStage`
  isn't a single goroutine; it's 3 worker goroutines all reading the *same*
  input channel (fan-out) and writing to the *same* output channel (fan-in),
  because resizing is the CPU-bound stage that benefits most from
  parallelism, while loading and saving stay single-goroutine stages.
- **`select` for cancellation at every send** — every stage's send to its
  output channel is wrapped in a `select` racing against `ctx.Done()`, so a
  cancelled/timed-out pipeline unwinds promptly instead of leaving a
  goroutine blocked forever trying to send into a channel nobody's reading
  anymore.
- **`sync.Cond`-based backpressure, actually visible** — `BoundedLog`'s
  capacity is deliberately tiny (4), so `Push` genuinely blocks sometimes,
  slowing down whichever stage tried to log while the log is full, until the
  drainer's next tick calls `DrainAll` and `Broadcast`s room being free
  again.

```
┌──────────────────────────────────────────────────────────┐
│   loadStage  ──▶  resizeStage (3 workers, fan-out/fan-in)  ──▶  saveStage   │
│       │                    │                                       │           │
│       └──────────┬─────────┴───────────────────┬───────────────────┘           │
│                   ▼                             ▼                                 │
│              BoundedLog.Push(...)          BoundedLog.Push(...)                       │
│                   │                             │                                        │
│                   └──────────────┬──────────────┘                                           │
│                                   ▼                                                            │
│                          drainer goroutine: every 15ms, DrainAll() + print                        │
└──────────────────────────────────────────────────────────┘
```

## Case Study: Why the Log Uses `sync.Cond` Instead of Just a Buffered Channel

A buffered `chan string` would honestly work fine for most of what
`BoundedLog` does here — but it's worth understanding exactly what
`sync.Cond` adds that a channel alone doesn't: `BoundedLog.DrainAll` removes
**everything currently queued in one batch call**, not one item at a time.
A channel's receive only ever gives you one value per receive; getting "all
currently-buffered items, right now, as a slice" out of a channel needs an
awkward non-blocking drain loop (`select { case v := <-ch: ... default: return }`)
that a `sync.Cond`-protected slice doesn't need at all — `DrainAll` just
locks, swaps the slice, unlocks. This is exactly the kind of situation the
guide flags as `sync.Cond`'s actual niche: **when the condition you're
waiting on isn't naturally "exactly one value arrived,"** but something
broader ("there's room now," which could be satisfied by draining any
number of items at once).

## Try It Yourself
- Lower the pipeline's overall timeout (`context.WithTimeout`) to something
  short enough to actually trigger mid-run, and observe every stage stop
  promptly via its `select`/`ctx.Done()` — count how many of the 10 images
  made it all the way through before the cutoff
- Change `resizeStage`'s worker count from 3 to 1, and time the whole run
  both ways — a direct, visible demonstration of fan-out's benefit for a
  genuinely CPU-bound stage
- Increase `BoundedLog`'s capacity from 4 to 100 and re-run — the pipeline
  should finish visibly faster, since stages rarely (if ever) block on
  `Push` waiting for log room; a good way to see backpressure's real cost
  when it's actually engaged vs. when it isn't
