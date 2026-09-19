# Go for Beginners — Module 27: Interview Preparation

## Contents

1. **[27-interview-prep.md](./27-interview-prep.md)** — DSA in Go (which
   standard-library primitives map to which data structures), LeetCode
   patterns with their Go-specific gotchas, the concurrency questions Go
   interviews ask far more than other languages do, system design
   interview structure (building on Module 26), and behavioral prep using
   STAR.

2. **[go-internals-reference.md](./go-internals-reference.md)** — The five
   internals topics that separate "writes Go" from "understands Go":
   memory (escape analysis, GC), channels (the behavior table worth
   memorizing), interfaces (including the nil-interface trap), maps
   (buckets, randomized iteration, why `&m[key]` doesn't compile), and
   slices (the three-word header, the aliasing trap, growth behavior).

3. **[interview-practice/](./interview-practice)** — Runnable code:
   proofs of all five internals gotchas, four classic concurrency
   problems, and LeetCode patterns in idiomatic Go — with a test suite
   including one test written specifically to catch the backtracking
   slice-aliasing bug.

## Setup

```bash
cd interview-practice
go run ./cmd/demo
go test ./...
```

No external dependencies.

---

## This Is the Last Module

Twenty-eight modules, 00 through 27 — from `go version` through
fundamentals, concurrency, HTTP, databases, authentication, microservices,
distributed systems, a full fintech backend, performance, cloud
deployment, system design, and now interview preparation.

The most valuable thing here isn't any single module. It's that when an
interviewer asks you to design a payment system, a chat server, or a
worker pool, you've already built one — and can talk about the real
trade-offs you hit, because you hit them.
