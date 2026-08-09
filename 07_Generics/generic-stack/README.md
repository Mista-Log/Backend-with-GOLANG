# Project 2 — Generic Stack

```bash
cd generic-stack
go run main.go
```

## What's Demonstrated Here

- **`Stack[T any]`** — same shape as Generic Queue, reversed (LIFO instead
  of FIFO): `Pop`/`Peek` look at the *end* of `items` instead of the front.
- **A custom constraint (`Ordered`)**, used only where it's actually needed —
  `Stack[T any]` itself stays maximally permissive, but `MaxInStack[T Ordered]`
  narrows the constraint specifically because *it* needs `>` to work.
- **`IsBalanced`** — a genuinely practical algorithm (matching brackets)
  built directly on `Stack[rune]`, not a toy example. This is the kind of
  problem a stack exists to solve: "the most recent unmatched open bracket
  must be the next one closed" is exactly LIFO order.

```
┌──────────────────────────────────────────────────────────┐
│   Stack[T any]        — no constraint beyond `any`               │
│     Push, Pop, Peek     never compare T values                     │
│                                                                        │
│   MaxInStack[T Ordered](s *Stack[T])  — narrower constraint             │
│     needs  item > max  to compile, which requires Ordered                 │
│                                                                                │
│   Same underlying Stack[T] works with BOTH — the constraint lives          │
│   on whichever function actually needs it, not on the data structure         │
│   itself.                                                                        │
└──────────────────────────────────────────────────────────┘
```

## Case Study: Why `MaxInStack` Is a Standalone Function, Not a Method

The tempting shape would be `func (s *Stack[T]) Max() (T, bool)` — but Go's
generics deliberately don't allow a method to narrow its receiver's type
parameter to a stricter constraint than the type itself declared. `Stack[T]`
committed to `T any` when it was defined; a method on `*Stack[T]` is stuck
with that same unconstrained `T` forever, even if the method body would only
ever need `Ordered`. The workaround — and it's a common, accepted one in
real Go code — is exactly what this project does: write `Max` as a
**standalone function** that *takes* a `*Stack[T]` and adds its own,
narrower constraint on `T`:

```go
func MaxInStack[T Ordered](s *Stack[T]) (T, bool) { ... }
```

This is a genuinely useful pattern to recognize: **keep container types
(`Stack`, `Queue`, `Cache`) as unconstrained as they can be, and push any
narrower requirements onto standalone functions that operate on them.** It
keeps the container reusable for the widest possible range of element types,
while still letting you write fully type-safe operations for the subset of
cases that need more.

## Try It Yourself
- Add a `Reverse()` method that reverses the stack in place (a good exercise
  in swapping elements by index without needing any constraint at all)
- Extend `IsBalanced` to also validate a simple arithmetic expression's
  parentheses AND quote-matching in one pass (you'll likely want a second,
  separate `Stack[rune]` or a small struct to track both kinds of state)
- Write `MinInStack`, then notice how much of it is identical to
  `MaxInStack` — a good motivator for the next step: a single
  `ExtremeInStack[T Ordered](s *Stack[T], less func(a, b T) bool) (T, bool)`
  that takes a comparison function and covers both cases
