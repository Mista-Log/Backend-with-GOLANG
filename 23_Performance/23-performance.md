# 23. Performance

Every module so far optimized for correctness and clarity first — the
right default. This module is about the moment correctness is no longer
the bottleneck, and you need to make correct code *fast*, guided by
measurement rather than guesswork. Go's toolchain has exceptional built-in
support for exactly this.

---

## Profiling

**Never optimize based on intuition alone** — profiling measures where a
program *actually* spends time and memory, which is reliably different
from where you'd guess. Go's `runtime/pprof` and `net/http/pprof` packages
make this close to zero-setup.


## Profiling

**Never optimize based on intuition alone** — profiling measures where a
program *actually* spends time and memory, which is reliably different
from where you'd guess. Go's `runtime/pprof` and `net/http/pprof` packages
make this close to zero-setup.

## Profiling

**Never optimize based on intuition alone** — profiling measures where a
program *actually* spends time and memory, which is reliably different
from where you'd guess. Go's `runtime/pprof` and `net/http/pprof` packages
make this close to zero-setup.


```
┌──────────────────────────────────────────────────────────┐
│   Profile types Go supports out of the box:                          │
│                                                                          │
│   CPU        → where is execution time actually spent?                    │
│   Heap        → what's currently allocated, and by what code?               │
│   Allocs        → what code has allocated the MOST over the program's         │
│                     lifetime (even if since freed)?                              │
│   Goroutine       → how many goroutines exist, and what are they doing?           │
│   Block            → where are goroutines blocking on channels/mutexes?              │
│   Mutex             → which locks have the most contention?                             │
└──────────────────────────────────────────────────────────┘
```

---

## pprof

For a long-running service, `net/http/pprof` exposes live profiling data
over HTTP with a **single import**:

```go
import _ "net/http/pprof" // registers /debug/pprof/* handlers on the DEFAULT mux, as a side effect

func main() {
	go http.ListenAndServe("localhost:6060", nil) // a SEPARATE port, never exposed publicly
	// ... your real server, on its own port/mux ...
}
```

```bash
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30   # CPU profile, 30s sample
go tool pprof http://localhost:6060/debug/pprof/heap                    # current heap snapshot
go tool pprof http://localhost:6060/debug/pprof/allocs                    # lifetime allocation profile
```

For a one-off program or benchmark, generate a profile file directly:
```go
f, _ := os.Create("cpu.prof")
pprof.StartCPUProfile(f)
defer pprof.StopCPUProfile()
// ... code to profile ...
```
```bash
go tool pprof cpu.prof
```

**Inside the interactive `pprof` tool**, the commands worth knowing first:
```
┌────────────────────────────────────────────────────┐
│   top          → the functions consuming the MOST time/memory,                │
│                    ranked                                                        │
│   top -cum      → ranked by CUMULATIVE time (including everything                   │
│                     a function calls) rather than time IN that                         │
│                     function alone                                                        │
│   list FuncName  → line-by-line breakdown for one function                                   │
│   web             → renders an interactive call graph in your browser                           │
│                       (needs Graphviz installed)                                                   │
└────────────────────────────────────────────────────┘
```

```
┌──────────────────────────────────────────────────────────┐
│   flat vs. cum, precisely:                                          │
│                                                                          │
│   flat  = time spent DIRECTLY in this function's own code                  │
│   cum   = flat + time spent in everything this function CALLS               │
│                                                                                  │
│   A function with high CUM but low FLAT is a thin wrapper around                   │
│   expensive work elsewhere — the REAL hotspot is further down the call                │
│   graph, not the wrapper itself.                                                         │
└──────────────────────────────────────────────────────────┘
```

---

## CPU Profiling

CPU profiles work by **sampling**: the runtime interrupts execution
roughly 100 times per second and records the current call stack — over
many samples, functions where the program spends more time appear more
often. This is why very short-running programs produce noisy, unreliable
CPU profiles: too few samples to be statistically meaningful.

```
┌──────────────────────────────────────────────────────────┐
│   go test -bench=. -cpuprofile=cpu.prof ./...                        │
│   go tool pprof cpu.prof                                                │
│   (pprof) top                                                              │
│                                                                                │
│   flat  flat%   sum%        cum   cum%                                          │
│   2.1s  42.0%  42.0%       2.1s  42.0%   encoding/json.(*encodeState).string      │
│   1.3s  26.0%  68.0%       1.3s  26.0%   fmt.Sprintf                                 │
│   0.8s  16.0%  84.0%       4.2s  84.0%   main.buildResponse                            │
│                                                                                              │
│   Reading this: buildResponse itself is cheap (0.8s flat) but its CUM               │
│   time (4.2s) reveals it's the caller driving most of the expensive JSON               │
│   encoding and Sprintf calls below it — THAT'S where to look for a fix.                  │
└──────────────────────────────────────────────────────────┘
```

---
## Memory Optimization

The single biggest lever for Go performance in practice: **reducing
allocations**, since every heap allocation costs CPU time to allocate and
adds work for the garbage collector later.

```go
// ❌ Allocates a new string on every concatenation — O(n²) for n pieces
func buildCSV(rows []string) string {
	result := ""
	for _, r := range rows {
		result += r + "\n" // each += allocates a NEW string, copying everything so far
	}
	return result
}

// ✅ strings.Builder grows one internal buffer, amortized O(n) total
func buildCSV(rows []string) string {
	var b strings.Builder
	b.Grow(estimatedSize(rows)) // pre-size when you can — avoids repeated regrowth
	for _, r := range rows {
		b.WriteString(r)
		b.WriteByte('\n')
	}
	return b.String()
}
```

```go
// ❌ append() with no capacity hint may reallocate multiple times as it grows
func collect(n int) []int {
	var out []int
	for i := 0; i < n; i++ {
		out = append(out, i) // Module 04's capacity-doubling reallocation, repeatedly
	}
	return out
}

// ✅ pre-allocate the capacity you already know you'll need
func collect(n int) []int {
	out := make([]int, 0, n) // ONE allocation, sized correctly up front
	for i := 0; i < n; i++ {
		out = append(out, i)
	}
	return out
}
```

**`sync.Pool`** reuses temporary objects across requests instead of
allocating fresh ones every time — most valuable for short-lived, mid-size
buffers (a `bytes.Buffer` used once per HTTP request, for instance):

```go
var bufferPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

func handler(w http.ResponseWriter, r *http.Request) {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset() // ALWAYS reset before reuse — it may still hold a previous caller's data
	defer bufferPool.Put(buf)
	// ... use buf ...
}
```

**Struct field ordering** affects memory size via alignment padding —
Go pads fields so each one starts at an address matching its own size
requirement, which can leave gaps:

```go
type Inefficient struct {
	A bool    // 1 byte, then 7 bytes of padding to align B
	B int64   // 8 bytes
	C bool    // 1 byte, then 7 bytes of padding (struct size rounds up)
} // total: 24 bytes

type Efficient struct {
	B int64   // 8 bytes
	A bool    // 1 byte
	C bool    // 1 byte, then 6 bytes padding
} // total: 16 bytes — same fields, ordered LARGEST to SMALLEST
```
`go vet -fieldalignment` (or the `fieldalignment` analyzer) flags exactly
this — worth running on any struct you'll allocate millions of.

---

## Garbage Collector

Go's GC is **concurrent** (runs alongside your program, not stop-the-world
for its main phases) and uses a **tri-color mark-and-sweep** algorithm —
you rarely need the internals, but two levers matter in practice:

```bash
GOGC=100     # default: trigger a GC cycle when heap has grown 100% since the last cycle
GOGC=200      # trigger LESS often — trades more memory for less CPU spent on GC
GOGC=off       # disable entirely (rare — only for short-lived batch jobs where you'd rather
                # let memory grow than pay ANY GC cost, since the process exits soon anyway)

GOMEMLIMIT=1GiB  # Go 1.19+: a hard memory ceiling — the GC works MORE aggressively as usage
                   # approaches this, useful in memory-constrained containers
```

```
┌──────────────────────────────────────────────────────────┐
│   Fewer, smaller allocations  →  less work for the GC on every cycle,   │
│                                     REGARDLESS of GOGC tuning               │
│                                                                                 │
│   This is why "reduce allocations" (the Memory Optimization section)              │
│   is almost always a better lever to pull FIRST than tuning GOGC — it                │
│   reduces the GC's workload directly, rather than just changing how                     │
│   often it runs against the SAME workload.                                                 │
└──────────────────────────────────────────────────────────┘
```

`GODEBUG=gctrace=1` prints one line per GC cycle to stderr — genuinely
useful for seeing GC frequency and pause times directly, without needing a
full profile.

---

## Benchmarking

Module 14 covered the mechanics (`testing.B`, `b.N`, `-benchmem`); the
performance-specific habit worth adding here is **profiling straight from
a benchmark**, so you're profiling the exact, isolated code path you
actually care about:

```bash
go test -bench=BenchmarkBuildCSV -cpuprofile=cpu.prof -memprofile=mem.prof ./...
go tool pprof cpu.prof
go tool pprof mem.prof
```

**Always compare benchmarks with `benchstat`**, not by eyeballing two
numbers — run-to-run noise is real, and `benchstat` reports whether a
difference is statistically meaningful:

```bash
go install golang.org/x/perf/cmd/benchstat@latest
go test -bench=. -count=10 ./... > old.txt   # BEFORE your change
go test -bench=. -count=10 ./... > new.txt   # AFTER your change
benchstat old.txt new.txt
```

---
## Escape Analysis

Introduced conceptually in Module 01 and used practically in Module 05 —
now the direct performance angle: every value the compiler decides must
live on the **heap** (because its address outlives the function, Module
05) costs an allocation the GC later has to track and reclaim. A value
that stays on the **stack** costs essentially nothing — freed the instant
its function returns, no GC involvement at all.

```bash
go build -gcflags="-m" ./...
# ./main.go:12:6: moved to heap: buf
# ./main.go:20:9: myFunc returns address of local variable, escapes to heap
```

```go
// ❌ Returning a pointer to a LARGE local struct forces a heap allocation
func newRequest() *http.Request {
	req := http.Request{ /* many fields */ }
	return &req // escapes — the caller needs it after this function returns
}

// Sometimes escape is genuinely unavoidable (the caller DOES need the
// value to outlive this call) — but a surprising number of escapes come
// from something narrower and fixable:

// ❌ Passing to an interface{}-typed parameter FORCES a heap allocation,
// even for a value that would otherwise safely stay on the stack —
// the compiler can't prove the interface won't be stored somewhere
// long-lived, so it plays it safe.
func logValue(v any) { fmt.Println(v) }

func process() {
	x := 42
	logValue(x) // x escapes to heap here, purely because of the `any` parameter
}
```

**Escape analysis surprises are worth hunting for specifically in hot
paths** (functions called millions of times per second) — `-gcflags="-m"`
output is verbose, but grepping for `escapes to heap` against your busiest
package is a genuinely productive five minutes.

---

## Inlining

The compiler can replace a small function call with that function's
actual body directly at the call site — eliminating call overhead
entirely, and often unlocking *further* optimizations (like keeping a
value on the stack that would otherwise have needed to escape across a
real function call boundary).

```bash
go build -gcflags="-m" ./...
# ./math.go:5:6: can inline Add
# ./main.go:12:9: inlining call to Add
```

```go
func Add(a, b int) int { return a + b } // trivially small — WILL be inlined

func ComplexCalculation(data []float64) float64 {
	// many lines, loops, branches — TOO large to inline;
	// the compiler has a cost budget for what it will inline,
	// and this exceeds it
}
```

```
┌──────────────────────────────────────────────────────────┐
│   Go's inliner is conservative and BUDGET-based — small, simple             │
│   functions (getters, simple arithmetic, thin wrappers) are prime               │
│   candidates; anything with loops, multiple returns, or significant                │
│   complexity generally isn't, on purpose (aggressive inlining bloats                  │
│   binary size and can hurt instruction-cache behavior instead of helping).                │
│                                                                                                │
│   `//go:noinline` above a function FORCES the compiler to skip inlining                         │
│   it — genuinely useful when writing a MICRO-BENCHMARK specifically                                │
│   isolating that one function's own cost, where inlining would otherwise                              │
│   fold it into the caller and make the benchmark measure something else                                  │
│   entirely.                                                                                                  │
└──────────────────────────────────────────────────────────┘
```

**You almost never write code specifically "to be inlined"** — the
compiler's decisions here are a byproduct of writing small, focused
functions for their own sake (Module 03's general style advice), not a
target to chase directly. Knowing `-gcflags="-m"` can show you these
decisions is mainly useful for understanding *why* a benchmark's numbers
look the way they do, not for micromanaging the compiler.

---

Onto the project — Optimize REST API takes one deliberately inefficient
API handler, profiles it, and fixes it one measured step at a time:
excessive string concatenation, missing slice pre-allocation, unnecessary
heap escapes, and repeated work that belongs outside the hot path —
with real benchmarks proving each fix actually helped.
