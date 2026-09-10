# Project — Optimize REST API

A real REST API (`main.go`, serving optimized handlers) plus a head-to-head
benchmark suite proving each optimization actually helped.

## Setup

```bash
cd optimize-rest-api
go run .                         # the actual API, with pprof on :6060
go test -bench=. -benchmem ./...   # the before/after comparison
```

## Reading the Benchmark Results

```
BenchmarkBuildCSVReport_Unoptimized-8    500    2,847,193 ns/op   1,048,912 B/op   3011 allocs/op
BenchmarkBuildCSVReport_Optimized-8     8000      182,004 ns/op      49,536 B/op      12 allocs/op
```

```
┌──────────────────────────────────────────────────────────┐
│   ~15x faster, ~21x less memory, ~250x fewer allocations —              │
│   exactly what fixing an O(n²) string-concatenation pattern with            │
│   strings.Builder should produce. The gap WIDENS as input size grows,           │
│   since the unoptimized version's cost grows QUADRATICALLY with rows,              │
│   while the optimized version's grows LINEARLY.                                       │
└──────────────────────────────────────────────────────────┘
```

Run `go install golang.org/x/perf/cmd/benchstat@latest` and compare with
`benchstat` for a statistically rigorous before/after, per the guide's
Benchmarking section, instead of eyeballing two numbers.

## Profiling the Real Server

```bash
go run .
# in another terminal, generate some load first:
for i in $(seq 1 200); do curl -s "http://localhost:8080/products/report.csv" > /dev/null; done

go tool pprof http://localhost:6060/debug/pprof/profile?seconds=10
(pprof) top
(pprof) list optimized.BuildCSVReport
```

## Checking Escape Analysis and Inlining Directly

```bash
go build -gcflags="-m" ./unoptimized/ 2>&1 | grep -E "escapes|inline"
go build -gcflags="-m" ./optimized/ 2>&1 | grep -E "escapes|inline"
```

Compare `unoptimized.LogValue`'s output against `optimized.LogValue`'s —
the `any`-typed version should show its argument escaping to the heap;
the `float64`-typed version should not, exactly per the guide's Escape
Analysis section.

## What Each Function Demonstrates

| Function | Unoptimized problem | Fix |
|---|---|---|
| `FilterByCategory` | No capacity pre-allocation | `make([]T, 0, estimate)` |
| `BuildCSVReport` | `+=` string concatenation, O(n²) | `strings.Builder` |
| `SearchByNamePattern` | Recompiles the regex every call | A `sync.Mutex`-protected compiled-pattern cache |
| `Summarize` | Unnecessary intermediate `[]string` + `Join` | Direct `strings.Builder` writes |
| `LogValue` | `any` parameter forces a heap escape | A concrete, typed parameter |

## Try It Yourself
- Add a `BenchmarkFilterByCategory_Unoptimized` variant at 10,000 and
  100,000 products, and watch the allocation-count GAP (not just the
  absolute numbers) grow between the two versions
- Add `//go:noinline` above one of the optimized functions and confirm via
  `-gcflags="-m"` that inlining no longer applies to it — then benchmark
  again to see how much (or little) that specific function's inlining was
  actually contributing
- Wire up `GODEBUG=gctrace=1` while running the server under heavy load
  (a longer curl loop) and compare GC cycle frequency between hitting the
  unoptimized vs. optimized CSV endpoint — swap `main.go` to call
  `unoptimized.BuildCSVReport` temporarily to see the difference directly
