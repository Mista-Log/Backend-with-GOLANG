# Project 2 — Calculator API

Where the Math Library project used functions internally, this one exposes
them over **HTTP** — and in doing so, leans on higher-order functions even
harder, since Go's whole `net/http` model is built on "a handler is just a
function."

```bash
cd calculator-api
go run .
```

Then, in another terminal:
```bash
curl "http://localhost:8080/calculate?op=add&a=4&b=5"
# {"result":9}

curl "http://localhost:8080/calculate?op=div&a=4&b=0"
# {"error":"division by zero"}

curl "http://localhost:8080/sum?n=1&n=2&n=3&n=4"
# {"result":10}

curl "http://localhost:8080/power?base=2&exp=10"
# {"result":1024}

curl -i "http://localhost:8080/calculate?op=add&a=1&b=1" | grep X-Request-Count
# X-Request-Count: 1   (increments on every request, across ALL routes)
```

## What Each Piece Demonstrates

| Piece | Concept |
|---|---|
| `operations` map | A **dispatch table** — functions stored as values, looked up by name instead of a growing `switch` |
| `withLogging` | A **higher-order function**: takes a handler, returns a new one that wraps it |
| `withRequestCount` | A **closure**-producing middleware — `count` is captured once and persists across every request that flows through the wrapped handler |
| `sumOf` | **Variadic** — the same shape as `mathlib.Sum`, reused in an HTTP context |
| `power` | **Recursion** — `base^exp = base * base^(exp-1)`, same shape as `Factorial` in the Math Library |

## Architecture: Composing Middleware

```
┌──────────────────────────────────────────────────────────────┐
│  mux.HandleFunc("/calculate",                                   │
│      withLogging(              ◀── outermost: logs AFTER the      │
│          withRequestCount(          inner handler finishes          │
│              calculateHandler   ◀── innermost: the actual logic       │
│          )                                                              │
│      )                                                                    │
│  )                                                                          │
│                                                                                │
│  Request flow:                                                                  │
│  incoming request                                                                 │
│       │                                                                             │
│       ▼                                                                              │
│  withLogging's returned func runs                                                      │
│       │  (records start time)                                                            │
│       ▼                                                                                    │
│  withRequestCount's returned func runs                                                       │
│       │  (increments count, sets response header)                                              │
│       ▼                                                                                          │
│  calculateHandler runs (the real work)                                                             │
│       │                                                                                               │
│       ▼                                                                                                 │
│  back up through withRequestCount, then withLogging                                                        │
│  (which logs total elapsed time on the way back out)                                                         │
└──────────────────────────────────────────────────────────────┘
```

Each middleware function is itself a higher-order function: it receives a
`http.HandlerFunc` and returns a *new* one. Wrapping a handler in two of them,
`withLogging(withRequestCount(calculateHandler))`, is exactly the closure
composition idea from the guide's `multiplier`/`double`/`triple` example,
just applied to request handling instead of arithmetic.

## Case Study: Why `withRequestCount` Needs a Mutex

`net/http`'s server handles every incoming request on its **own goroutine**,
running concurrently. If two requests hit the server at almost the same
instant, both goroutines could read `count`, increment their own copy, and
write back — losing one of the increments (a classic **race condition**).
`sync.Mutex` ensures only one goroutine can execute the
`count++` / read-current-value section at a time. This project doesn't get
into goroutines and channels in depth yet — that's a later module — but it's
worth noticing now: **any time a closure captures mutable state and might be
called concurrently, that state needs protection.**

## Try It Yourself
- Add a `/factorial?n=<int>` endpoint reusing the recursive style from `power`
- Add a `withRateLimit` middleware (another higher-order function) that
  rejects requests beyond N per minute — you'll want another closure-captured
  counter, reset on a timer
- Change `operations` to also support a variadic `/calculate-many?op=add&n=1&n=2&n=3`
  that folds the whole list through the dispatch table with `Reduce`-style logic
