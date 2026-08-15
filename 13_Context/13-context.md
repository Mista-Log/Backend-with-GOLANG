# 13. Context

`context.Context` showed up constantly in Module 12's projects without a
dedicated explanation — every timeout, every cancellation, every
`ctx.Done()` in a `select`. This module is that explanation: what a
`Context` actually carries, the four ways to create one, and the rules that
keep it useful instead of confusing across a real call chain.

---

## `context.Background`

`context.Background()` is the **root** of every context tree — an empty,
never-cancelled, no-deadline, no-values context. Every real context you use
is ultimately derived from one of two roots, and `Background` is the one you
reach for in `main`, at the top of a request handler, or in tests.

```go
ctx := context.Background() // the starting point — never nil, never cancelled
```

```
┌────────────────────────────────────────────────────┐
│   context.Background()                                    │
│        │                                                     │
│        ├── WithCancel(ctx)     → derived, cancellable            │
│        ├── WithTimeout(ctx, d)  → derived, auto-cancels after d      │
│        ├── WithDeadline(ctx, t)  → derived, auto-cancels at time t      │
│        └── WithValue(ctx, k, v)   → derived, carries one extra key/value  │
│                                                                                │
│   Every derived context is a CHILD — cancelling a parent cancels                 │
│   every child (and grandchild) automatically; a child never affects                 │
│   its parent.                                                                          │
└────────────────────────────────────────────────────┘
```

---

## `context.TODO`

`context.TODO()` is **behaviorally identical** to `Background()` — same
empty, uncancellable context — but it exists as a *documentation marker*:
"this function should eventually receive a real context from its caller, but
that plumbing isn't in place yet."

```go
func legacyFunction() {
	ctx := context.TODO() // signals: "this needs a real ctx from the caller
	                        //  eventually — using TODO instead of Background
	                        //  so it's grep-able and obviously provisional"
	doSomething(ctx)
}
```

**Use `Background` when a context genuinely has no parent** (the true root
of a program or request). **Use `TODO` when you're migrating code toward
accepting a context, and don't have a real one to pass yet.** The distinction
is entirely for humans and static analysis tools — `go vet` and some linters
can flag lingering `TODO()` calls as exactly what they say: work still to do.

---

## Cancel

`context.WithCancel` returns a derived context plus a `cancel` function —
calling it cancels the context (closes its `Done()` channel) immediately,
propagating to every child context too.

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel() // ALWAYS defer cancel, even if you expect the context to
                 // finish naturally — see the case study below

go func() {
	select {
	case <-ctx.Done():
		fmt.Println("cancelled:", ctx.Err()) // context.Canceled
	}
}()

time.Sleep(50 * time.Millisecond)
cancel() // triggers the select above
```

```
┌────────────────────────────────────────────────────┐
│   ctx, cancel := context.WithCancel(parent)                │
│                                                                │
│   cancel()  →  closes ctx.Done()'s channel  →  every            │
│                  goroutine select-ing on ctx.Done() unblocks       │
│                  immediately, and ctx.Err() now returns              │
│                  context.Canceled                                       │
└────────────────────────────────────────────────────┘
```

**Why `defer cancel()` even when you expect natural completion:** every
`WithCancel`/`WithTimeout`/`WithDeadline` context holds resources internally
(at minimum, a timer for the deadline-based ones) that only get released
when `cancel` is called — either because the deadline arrived, or because
you called it explicitly. Skipping `defer cancel()` on a context that
finishes "successfully" on its own **still leaks that internal resource**
until the runtime eventually garbage-collects it — `go vet` will actually
warn about a `context.WithCancel`/`WithTimeout` result whose `cancel` is
never called on any path.

---

## Timeout

`context.WithTimeout` is `WithCancel` plus an automatic cancellation after a
**duration** — this is what Module 12's projects used constantly for
per-request and per-operation budgets.

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

select {
case result := <-doWork(ctx):
	fmt.Println("got result:", result)
case <-ctx.Done():
	fmt.Println("timed out:", ctx.Err()) // context.DeadlineExceeded
}
```

```
┌────────────────────────────────────────────────────┐
│   WithTimeout(parent, 2s)                                  │
│        │                                                       │
│        ▼                                                          │
│   context cancels automatically after 2 SECONDS — OR earlier          │
│   if `cancel` is called manually, OR earlier still if the                │
│   PARENT context is cancelled first (whichever happens FIRST wins)          │
└────────────────────────────────────────────────────┘
```

---

## Deadline

`context.WithDeadline` is nearly identical to `WithTimeout`, but takes an
absolute **point in time** instead of a duration — genuinely the same
mechanism (`WithTimeout(ctx, d)` is literally implemented as
`WithDeadline(ctx, time.Now().Add(d))` internally), useful when you're
computing a cutoff from something other than "starting now":

```go
deadline := time.Now().Add(500 * time.Millisecond)
ctx, cancel := context.WithDeadline(context.Background(), deadline)
defer cancel()

// Any context (yours or one you received) can report its own deadline:
if d, ok := ctx.Deadline(); ok {
	fmt.Println("this context expires at:", d)
}
```

Reach for `WithDeadline` specifically when the cutoff is a **shared,
absolute** moment — e.g., "this whole batch job must finish by 5:00 PM,"
computed once and passed down, rather than each downstream call getting its
own relative duration that might not add up to the real overall budget.

---

## Values

`context.WithValue` attaches a single key/value pair to a derived context —
for passing **request-scoped data** through a call chain (a request ID, an
authenticated user, a trace ID) without threading it through every
function's parameter list explicitly.

```go
type contextKey string // a DEDICATED type, not a bare string — see below

const requestIDKey contextKey = "requestID"

ctx := context.WithValue(context.Background(), requestIDKey, "req-abc-123")

func handleRequest(ctx context.Context) {
	if id, ok := ctx.Value(requestIDKey).(string); ok { // type assertion — Module 06
		fmt.Println("handling request:", id)
	}
}
```

```
┌────────────────────────────────────────────────────┐
│   WithValue(parent, key, value)                            │
│        │                                                       │
│        ▼                                                          │
│   Value(key) walks UP the context tree from wherever you're           │
│   calling it, checking each ancestor, until it finds a match             │
│   (or reaches the root and returns nil)                                     │
└────────────────────────────────────────────────────┘
```

**Use context values sparingly, for genuinely request-scoped metadata
only** — request IDs, auth tokens, deadlines/tracing info. **Never** use
`context.Value` to pass ordinary function parameters or optional
configuration — those belong as real, explicit, typed function arguments.
The Go team's own guidance is blunt about this: context values should be
used for data that transits process and API boundaries, not for passing
optional parameters to functions. If a value is required for a function to
do its job correctly, it should be a parameter, full stop — burying it in
`ctx.Value` makes it invisible at the call site and loses all compile-time
type checking.

**Why a dedicated key type (`type contextKey string`), not a bare string:**
context values from *different packages* could otherwise collide if both
happen to use the plain string `"requestID"` as a key — an unexported,
package-local type for the key prevents any other package's `WithValue` call
from accidentally overwriting (or reading) yours, even if they picked the
identical-looking string.

---

## Propagation

The convention across essentially all of Go's ecosystem: **`context.Context`
is the first parameter of any function that might block, do I/O, or need to
be cancelled — named `ctx`, never stored in a struct field.**

```go
func fetchUser(ctx context.Context, id int) (*User, error) { ... } // ✅ idiomatic

type Service struct {
	ctx context.Context // ❌ Go's own documentation explicitly calls this
	                       //     out as wrong — see the case study below
}
```

```
┌────────────────────────────────────────────────────────┐
│   main()                                                       │
│     ctx := context.WithTimeout(Background(), 5*time.Second)        │
│        │                                                              │
│        ▼                                                                │
│   handleRequest(ctx)                                                       │
│        │  (same ctx, or a further-derived child of it)                        │
│        ▼                                                                         │
│   validatePayment(ctx)                                                              │
│        │                                                                               │
│        ▼                                                                                  │
│   chargeCard(ctx)  ──▶  callBankAPI(ctx)                                                     │
│                                                                                                   │
│   The SAME deadline/cancellation signal flows all the way down —                                    │
│   if the top-level 5s timeout fires, EVERY function in this chain                                      │
│   sees ctx.Done() close, all at once, no matter how deep it's nested.                                     │
└────────────────────────────────────────────────────────┘
```

## Case Study: Why Context Never Lives in a Struct Field

Storing `ctx` on a struct looks convenient — no more threading it through
every method signature — but it breaks the entire model this module is
built on: **a `Context` represents the scope of one specific operation or
request**, not the lifetime of an object. A `Service` struct typically
outlives any single request; if it held one `ctx` field, every method call
on it — for every different caller, forever — would share that *one*
context's cancellation and deadline, which is almost never what any
individual caller actually wants. Passing `ctx` explicitly as each method's
first parameter keeps every call's cancellation scope correctly tied to
*that specific call*, which is exactly the shape Module 12's Job Queue and
Concurrent Web Scraper projects both relied on — each worker's `select`
raced its own operation against the context **it was handed for that call**,
not some shared, ambient one.

---

Onto the projects — Payment Timeout builds a multi-layer call chain with a
single top-level timeout propagating all the way down, plus a request ID
carried via context values; API Timeout applies the same propagation
pattern across real HTTP calls, including the specific case of a downstream
call that ignores the incoming deadline (and why that's a bug worth testing
for).
