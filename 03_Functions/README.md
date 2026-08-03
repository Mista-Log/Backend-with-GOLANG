# Go for Beginners — Module 03: Functions

## Contents

1. **[03-functions.md](./03-functions.md)** — Function declaration, multiple
   returns, named returns (and their `defer`/`recover` synergy), variadic
   functions, anonymous functions, closures (including the pre-1.22 loop
   variable gotcha), recursion, and higher-order functions. Diagrams included
   throughout.

2. **[math-library/](./math-library)** — A real importable package (`mathlib`)
   plus a demo `main.go`. Covers recursion (`Factorial`, `GCD`/`LCM`), a
   memoized-Fibonacci closure, variadic functions with a named-return error
   case (`Average`), and `Map`/`Filter`/`Reduce` as higher-order functions.

3. **[calculator-api/](./calculator-api)** — A small HTTP JSON API where every
   endpoint is powered by a function concept: a dispatch-table map of
   operations, higher-order middleware functions (`withLogging`,
   `withRequestCount`), a closure-captured request counter (with a case study
   on why it needs a mutex), a variadic sum endpoint, and a recursive power
   endpoint.

## Suggested Order

```
Functions guide ──▶ Math Library ──▶ Calculator API
```

Math Library keeps everything in a single process, calling functions
directly. Calculator API takes the same ideas (dispatch tables, closures,
recursion, variadic functions) and exposes them over HTTP, where the
higher-order "wrap a handler in another handler" pattern really earns its
keep.

## Quick Reference: Running Each Project

```bash
# Math Library
cd math-library && go run .

# Calculator API
cd calculator-api && go run .
# then, in another terminal:
curl "http://localhost:8080/calculate?op=add&a=4&b=5"
```

*Note: this module builds on Modules 00–02 — start there first if you
haven't already.*
