# Go Internals — Interview Reference

The five topics interviewers use to separate "writes Go" from
"understands Go." Each answer below is the depth a strong candidate gives.

---

## Memory

**Q: Stack or heap — who decides, and how?**

The **compiler**, via escape analysis (Modules 01/05/23). If a value's
address can outlive its function, it's moved to the heap; otherwise it
stays on the stack and costs essentially nothing to free.

```bash
go build -gcflags="-m" ./...   # shows every decision
```

**Q: Why is returning a pointer to a local variable safe in Go but a bug
in C?**

Because Go's escape analysis *notices* and heap-allocates that variable
instead. In C, the stack frame is gone and the pointer dangles.

**Q: How does Go's GC work?**

Concurrent tri-color mark-and-sweep. Mostly runs alongside your program;
`GOGC` controls how much the heap grows before a cycle triggers
(default 100 = trigger when the heap doubles). `GOMEMLIMIT` (1.19+) sets a
soft ceiling. **The practical lever is reducing allocations, not tuning
GOGC** (Module 23).

---

## Channels

**Q: What IS a channel, internally?**

An `hchan` struct containing: a circular buffer (for buffered channels),
a send queue and receive queue of waiting goroutines, and a mutex. Sends
and receives take that lock briefly — a channel is *not* lock-free, it's a
well-designed wrapper around a lock plus goroutine parking.

**Q: Behavior table — memorize this:**

```
┌──────────────────────────────────────────────────────────┐
│   Operation          nil channel      closed channel     empty/full     │
│  ────────────────────────────────────────────────────────  │
│   send                blocks forever   PANIC               blocks if full   │
│   receive              blocks forever   zero value, ok=false  blocks if empty   │
│   close                 PANIC            PANIC                 —                  │
└──────────────────────────────────────────────────────────┘
```

**Q: Who should close a channel?**

The **sender**, always — and only if there's exactly one, or they
coordinate (e.g. a `sync.WaitGroup` then one close, Module 12's fan-in).
A receiver closing risks a send-on-closed panic.

---

## Interfaces

**Q: How is an interface value represented?**

Two words: a pointer to a **type descriptor** (itab, holding the method
set) and a pointer to the **data**. This is why `interface{}` boxing
forces a heap allocation for values that would otherwise stay on the
stack (Module 23).

**Q: The nil interface trap — why isn't this nil?**

```go
var p *MyError = nil
var err error = p
fmt.Println(err == nil) // FALSE — this surprises almost everyone
```

Because the interface value holds a **type** (`*MyError`) even though its
**data** pointer is nil. An interface is nil only when BOTH words are
nil. This is why a function should return `nil` explicitly rather than a
nil-valued concrete pointer:

```go
// ❌ callers checking `if err != nil` get TRUE even on success
func doThing() error {
    var e *MyError
    if failed { e = &MyError{} }
    return e
}

// ✅
func doThing() error {
    if failed { return &MyError{} }
    return nil
}
```

**Q: Value vs. pointer receivers and interface satisfaction?**

If a method has a pointer receiver, only `*T` satisfies the interface —
not `T` (Module 06). A method with a value receiver is in both method
sets.

---

## Maps

**Q: How is a Go map implemented?**

A hash table of **buckets**, each holding up to 8 key/value pairs. Keys
hash to a bucket; overflow buckets chain when one fills. Growth
(doubling) happens **incrementally** — a few buckets are migrated per
operation, not all at once, to avoid a long pause.

**Q: Why is map iteration order randomized?**

Deliberately, so nobody writes code depending on an order the spec never
promised. Randomized on *every* range, from a random starting bucket
(Module 04).

**Q: Are maps concurrency-safe?**

**No.** Concurrent read+write panics with an explicit "concurrent map
writes" runtime error (Go actively detects this). Use a `sync.RWMutex` or
`sync.Map` (Module 12).

**Q: Can you take the address of a map element?**

No — `&m[key]` doesn't compile, because growth can move elements. Store
pointers as values (`map[K]*V`) if you need stable references (Module 04's
Inventory System did exactly this).

---

## Slices

**Q: What is a slice, exactly?**

A three-word header: **pointer** to a backing array, **len**, **cap**.
Passing a slice copies the *header*, not the data — which is why a
function can modify elements the caller sees, but `append`ing inside a
function may or may not be visible (Module 04).

**Q: The aliasing trap:**

```go
a := []int{1, 2, 3, 4, 5}
b := a[1:3]        // len 2, cap 4 — shares a's backing array
b = append(b, 99)  // fits within cap → OVERWRITES a[3]!
fmt.Println(a)     // [1 2 3 99 5]
```

Use a **full slice expression** to prevent this: `a[1:3:3]` caps it, so
append must reallocate.

**Q: How does append grow?**

Roughly doubles for small slices, then grows by a smaller factor (~1.25x)
for large ones. Each growth allocates a new array and copies — which is
why pre-sizing with `make([]T, 0, n)` matters in hot paths (Module 23).

**Q: `nil` slice vs. empty slice?**

```go
var a []int          // nil: a == nil is true, len 0, marshals to JSON `null`
b := []int{}         // empty: b == nil is FALSE, len 0, marshals to `[]`
```
Both are safe to `append` to and `range` over. The JSON difference is the
one that bites in real APIs.
