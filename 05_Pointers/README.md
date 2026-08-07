# Go for Beginners — Module 05: Pointers

## Contents

1. **[05-pointers.md](./05-pointers.md)** — Memory, addresses (`&`), dereferencing
   (`*`), pointer receivers vs. value receivers (and why real Go code picks one
   consistently per type), escape analysis, the heap, and the stack — including
   how Go's escape analysis makes returning a pointer to a local variable
   completely safe, unlike C. Diagrams included throughout.

2. **[mini-database/](./mini-database)** — An in-memory key/value store built
   entirely to make pointer semantics tangible: a dangerous `GetLive` (hands
   out the real stored pointer) next to a safe `GetCopy` (dereferences and
   copies), a `Set` method whose local variable visibly needs heap allocation
   (confirmable with `go build -gcflags="-m"`), and a transaction/rollback
   system whose correctness depends entirely on `Snapshot()` copying values
   instead of pointers.

## Suggested Order

```
Pointers guide ──▶ Mini Database
```

The project is a single, focused piece of work — read the case studies in
its README alongside the code; they walk through *why* each pointer-related
decision was made, not just what the code does.

## Quick Reference: Running the Project

```bash
cd mini-database
go run main.go

# to see the compiler's escape analysis decisions directly:
go build -gcflags="-m" .
```

*Note: this module builds on Modules 00–04 — start there first if you
haven't already, especially Module 01's memory layout section and Module
04's map-of-pointers projects, which this module builds on directly.*
