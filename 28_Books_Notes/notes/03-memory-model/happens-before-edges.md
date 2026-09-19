# Only channels, sync primitives, and atomics create happens-before edges — nothing else guarantees one goroutine sees another's writes

**Source**: https://go.dev/ref/mem
**Why I read this**: I had code that "worked" reading a bool flag set by
another goroutine without any synchronization, and wanted to know whether
it was correct or just lucky.

## The claim, in my words

Without a happens-before edge, the compiler and CPU are both free to
reorder and cache writes. A goroutine may *never* observe another's write,
no matter how long it waits. "It worked in testing" proves nothing — it's
undefined behavior, not a race you sometimes win.

## The edges that exist (the memorizable list)

```
- A send on a channel happens-before the corresponding receive completes
- A receive from an unbuffered channel happens-before the send completes
- close(ch) happens-before a receive that returns zero-value-because-closed
- mu.Unlock() happens-before a subsequent mu.Lock() returns
- once.Do(f) — f completes before any Do() call returns
- A `go` statement happens-before the new goroutine starts
- sync/atomic operations on the same address are totally ordered
```

## Concrete example

```go
// BROKEN — no edge. `done` may never be observed as true.
var done bool
var msg string
go func() { msg = "hello"; done = true }()
for !done { }          // may spin forever, legally
fmt.Println(msg)       // may print "" even if done became true

// CORRECT — the channel creates the edge
var msg string
done := make(chan struct{})
go func() { msg = "hello"; close(done) }()
<-done
fmt.Println(msg)       // GUARANTEED to print "hello"
```

**What actually happened when I ran it**: the broken version terminated
on my machine at `GOMAXPROCS>1`, and hung under `GOMAXPROCS=1` — a
perfect demonstration that "works on my machine" means nothing here.
`go run -race` flags the broken version immediately.

## When this applies / doesn't

Applies to **every** shared variable touched by more than one goroutine
where at least one access is a write. Read-only shared data after
initialization is fine — but the initialization itself must happen-before
the goroutines start (the `go` statement edge covers this).

## Links

- Module 12 (race detector, `-race`)
- Module 22 (the same reasoning, extended across machines)
