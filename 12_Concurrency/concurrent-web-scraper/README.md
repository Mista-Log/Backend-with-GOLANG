# Project 1 — Concurrent Web Scraper

```bash
cd concurrent-web-scraper
go run main.go
```

Runs entirely offline against a local `httptest` server. One URL is
deliberately slow, one deliberately errors, one basically never responds,
and one is a duplicate — so every interesting code path actually fires.

## What's Demonstrated Here

- **Fan-out**: 3 `worker` goroutines all read from the same `urlCh` — at
  most 3 requests are in flight at once, no matter how many URLs there are.
- **Fan-in**: every worker writes to the *same* `resultsCh` — the merge is
  implicit (there's nothing extra to "merge," since they all target one
  channel), and a separate goroutine's `workerWG.Wait()` + `close(resultsCh)`
  is what makes `main`'s `for range resultsCh` terminate correctly once
  every worker is truly done.
- **`sync.Map` for concurrent deduplication** — `visited.LoadOrStore(url, true)`
  is one atomic operation: "check AND mark as seen," indivisibly. Using a
  plain `map` here without a mutex around this exact check-then-set would be
  a textbook data race (two workers could both see "not visited yet"
  simultaneously and both scrape the same URL).
- **Two layers of `context` timeout** — an overall 3-second budget for the
  whole scrape (`main`'s `ctx`), and a *separate*, tighter 150ms budget per
  individual request (`scrapeOne`'s `reqCtx`). `/slow` (300ms) and
  `/timeout` (5s) both get cut off by the per-request timeout specifically,
  without threatening the other URLs' chances to finish within the overall
  budget.

```
┌──────────────────────────────────────────────────────────┐
│   urlCh:  page1, page2, page1(dup), page3, slow, broken, timeout, page4   │
│              │                                                              │
│      ┌───────┼───────┐                                                        │
│      ▼       ▼       ▼                                                          │
│   worker1  worker2  worker3     ◀── FAN-OUT (shared urlCh)                         │
│      │       │       │                                                                │
│      └───────┼───────┘                                                                  │
│              ▼                                                                             │
│         resultsCh          ◀── FAN-IN (shared resultsCh — implicit merge)                     │
│              │                                                                                    │
│         main's range loop prints each result as it arrives                                          │
└──────────────────────────────────────────────────────────┘
```

## Case Study: Why `LoadOrStore`, Not `Load` Then `Store`

The tempting-but-broken version:
```go
if _, ok := visited.Load(url); !ok {  // ❌ TWO separate operations
	visited.Store(url, true)             // — a race lives in the GAP between them
	results <- scrapeOne(ctx, client, url)
}
```
Between the `Load` returning "not found" and the `Store` actually happening,
**another goroutine could run the exact same check** and also conclude the
URL hasn't been visited yet — both workers would then scrape the duplicate
URL, defeating the whole point of deduplication. `LoadOrStore` closes that
gap by doing both steps as one indivisible operation: it stores the value
if the key is absent, and *atomically* tells you whether it *was* already
present — there's no window for another goroutine to interleave in the
middle. This is the exact same "check-then-act needs to be one atomic step"
lesson as `Cache.SetIfAbsent` from Module 07's "Try It Yourself" section,
now actually implemented.

## Try It Yourself
- Lower `numWorkers` to 1 and observe the total run time change — with one
  worker, everything is fully serial (still concurrent-*shaped* code, but no
  parallel benefit, connecting back to the guide's concurrency-vs-parallelism
  note)
- Add a retry for failed requests using Module 08's `retry` package
  concepts (or the package itself, if you still have it) — but stop retrying
  once the *overall* context is nearly expired
- Run this with `go run -race main.go` — it should report clean. Then
  deliberately swap `LoadOrStore` for the broken `Load`-then-`Store` version
  above and run `-race` again to see it flagged (note: `-race` catches
  *some* race window hits probabilistically, not with 100% certainty on
  every single run — that unreliability is exactly why races are so
  dangerous, and why running `-race` routinely matters more than running it
  once)
