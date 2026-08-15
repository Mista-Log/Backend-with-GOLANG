# Project 2 — API Timeout

```bash
cd api-timeout
go run main.go
```

Three real HTTP hops, all local (`httptest`, no real network): **client →
gateway → downstream**. Both gateway endpoints call the same slow (300ms)
downstream service; the client gives up after only 100ms on both. What
differs is what the **gateway** does next.

## What's Demonstrated Here

- **Three-hop propagation, correctly done (`/forecast`)** — the gateway
  handler derives its context with `context.WithTimeout(r.Context(), ...)`.
  Because it's derived *from* `r.Context()`, when the client disconnects
  (its own 100ms budget expires), `net/http` cancels `r.Context()`
  automatically — and that cancellation flows straight through to the
  gateway's outbound call to downstream, and from there to the downstream
  handler's own `select` on *its* `r.Context()`. One client giving up
  cancels work at every hop, immediately.
- **The same chain, broken (`/forecast-buggy`)** — the only difference is
  `ctx := context.Background()` instead of deriving from `r.Context()`. This
  single line completely severs the chain: the client giving up has *zero*
  effect on this handler, which keeps running — and keeps its outbound
  connection to downstream open — for the entire 300ms, producing a response
  that gets thrown away because nothing is listening for it anymore.
- **`sync.WaitGroup` proving the server-side difference empirically** —
  `main` doesn't just claim the buggy handler keeps running; it actually
  waits (`handlerWG.Wait()`) for both server-side handlers to fully finish
  and prints their real completion times, so the wasted ~200ms is directly
  visible in the output, not just asserted.

```
┌──────────────────────────────────────────────────────────┐
│   CORRECT:  client ──100ms──✗  (gives up)                        │
│               │                                                     │
│               ▼ (r.Context() cancelled)                               │
│            gateway ──✗ (aborts almost immediately, ~100ms total)         │
│               │                                                             │
│               ▼ (outbound ctx cancelled too)                                  │
│            downstream ──✗ (select notices, stops, ~100ms total)                  │
│                                                                                       │
│   BUGGY:    client ──100ms──✗  (gives up)                                              │
│               │                                                                           │
│               ╳  (context.Background() — NO connection to r.Context() at all)               │
│               ▼                                                                                │
│            gateway keeps running... downstream keeps running...                                  │
│               ▼                                                                                      │
│            BOTH finish naturally at ~300ms — 200ms of pure waste                                        │
└──────────────────────────────────────────────────────────┘
```

## Case Study: This Bug Is Everywhere, and It's Subtle

`context.Background()` used for an outbound call inside an HTTP handler
*looks* completely reasonable — it compiles, it runs, the response even
comes back correctly under normal conditions (when the client doesn't give
up early, both `/forecast` and `/forecast-buggy` behave identically). The
bug only shows up under exactly the conditions that matter most in
production: slow downstream services combined with clients that time out or
disconnect — precisely when you most need the server to stop doing wasted
work, not keep grinding away on a request nobody will ever read the
response to. At real scale, this exact bug is a genuine, well-known source
of **goroutine leaks and connection pool exhaustion**: a service under load,
serving slow requests to impatient clients, where every abandoned request
keeps consuming a goroutine and an outbound connection for its *full*
original duration instead of being cancelled — the server's actual load
under stress ends up far higher than its *useful* work.

## Try It Yourself
- Add a third gateway endpoint that's buggy in a *different*, subtler way:
  it correctly derives from `r.Context()`, but wraps it in its own
  `context.WithTimeout` using a duration *longer* than the client could
  realistically wait — technically "propagating," but still wasteful. Think
  through why deriving from the parent isn't sufficient on its own; the
  *value* you choose for any additional timeout matters too
- Change the downstream handler to NOT select on `r.Context().Done()` at
  all (just `time.Sleep(300 * time.Millisecond)` unconditionally) and rerun
  — confirm the *gateway's* good handler still aborts its own wait
  correctly, but the downstream server itself now wastes the full 300ms
  regardless, since propagation has to be honored at **every** hop to fully
  pay off, not just the first one
- Measure actual resource usage (e.g., with `runtime.NumGoroutine()` sampled
  right after both client calls return, before `handlerWG.Wait()`) to see
  the leaked goroutine directly, not just its effect on timing
