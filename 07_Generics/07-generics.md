# 07. Generics

Generics arrived in **Go 1.18** (2022) — the single most requested feature in
the language's history, and deliberately the last major piece added, after
years of the Go team looking for a design that stayed true to Go's
simplicity goals. This module covers writing code that works across many
types without giving up compile-time type safety or falling back to `any`
everywhere.

---

## 1. Type Parameters

A type parameter is a placeholder type, declared in square brackets right
after a function or type's name, that gets filled in with a real type at the
call site.

```go
func Max[T int | float64](a, b T) T {
	if a > b {
		return a
	}
	return b
}

fmt.Println(Max(3, 7))       // T inferred as int → 7
fmt.Println(Max(2.5, 1.1))    // T inferred as float64 → 2.5
fmt.Println(Max[int](3, 7))    // T given explicitly — rarely needed, inference usually works
```

```
┌───────────────────────────────────────────────────────┐
│  func   Max   [ T int | float64 ]   (a, b T)   T   { ... }   │
│           │        │                    │           │           │
│         name   type parameter,      params use   return       │
│                constrained to        the type      type also    │
│                int OR float64         parameter T    uses T        │
└───────────────────────────────────────────────────────┘
```

Before generics, this problem had two bad options: write `MaxInt`, `MaxFloat64`,
`MaxString`... one function per type (repetitive), or write `func Max(a, b any) any`
and type-assert inside (loses compile-time safety, and callers lose it too —
`Max(3, 7)` would return `any`, not `int`). Generics give you one function,
one implementation, and the compiler still knows the exact type at every
call site.

---

## 2. Constraints

A **constraint** is the set of types allowed to fill in a type parameter —
written where you'd normally see a single type, but as an interface listing
which types (or which methods) qualify.

```go
type Number interface {
	int | int64 | float64
}

func Sum[T Number](nums []T) T {
	var total T
	for _, n := range nums {
		total += n
	}
	return total
}

Sum([]int{1, 2, 3})           // T = int
Sum([]float64{1.5, 2.5})       // T = float64
// Sum([]string{"a", "b"})      // COMPILE ERROR — string isn't in the Number constraint
```

Constraints are just **interfaces** — the same concept from Module 06,
extended to also describe "which underlying types are allowed," not only
"which methods are required."

```
┌────────────────────────────────────────────────────────┐
│   Constraint = an interface listing allowed types             │
│                                                                    │
│   type Number interface {                                          │
│       int | int64 | float64     ◀── a type SET, using | to           │
│   }                                  union multiple allowed types       │
│                                                                                │
│   func Sum[T Number](nums []T) T { ... }                                        │
│                                                                                       │
│   Only types matching ONE of these can be used as T — the compiler                     │
│   REJECTS Sum([]string{...}) before the program ever runs.                                │
└────────────────────────────────────────────────────────┘
```

---

## 3. `comparable`

`comparable` is a **built-in constraint** matching any type that supports
`==` and `!=` — numbers, strings, bools, pointers, and structs made entirely
of comparable fields (recall Module 04's map-internals section: this is the
exact same rule as "what can be a map key").

```go
func Contains[T comparable](items []T, target T) bool {
	for _, item := range items {
		if item == target { // only legal because T is constrained to comparable
			return true
		}
	}
	return false
}

Contains([]int{1, 2, 3}, 2)          // true
Contains([]string{"a", "b"}, "c")     // false
```

Without `comparable`, `item == target` wouldn't compile at all — `==` isn't
defined for every possible type (slices and maps, for instance, can't be
compared with `==`), so the compiler needs the constraint to know the
operation is safe for whatever `T` ends up being.

---

## 4. `any`

You met `any` back in Module 06 as the empty interface — in a generics
context, it means **"no constraint at all, literally any type is allowed"**:

```go
func Print[T any](value T) {
	fmt.Println(value)
}

Print(42)
Print("hello")
Print(Circle{Radius: 5})
```

The trade-off is exactly what you'd expect: with `T any`, you can't do
anything type-specific inside the function (no `+`, no `==`, no field
access) — only operations that work identically for every possible type,
like printing, storing in a slice, or passing along to another `any`-typed
place. Reach for a narrower constraint (`comparable`, `Number`, or a custom
one) the moment you need the function to actually *do* something with the
value beyond just holding onto it.

---

## 5. Custom Constraints

You can build constraints combining specific types, `~underlying-type`
(matches any type whose *underlying* type is that one, not just the exact
type itself), and even required methods:

```go
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~float32 | ~float64 | ~string
}

func Max[T Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

type Priority int // a NAMED type whose underlying type is int

var p1, p2 Priority = 5, 3
Max(p1, p2) // works — Priority's underlying type (int) matches ~int
```

Without the `~`, `Max[Priority]` would be rejected — a constraint listing
plain `int` only matches the exact type `int`, not `Priority` (even though
`Priority` is "just an int" underneath). This distinction matters a lot in
practice, since named types built on top of a basic type (`type UserID int`,
`type Celsius float64`) are extremely common in real Go code.

Constraints can also require methods, exactly like a normal interface:
```go
type Stringer interface {
	String() string
}

func Join[T Stringer](items []T, sep string) string {
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = item.String()
	}
	return strings.Join(parts, sep)
}
```

The standard library ships a ready-made `Ordered` constraint (and more) in
the `cmp` package (Go 1.21+) and `golang.org/x/exp/constraints` — worth
knowing they exist before hand-rolling your own for common cases.

---

## 6. Generic Data Structures

This is where generics earn their keep the most: a data structure written
**once**, usable with any element type, with zero runtime type assertions
needed by callers.

```go
type Stack[T any] struct {
	items []T
}

func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}

func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	last := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return last, true
}

intStack := &Stack[int]{}
intStack.Push(1)
intStack.Push(2)
val, ok := intStack.Pop() // val is an int, ok is true — no type assertion needed anywhere

stringStack := &Stack[string]{}
stringStack.Push("hello")
```

```
┌────────────────────────────────────────────────────────┐
│   type Stack[T any] struct { items []T }                       │
│                                                                     │
│   Stack[int]     →  items is []int     →  Push/Pop deal in int      │
│   Stack[string]  →  items is []string  →  Push/Pop deal in string     │
│                                                                            │
│   ONE implementation, MANY concrete types — each fully type-checked,        │
│   with no `any` and no type assertions anywhere in the calling code.          │
└────────────────────────────────────────────────────────┘
```

**Before generics**, the honest options for a reusable stack were: write it
once per type you need (`IntStack`, `StringStack`, ...), or write it with
`any` elements and force every caller to type-assert on `Pop()`. Generic
data structures are the direct replacement for both — this is also exactly
why Go's standard library added a `container/list`-style `slices` and `maps`
package (Go 1.21+) built on generics, replacing a lot of hand-rolled
per-type utility code across the ecosystem.

**One notable limitation worth knowing:** methods themselves cannot
introduce *new* type parameters beyond what the type already declares — you
can write `func (s *Stack[T]) Push(item T)`, but you can't give `Push` its
own separate type parameter unrelated to `T`. Any additional generic-ness
has to live on a standalone function instead.

---

Onto the projects — Generic Queue, Generic Stack, and Generic Cache are all
data structures deliberately chosen because each one needs slightly
different constraints: a plain `[T any]` for the Queue and Stack (they never
compare elements), and `comparable` for the Cache (it needs its keys to work
in a map).
