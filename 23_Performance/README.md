# Go for Beginners — Module 23: Performance

## Contents

1. **[23-performance.md](./23-performance.md)** — Profiling and pprof
   (including reading `flat` vs. `cum` correctly), CPU profiling via
   sampling, memory optimization (string building, slice pre-allocation,
   `sync.Pool`, struct field ordering), the garbage collector (`GOGC`,
   `GOMEMLIMIT`, and why reducing allocations beats tuning GC knobs),
   benchmarking with `benchstat`, escape analysis, and inlining. Diagrams
   throughout.

2. **[optimize-rest-api/](./optimize-rest-api)** — A real REST API with a
   deliberately inefficient `unoptimized` package and a fixed `optimized`
   package implementing the exact same functions, benchmarked head to
   head with `-benchmem`. Covers every inefficiency from the guide's
   Memory Optimization section for real: O(n²) string concatenation fixed
   with `strings.Builder`, missing slice capacity hints, a regex
   recompiled on every call fixed with a cached, mutex-protected compiled
   pattern, and an `any`-typed parameter's forced heap escape fixed with a
   concrete type — checkable directly via `-gcflags="-m"`.

## Suggested Order

```
Performance guide ──▶ Optimize REST API
```

## Setup

```bash
cd optimize-rest-api
go run .                          # the real API, pprof on :6060
go test -bench=. -benchmem ./...    # before/after comparison
```

No external dependencies beyond the optional `benchstat` tool mentioned in
the guide.

*Note: this module builds on Module 04 (slices — the pre-allocation fix),
Module 05 (escape analysis, introduced there and made practical here),
Module 12 (the mutex-protected regex cache), and Module 14 (benchmarking
mechanics, extended here with profiling flags).*
