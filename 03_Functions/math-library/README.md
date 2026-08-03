# Project 1 — Math Library

This one is structured as a real, importable **package** (`mathlib`) plus a
separate `main.go` that demonstrates it — your first taste of a multi-file,
multi-package project instead of a single `main.go`.

```
math-library/
├── go.mod              module mathlibdemo
├── main.go             package main — imports and demos mathlib
└── mathlib/
    └── mathlib.go       package mathlib — the actual library
```

## Run It

```bash
cd math-library
go run .
```

## What Each Function Demonstrates

| Function | Concept |
|---|---|
| `Factorial` | Plain recursion, with a base case (`n <= 1`) and an error return for negative input |
| `NewMemoizedFibonacci` | Returns a **closure** that captures a `cache` map — repeated calls to the *same* returned function reuse prior work instead of recomputing |
| `IsPrime` | A `for` loop bounded by `i*i <= n` — not recursion, included as a contrast |
| `GCD` / `LCM` | Recursion again, but via the classic Euclidean algorithm; `LCM` is built by *composing* `GCD` |
| `Sum` | Variadic (`...int`) — works with zero, one, or many arguments |
| `Average` | Variadic + a **named return** (`avg float64, err error`), so the zero-numbers error case reads clearly |
| `Map` / `Filter` / `Reduce` | Higher-order functions — each takes a function as a parameter |

## Case Study: Why `NewMemoizedFibonacci` Returns a Function Instead of Just Being One

A naive recursive Fibonacci — `fib(n) = fib(n-1) + fib(n-2)` with no cache — is
exponential time, because `fib(5)` recomputes `fib(3)` and `fib(2)` from
scratch multiple times over. The fix is memoization: remember answers you've
already computed.

```
┌────────────────────────────────────────────────────────────┐
│              Naive fib(5) — massive redundant work             │
│                                                                    │
│                          fib(5)                                     │
│                        /        \                                     │
│                   fib(4)          fib(3)                                │
│                  /      \        /      \                                 │
│              fib(3)   fib(2)  fib(2)   fib(1)                               │
│              /    \                                                           │
│          fib(2)  fib(1)     ◀── fib(2) and fib(3) are recomputed              │
│                                   from scratch, more than once each               │
└────────────────────────────────────────────────────────────┘
```

The fix needs a `cache` that **persists between calls**. A package-level
global variable would work but pollutes the package with shared mutable
state that every caller secretly shares. Instead, `NewMemoizedFibonacci`
returns a fresh closure, each with its **own private `cache`** — call it twice
and you get two independent memoized calculators that don't interfere with
each other, with no global state at all. This is the same pattern as
`makeCounter` in the guide, applied to something actually useful.

## Try It Yourself
- Add a `Reverse(nums []int) []int` and a `Max(nums ...int) (int, error)`
  (variadic, matching `Sum`'s style)
- Rewrite `Factorial` iteratively (a `for` loop) and compare: for very large
  `n`, which one risks a stack overflow first?
- Add a `Compose(f, g func(int) int) func(int) int` higher-order function that
  returns a new function equivalent to `f(g(x))`
