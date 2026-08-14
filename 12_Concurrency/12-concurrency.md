# 12. Concurrency (Most Important)

This is the module Go is famous for. Everything before this has been "how to
write correct Go" — this module is "how to write Go that does many things at
once, correctly." Take it slower than the others; the bugs here (races,
deadlocks) often don't show up every run, which makes them the most
dangerous class of mistake in the whole language.

---

## 1. Goroutines

A goroutine is a function running **concurrently** with the rest of your
program — started with the `go` keyword, unbelievably cheap (a few KB of
stack, growable), and scheduled by Go's own runtime (Module 01), not the OS
directly.

```go
func sayHello() {
	fmt.Println("hello from a goroutine")
}

func main() {
	go sayHello() // starts running CONCURRENTLY, doesn't block main
	fmt.Println("hello from main")
	time.Sleep(100 * time.Millisecond) // crude — just to let sayHello finish
}
```

```
┌────────────────────────────────────────────────────┐
│   main()  ──────────────────────────────────────▶  ends       │
│     │                                                             │
│     └── go sayHello()  ──────▶  runs CONCURRENTLY, on its own          │
│                                    goroutine, interleaved with            │
│                                    main() by the Go scheduler                │
└────────────────────────────────────────────────────┘
```

**The `time.Sleep` above is a crude hack, not a real solution** — if `main`
returns before the goroutine finishes, the goroutine is simply killed
mid-flight; Go does **not** wait for background goroutines automatically.
The rest of this module is largely about *real* ways to coordinate goroutine
completion (`WaitGroup`, channels) instead of guessing at sleep durations.

**Closures over loop variables** (Module 03) matter enormously here — Go
1.22+ scopes `for` loop variables per-iteration, so this is safe:
```go
for i := 0; i < 3; i++ {
	go func() {
		fmt.Println(i) // Go 1.22+: correctly prints 0, 1, 2 (in some order)
	}()
}
```

---

## 2. Channels

A channel is a **typed pipe** goroutines use to send and receive values
safely — Go's primary answer to "how do goroutines communicate," embodying
the language's famous proverb: *"Don't communicate by sharing memory; share
memory by communicating."*

```go
ch := make(chan int)  // an unbuffered channel of int

go func() {
	ch <- 42 // SEND — blocks until someone receives
}()

value := <-ch // RECEIVE — blocks until someone sends
fmt.Println(value) // 42
```

```
┌────────────────────────────────────────────────────┐
│   ch <- 42     "send 42 into ch"                          │
│   v := <-ch     "receive a value from ch, assign to v"        │
│                                                                    │
│   A channel is a TYPE: chan int, chan string, chan MyStruct...       │
│   Only values of that type can travel through it.                       │
└────────────────────────────────────────────────────┘
```

Closing a channel signals **"no more values are coming"** — a receiver can
detect this with the two-value receive form:
```go
close(ch)
v, ok := <-ch // ok is false once the channel is closed AND drained
```
`range` over a channel automatically stops when it's closed:
```go
for v := range ch {
	fmt.Println(v) // loops until ch is closed
}
```
**Only the sender should close a channel, never the receiver** — sending on
a closed channel panics, and closing an already-closed channel also panics.

---

## 3. Unbuffered Channels

An unbuffered channel (`make(chan int)`, no capacity) has **zero storage** —
a send blocks until a receiver is ready, and a receive blocks until a sender
is ready. This is a **synchronization point**, not just a queue: the
send/receive handshake is how the two goroutines know they've "met."

```
┌────────────────────────────────────────────────────┐
│   Goroutine A:  ch <- 1         (BLOCKS here)              │
│   Goroutine B:      <-ch         (BLOCKS here, until A sends)  │
│                                                                    │
│   The moment BOTH are ready, the value transfers and BOTH             │
│   goroutines unblock at the same instant — a rendezvous.                 │
└────────────────────────────────────────────────────┘
```

---

## 4. Buffered Channels

A buffered channel (`make(chan int, 5)`) has capacity — a send only blocks
once the buffer is **full**; a receive only blocks once the buffer is
**empty**.

```go
ch := make(chan int, 3)
ch <- 1 // doesn't block — buffer has room
ch <- 2 // doesn't block
ch <- 3 // doesn't block — buffer now FULL
ch <- 4 // BLOCKS — no room until something receives
```

```
┌────────────────────────────────────────────────────┐
│   Unbuffered (cap 0):  send blocks until a receiver is READY,    │
│                          every single time — synchronous hand-off      │
│                                                                              │
│   Buffered (cap N):     send only blocks once N items are already          │
│                          waiting to be received — asynchronous UP TO           │
│                          the buffer size, then synchronous                        │
└────────────────────────────────────────────────────┘
```

A common, useful buffer size is exactly the number of items you know you'll
send, so producers never block at all — or `1`, for a "signal" channel where
you only care that *something* happened, not queuing multiple signals.

---

## 5. WaitGroup

`sync.WaitGroup` is the correct replacement for the `time.Sleep` hack in
Section 1 — it lets `main` (or any goroutine) wait for a known number of
other goroutines to finish.

```go
var wg sync.WaitGroup

for i := 0; i < 3; i++ {
	wg.Add(1) // increment the counter BEFORE starting the goroutine
	go func(id int) {
		defer wg.Done() // decrement when this goroutine finishes — Module 02's defer
		fmt.Println("worker", id, "done")
	}(i)
}

wg.Wait() // blocks until the counter reaches zero
fmt.Println("all workers finished")
```

```
┌────────────────────────────────────────────────────┐
│   wg.Add(1)  →  counter++            (call BEFORE `go`)        │
│   wg.Done()  →  counter--             (deferred, inside the        │
│                                          goroutine)                     │
│   wg.Wait()   →  blocks until counter == 0                                │
│                                                                                │
│   Add MUST happen before the goroutine starts — Add-inside-the-                  │
│   goroutine is a classic race: main might reach Wait() before the                   │
│   Add() even runs, and Wait() returns immediately, wrongly.                            │
└────────────────────────────────────────────────────┘
```

---

## 6. Mutex

`sync.Mutex` (mutual exclusion lock) protects **shared mutable state** —
exactly the kind of thing Module 06's Calculator API and Module 07's Generic
Cache both flagged as needing protection once concurrent goroutines are
involved.

```go
type Counter struct {
	mu    sync.Mutex
	count int
}

func (c *Counter) Increment() {
	c.mu.Lock()
	defer c.mu.Unlock() // guarantees Unlock even if something panics — Module 02/08
	c.count++
}
```

```
┌────────────────────────────────────────────────────┐
│   Goroutine A: mu.Lock() ──▶ [critical section] ──▶ mu.Unlock()    │
│   Goroutine B:    mu.Lock() ──▶ BLOCKS until A unlocks ──▶ then proceeds  │
│                                                                                │
│   Only ONE goroutine can hold the lock at a time — everyone else                 │
│   attempting to Lock() simply waits their turn.                                     │
└────────────────────────────────────────────────────┘
```

**`defer mu.Unlock()` immediately after `Lock()` is close to a hard rule** —
any early return, or a panic, without it leaves the mutex locked forever,
and every other goroutine trying to acquire it blocks permanently (a
deadlock — see the Advanced section).

---

## 7. RWMutex

`sync.RWMutex` distinguishes **readers** from **writers** — any number of
readers can hold the lock simultaneously, but a writer needs exclusive
access, matching a very common real pattern: data read constantly, written
rarely.

```go
type Config struct {
	mu   sync.RWMutex
	data map[string]string
}

func (c *Config) Get(key string) string {
	c.mu.RLock()         // shared — many goroutines can RLock at once
	defer c.mu.RUnlock()
	return c.data[key]
}

func (c *Config) Set(key, value string) {
	c.mu.Lock()           // exclusive — blocks ALL readers and writers
	defer c.mu.Unlock()
	c.data[key] = value
}
```

```
┌────────────────────────────────────────────────────┐
│   RLock() / RLock() / RLock()   →  ALL allowed simultaneously,      │
│                                       as long as no writer holds Lock()   │
│                                                                                │
│   Lock()                          →  waits for ALL current readers to        │
│                                       finish, then holds EXCLUSIVELY            │
│                                                                                      │
│   Use RWMutex when reads vastly outnumber writes — for balanced                       │
│   read/write traffic, a plain Mutex is simpler and often just as fast.                   │
└────────────────────────────────────────────────────┘
```

---

## 8. Atomic

The `sync/atomic` package provides lock-free operations on individual
values — for simple counters or flags, atomics are cheaper than a full
`Mutex`, since there's no lock to acquire/release at all, just one
hardware-level atomic instruction.

```go
var counter atomic.Int64 // Go 1.19+ typed atomics — the modern, preferred API

counter.Add(1)
counter.Add(1)
fmt.Println(counter.Load()) // 2, guaranteed correct even with concurrent Add calls
```

```
┌────────────────────────────────────────────────────┐
│   Mutex:   Lock() → read-modify-write → Unlock()          │
│              (general purpose, protects ANY critical section) │
│                                                                    │
│   Atomic:  ONE hardware instruction does the whole                  │
│              read-modify-write indivisibly — no lock needed            │
│              (narrow purpose: single values only — counters,             │
│               flags, pointers)                                              │
└────────────────────────────────────────────────────┘
```

Reach for atomics specifically for simple counters/flags under heavy
concurrent access; reach for a `Mutex` the moment you need to update **more
than one related value together** (atomics can't keep two separate fields
consistent with each other — that needs a lock around both).

---

## 9. sync.Once

`sync.Once` guarantees a function runs **exactly once**, no matter how many
goroutines call it concurrently — the standard tool for lazy, thread-safe
initialization.

```go
var (
	once   sync.Once
	client *SomeExpensiveClient
)

func getClient() *SomeExpensiveClient {
	once.Do(func() {
		client = newExpensiveClient() // runs ONCE, even if 100 goroutines call getClient() simultaneously
	})
	return client
}
```

```
┌────────────────────────────────────────────────────┐
│   100 goroutines all call getClient() at roughly the same time    │
│                                                                        │
│   once.Do(fn)  →  exactly ONE of them actually runs fn;                  │
│                     the other 99 BLOCK until it finishes, then ALL          │
│                     99 proceed without running fn again                        │
└────────────────────────────────────────────────────┘
```

---

## 10. sync.Map

A specialized map safe for concurrent use without an external `Mutex` —
optimized specifically for two access patterns: keys written once and read
many times, or many goroutines operating on **disjoint** sets of keys.

```go
var m sync.Map

m.Store("key1", 100)
value, ok := m.Load("key1")
m.Delete("key1")

m.Range(func(key, value any) bool {
	fmt.Println(key, value)
	return true // return false to stop iterating early
})
```

```
┌────────────────────────────────────────────────────┐
│   A regular map + sync.Mutex:  general purpose, YOU control            │
│                                   exactly what's protected together          │
│                                                                                    │
│   sync.Map:                     specialized for high-churn concurrent           │
│                                   access with little cross-key coordination         │
│                                   needed — but its API is `any`-typed              │
│                                   (no generics), so you lose compile-time            │
│                                   type safety compared to a typed map                  │
│                                   guarded by a Mutex                                       │
└────────────────────────────────────────────────────┘
```

**Default to a plain `map` protected by a `Mutex`/`RWMutex` unless you have
a specific, measured reason to reach for `sync.Map`** — it's a narrower tool
than it looks, and the Go team has said as much explicitly in its own docs.

---

## 11. Condition Variables

`sync.Cond` lets goroutines **wait** for a condition to become true, and be
**woken up** by whichever goroutine changes that condition — useful when
`Mutex` alone isn't enough because a goroutine needs to *block* until some
specific state changes, not just get exclusive access to check it once.

```go
type BoundedQueue struct {
	mu       sync.Mutex
	cond     *sync.Cond
	items    []int
	capacity int
}

func NewBoundedQueue(capacity int) *BoundedQueue {
	q := &BoundedQueue{capacity: capacity}
	q.cond = sync.NewCond(&q.mu) // Cond is built ON TOP of a Mutex/Locker
	return q
}

func (q *BoundedQueue) Push(item int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.items) >= q.capacity { // NOTE: a `for` loop, not `if` — see below
		q.cond.Wait() // atomically unlocks mu AND sleeps, until Signal/Broadcast
	}
	q.items = append(q.items, item)
	q.cond.Signal() // wake ONE waiting goroutine (e.g. a Pop waiting for an item)
}
```

```
┌────────────────────────────────────────────────────┐
│   cond.Wait()  →  atomically: unlock the mutex, THEN sleep              │
│                     when woken: re-acquire the mutex before returning       │
│                                                                                    │
│   cond.Signal()    →  wake ONE waiting goroutine                                    │
│   cond.Broadcast()  →  wake ALL waiting goroutines                                     │
│                                                                                              │
│   ALWAYS re-check the condition in a `for` loop after Wait() returns,                          │
│   never assume it's still true — another goroutine might have changed                            │
│   it again before YOUR goroutine got scheduled back in (a "spurious                                 │
│   wakeup" is even possible, though rare)                                                              │
└────────────────────────────────────────────────────┘
```

**In practice, channels solve the vast majority of problems `sync.Cond`
could solve, more simply** (a buffered channel IS a bounded queue with
built-in blocking behavior). Reach for `sync.Cond` specifically when you
need to wake goroutines waiting on a condition that **isn't naturally
"a value arrived on a channel"** — e.g., "wait until this counter drops
below N" where N different things might cause that.

---

## 12. Select

`select` waits on **multiple channel operations at once**, proceeding with
whichever one is ready first — the concurrent equivalent of a `switch` over
channels.

```go
select {
case v := <-ch1:
	fmt.Println("got from ch1:", v)
case v := <-ch2:
	fmt.Println("got from ch2:", v)
case ch3 <- 42:
	fmt.Println("sent to ch3")
default:
	fmt.Println("none ready — this branch makes select NON-BLOCKING")
}
```

```
┌────────────────────────────────────────────────────┐
│   select {                                                │
│       case <-ch1: ...      ◀── each case is a channel          │
│       case <-ch2: ...            operation — Go picks whichever    │
│       case ch3<-v: ...            is READY FIRST                     │
│   }                                                                       │
│                                                                              │
│   If MULTIPLE are ready simultaneously, Go picks one at RANDOM —              │
│   deliberately, to avoid any case starving the others over time.                 │
│                                                                                        │
│   No `default` → select BLOCKS until at least one case is ready                         │
│   WITH `default` → select never blocks; falls through immediately                          │
└────────────────────────────────────────────────────┘
```

**The single most common `select` pattern** — timeout via `time.After`:
```go
select {
case result := <-workCh:
	fmt.Println("got result:", result)
case <-time.After(2 * time.Second):
	fmt.Println("timed out waiting for result")
}
```

And the standard **cancellation** pattern via `context.Context`
(`ctx.Done()` returns a channel that closes when the context is cancelled):
```go
select {
case result := <-workCh:
	return result, nil
case <-ctx.Done():
	return nil, ctx.Err()
}
```

---

Continued below in the same file — Worker Pools, Pipelines, Fan-in/Fan-out,
then the Advanced section (races, deadlocks, the memory model, and the
scheduler).
## 13. Worker Pools

A worker pool is N goroutines all reading from the **same** channel of work,
so at most N pieces of work run concurrently — controlling parallelism
instead of spawning an unbounded goroutine per task (which can exhaust
memory or overwhelm a downstream system).

```go
func worker(id int, jobs <-chan int, results chan<- int) {
	for job := range jobs { // keeps pulling until jobs is closed AND drained
		results <- job * 2 // pretend this is real work
	}
}

func main() {
	jobs := make(chan int, 100)
	results := make(chan int, 100)

	for w := 1; w <= 3; w++ { // 3 workers — at most 3 jobs process concurrently
		go worker(w, jobs, results)
	}

	for j := 1; j <= 9; j++ {
		jobs <- j
	}
	close(jobs) // signals workers: no more jobs coming, finish and exit

	for a := 1; a <= 9; a++ {
		fmt.Println(<-results)
	}
}
```

```
┌────────────────────────────────────────────────────────┐
│   jobs channel:  [1][2][3][4][5][6][7][8][9]                    │
│                       │                                             │
│         ┌─────────────┼─────────────┐                                 │
│         ▼             ▼             ▼                                   │
│      worker 1      worker 2      worker 3     ◀── all reading from        │
│         │             │             │              the SAME jobs channel     │
│         ▼             ▼             ▼                                            │
│                  results channel                                                     │
└────────────────────────────────────────────────────────┘
```

Note the parameter types: `jobs <-chan int` (receive-only) and
`results chan<- int` (send-only) — Go lets you narrow a bidirectional
channel to one direction in a function signature, which is good practice:
the compiler now enforces that `worker` can never accidentally send on
`jobs` or receive from `results`.

---

## 14. Pipelines

A pipeline chains several **stages** together, each one a goroutine (or
group of goroutines) reading from an input channel and writing to an output
channel — data flows through the whole chain concurrently, each stage
working on a different item at the same time.

```go
func generate(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out) // ALWAYS close the channel you own when you're done sending
		for _, n := range nums {
			out <- n
		}
	}()
	return out
}

func square(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			out <- n * n
		}
	}()
	return out
}

// Chained: generate -> square -> square
for v := range square(square(generate(2, 3, 4))) {
	fmt.Println(v) // 16, 81, 256 — each number's fourth power
}
```

```
┌────────────────────────────────────────────────────────┐
│   generate(2,3,4)  ──▶  square  ──▶  square  ──▶  range prints           │
│         │                   │              │                                │
│    goroutine A          goroutine B    goroutine C   — all running             │
│                                                          CONCURRENTLY,             │
│                                                          each processing              │
│                                                          different items                 │
│                                                          of the stream at                   │
│                                                          the same moment                       │
└────────────────────────────────────────────────────────┘
```

**Each stage owns closing its own output channel** — this is the rule that
makes `for range` on the next stage terminate correctly, and it's why
`generate`/`square` both `defer close(out)`.

---

## 15. Fan-out

Fan-out is starting **multiple goroutines reading from the same channel**,
to spread work across them — this is exactly what the Worker Pool section
did (3 workers, one `jobs` channel); "fan-out" is the general name for that
shape when it appears as a pipeline stage.

```
┌────────────────────────────────────────────────────┐
│                        ┌──▶ worker A                       │
│   input channel  ──────┼──▶ worker B     (fan-OUT: one           │
│                        └──▶ worker C      source, many consumers)   │
└────────────────────────────────────────────────────┘
```

---

## 16. Fan-in

Fan-in is the reverse: **merging multiple channels into one**, so a single
downstream consumer can read from many upstream sources without knowing how
many there are or juggling several `select` cases by hand.

```go
func fanIn(channels ...<-chan int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup

	for _, ch := range channels {
		wg.Add(1)
		go func(c <-chan int) {
			defer wg.Done()
			for v := range c {
				out <- v
			}
		}(ch)
	}

	go func() {
		wg.Wait()   // wait for ALL input channels to be drained and closed
		close(out)   // THEN close the merged output channel
	}()

	return out
}
```

```
┌────────────────────────────────────────────────────┐
│   worker A ──┐                                            │
│   worker B ──┼──▶  merged output channel   (fan-IN: many         │
│   worker C ──┘                                sources, one consumer)│
│                                                                          │
│   WaitGroup tracks when ALL inputs are done, so `out` only closes           │
│   once every source has finished — closing too early would drop                │
│   values still in flight from a slower source.                                    │
└────────────────────────────────────────────────────┘
```

Fan-out and fan-in are almost always used **together**: fan-out spreads work
across workers, fan-in collects their results back into one stream for the
next pipeline stage or the final consumer — this exact combination is what
the Concurrent Web Scraper project builds.

---

## Advanced: Race Conditions, Deadlocks, the Memory Model, and the Scheduler

### Race Conditions

A data race happens when **two or more goroutines access the same memory
location concurrently, at least one of them writing, with no
synchronization** — the outcome becomes genuinely unpredictable, not just
"wrong in an obvious way."

```go
// BROKEN — a real, classic data race:
counter := 0
var wg sync.WaitGroup
for i := 0; i < 1000; i++ {
	wg.Add(1)
	go func() {
		defer wg.Done()
		counter++ // READ, increment, WRITE — three steps, not one atomic operation
	}()
}
wg.Wait()
fmt.Println(counter) // often NOT 1000 — some increments get lost
```

```
┌────────────────────────────────────────────────────────┐
│   Goroutine A reads counter (0)                                  │
│                        Goroutine B reads counter (0)  ◀── BOTH read        │
│   Goroutine A writes 1                                    the SAME old        │
│                        Goroutine B writes 1  ◀── LOST an increment!  value        │
└────────────────────────────────────────────────────────┘
```

**Fix with either a `Mutex` or an atomic** (both from earlier in this
guide):
```go
var mu sync.Mutex
counter := 0
// ... inside the goroutine:
mu.Lock()
counter++
mu.Unlock()

// OR, simpler for a single counter:
var counter atomic.Int64
// ... inside the goroutine:
counter.Add(1)
```

**Go ships a built-in race detector** — run any program or test with
`-race` and it instruments every memory access, reporting races with exact
file/line locations for both conflicting accesses:
```bash
go run -race main.go
go test -race ./...
```
**Run this constantly during development of concurrent code.** Races often
don't manifest as a visible bug for thousands of runs, then fail
unpredictably in production — the race detector catches the *possibility*,
not just the rare moment it actually goes wrong.

### Deadlocks

A deadlock is when goroutines are all stuck waiting on each other, and
**none of them can ever proceed** — Go's runtime can even detect the
simplest case (the whole program deadlocked, nothing left running at all)
and crashes with `fatal error: all goroutines are asleep - deadlock!`
instead of hanging forever silently.

```go
// BROKEN — deadlocks immediately:
ch := make(chan int) // unbuffered
ch <- 1                // main goroutine sends... but nobody is receiving,
                         // and nothing else is running to receive it
fmt.Println(<-ch)        // never reached
```

**The classic two-mutex deadlock** (each goroutine holds one lock the other
needs):
```go
// Goroutine A:  mu1.Lock(); mu2.Lock(); ...   (locks in order 1, 2)
// Goroutine B:  mu2.Lock(); mu1.Lock(); ...   (locks in order 2, 1)
// If A holds mu1 and B holds mu2 at the same moment, both block FOREVER
// waiting for the other's lock.
```

```
┌────────────────────────────────────────────────────────┐
│   Goroutine A:  holds mu1, WANTS mu2                             │
│   Goroutine B:  holds mu2, WANTS mu1                                │
│                                                                          │
│        A ──waits for──▶ mu2 (held by B)                                    │
│        ▲                              │                                       │
│        └──────── B waits for mu1 ◀────┘                                          │
│                                                                                        │
│   A circular wait — neither can EVER proceed.                                            │
└────────────────────────────────────────────────────────┘
```

**Fix: always acquire multiple locks in the same, consistent order across
every goroutine** — if every goroutine locks `mu1` before `mu2`, the
circular wait above becomes structurally impossible. This is the single most
important deadlock-avoidance rule for multi-lock code.

### Memory Model

Go's memory model defines exactly which **"happens-before"** guarantees you
can rely on when goroutines communicate — without one of these guarantees,
the compiler and CPU are both free to reorder or cache operations in ways
that break naive assumptions about "the other goroutine will see my write."

**The guarantees that matter most in everyday code:**
```
┌────────────────────────────────────────────────────────┐
│   A channel SEND happens-before the corresponding RECEIVE           │
│   completes — anything goroutine A did before sending is                │
│   GUARANTEED visible to goroutine B after it receives.                     │
│                                                                                │
│   A Mutex Unlock happens-before the next successful Lock on the                 │
│   same mutex — anything done while holding the lock is visible to                  │
│   whoever locks it next.                                                              │
│                                                                                              │
│   A `go` statement happens-before the goroutine it starts begins             │
│   running — the new goroutine sees everything set up before `go`.               │
└────────────────────────────────────────────────────────┘
```

**The practical takeaway:** channels and the `sync` package aren't just
"convenient" ways to coordinate goroutines — they're the *only* things
providing these visibility guarantees. Reading/writing a shared variable
with no channel or `sync` primitive involved is a data race even if it
"seems to work" in testing, precisely because the memory model makes no
promises at all about visibility in that case — Section "Race Conditions"
above is this same guarantee, from the failure side.

### Scheduler

Introduced in Module 01's runtime section — Go's **M:N scheduler** maps many
goroutines (M) onto a smaller number of OS threads (N), controlled by
`GOMAXPROCS` (defaults to the number of logical CPUs).

```
┌────────────────────────────────────────────────────────┐
│   Thousands of goroutines (cheap, ~2KB stack each)                  │
│         │        │        │        │        │        │                │
│         └────────┴────────┴────────┴────────┴────────┘                   │
│                              │                                               │
│                    Go SCHEDULER multiplexes them                                │
│                              │                                                     │
│              ┌───────────────┼───────────────┐                                       │
│              ▼               ▼               ▼                                         │
│         OS thread 1     OS thread 2     OS thread 3    (typically ==                       │
│                                                           GOMAXPROCS,                          │
│                                                           i.e. CPU cores)                          │
└────────────────────────────────────────────────────────┘
```

The scheduler is **cooperative-ish**: a goroutine yields the thread it's
running on at function calls, channel operations, `select`, blocking system
calls, and a few other points — not at arbitrary instructions. This is why a
goroutine stuck in a tight loop with no function calls or channel ops at all
can, in rare cases, starve others on the same thread; in practice this is
extremely uncommon in normal Go code, since almost everything you write
naturally includes enough of these yield points.

```bash
GOMAXPROCS=1 go run main.go   # force everything onto ONE OS thread —
                                # goroutines still run concurrently
                                # (interleaved), just never in true
                                # PARALLEL (simultaneous) execution
```

**Concurrency vs. parallelism, precisely:** concurrency is *structuring* a
program as independently-progressing pieces (true even with `GOMAXPROCS=1`
— goroutines still interleave); parallelism is those pieces *literally
executing at the same instant* on different cores (needs `GOMAXPROCS > 1`,
and only helps for genuinely CPU-bound work — I/O-bound work benefits from
concurrency even on a single core, since goroutines yield while waiting on
I/O instead of blocking everything).

---

Onto the projects — Concurrent Web Scraper combines fan-out/fan-in and
`sync.Map`; Job Queue centers on worker pools, `WaitGroup`, and a
Mutex-protected status map; Email Worker uses `sync.Once`, atomics, and an
`RWMutex`; Image Processor builds a full multi-stage pipeline with
`select`-based cancellation.
