# sync.Once uses an atomic fast path plus a mutex slow path — the atomic alone would be a race

**Source**: Go source, `src/sync/once.go`
**Why I read this**: `sync.Once` is ~20 lines and I wanted to understand
why it needs *both* an atomic and a mutex rather than just one.

## The implementation (paraphrased)

```go
type Once struct {
    done atomic.Uint32
    m    Mutex
}

func (o *Once) Do(f func()) {
    if o.done.Load() == 0 {    // FAST PATH: atomic read, no lock
        o.doSlow(f)
    }
}

func (o *Once) doSlow(f func()) {
    o.m.Lock()
    defer o.m.Unlock()
    if o.done.Load() == 0 {    // re-check UNDER the lock
        defer o.done.Store(1)  // set AFTER f() returns
        f()
    }
}
```

## Why both primitives are needed

The atomic alone can't work: two goroutines could both read `done == 0`
simultaneously and both run `f()`. The mutex is what makes the
check-then-act atomic as a unit.

The mutex alone would work correctly but would make *every* call after
the first acquire a lock — the atomic fast path means the common case
(already done) is a single uncontended atomic load.

## The subtle part worth noting

`done` is set **after** `f()` returns, not before. This is what guarantees
that when `Do` returns to *any* caller, `f` has genuinely completed — a
second goroutine calling `Do` while the first is mid-`f()` blocks on the
mutex rather than proceeding early. Module 03's memory model note covers
the happens-before edge this establishes.

## What I took from reading it

The "atomic fast path + mutex slow path" shape is reusable — it's the
right structure whenever a one-time (or rare) expensive operation guards
a very frequently-read flag.

## Links

- Module 12 (`sync.Once` usage, mutexes, atomics)
- Related: `notes/03-memory-model/happens-before-edges.md`
