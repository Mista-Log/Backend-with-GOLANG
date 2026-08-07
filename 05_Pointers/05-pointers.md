# 05. Pointers

This module goes deep on something you've already been using without much
explanation since Module 01's memory layout section and Module 04's
map-of-pointers projects — now it's time to understand exactly what a pointer
*is*, and why Go's design around them is unusually safe compared to C.

---

## 1. Memory

Every running program's variables live somewhere in memory — a giant array of
numbered byte-slots. When you declare `x := 42`, Go picks a spot in memory,
stores `42` there, and lets you refer to that spot by the name `x`.

```
┌───────────────────────────────────────────────────┐
│   Memory (simplified)                                   │
│                                                            │
│   address:  0x1040   0x1041   0x1042   0x1043   ...          │
│   value:      42        ?        ?        ?                    │
│               ▲                                                   │
│               └── this is where `x := 42` actually lives            │
└───────────────────────────────────────────────────┘
```

A **pointer** is just a variable whose *value* is a memory address instead of
an ordinary value like a number or string — "the location where some other
value lives," not the value itself.

---

## 2. Address

Every variable has an address, retrievable with `&`:

```go
x := 42
fmt.Println(x)   // 42        — the value
fmt.Println(&x)   // 0xc0000... — the ADDRESS where x is stored

p := &x            // p is now a pointer TO x — its type is *int
fmt.Println(p)       // same address as &x
fmt.Printf("%T\n", p) // *int
```

```
┌────────────────────────────────────────┐
│   x  ──────▶  [ 42 ]   at address 0xc0000     │
│                  ▲                              │
│   p  ────────────┘   p's VALUE is that address    │
│  (p's type is *int — "pointer to int")               │
└────────────────────────────────────────┘
```

---

## 3. Dereference

Given a pointer, `*p` gets you back to the value it points at — this is
"following" the pointer, called **dereferencing**.

```go
x := 42
p := &x

fmt.Println(*p) // 42 — dereference: "the value at the address p holds"

*p = 100          // dereference on the LEFT side of = writes THROUGH the pointer
fmt.Println(x)     // 100 — x itself changed, because p pointed straight at it
```

```
┌────────────────────────────────────────────┐
│   &x    "address of x"     →  a pointer          │
│   *p    "value at p"        →  dereference          │
│                                                          │
│   x := 42                                                  │
│   p := &x        p now holds x's address                     │
│   *p = 100        writes 100 into THAT address                 │
│   x is now 100    because that address IS x                      │
└────────────────────────────────────────────┘
```

The zero value of any pointer type is `nil` — dereferencing a `nil` pointer
(`*p` when `p == nil`) panics at runtime, since there's no address to follow:
```go
var p *int
fmt.Println(*p) // panic: runtime error: invalid memory address or nil pointer dereference
```

---

## 4. Pointer Receivers

Methods in Go can be defined on either a value or a pointer of a type — this
choice is one of the most consequential decisions you'll make about any
struct.

```go
type Counter struct {
	count int
}

// POINTER receiver — the method operates on the ACTUAL struct, through
// its address, so mutations are visible to the caller afterward.
func (c *Counter) Increment() {
	c.count++
}

c := Counter{}
c.Increment() // Go automatically takes &c for you here — same as (&c).Increment()
fmt.Println(c.count) // 1
```

```
┌────────────────────────────────────────────────────┐
│   c := Counter{count: 0}                                 │
│                                                                │
│   c.Increment()                                                 │
│        │                                                          │
│        ▼                                                            │
│   Go rewrites this as (&c).Increment() automatically                  │
│        │                                                                │
│        ▼                                                                  │
│   Increment receives a POINTER to the real c — c.count++ writes           │
│   directly into c's own memory, not a copy                                  │
└────────────────────────────────────────────────────┘
```

---

## 5. Value Receivers

```go
// VALUE receiver — the method gets a COPY of the struct. Any change it
// makes disappears the moment the method returns.
func (c Counter) IncrementBroken() {
	c.count++ // modifies the COPY only — caller's c is untouched
}

c := Counter{}
c.IncrementBroken()
fmt.Println(c.count) // still 0!
```

**Rule of thumb:** if a method needs to mutate the receiver, or the struct is
large enough that copying it on every call would be wasteful, use a pointer
receiver. If the method only reads data and the struct is small (a couple of
ints/strings), a value receiver is fine and slightly simpler. **In practice,
most real Go code picks pointer receivers for every method on a given type
once even one method needs mutation** — Go requires all methods on a type to
use the same receiver kind consistently in idiomatic code, even though the
compiler doesn't strictly force this.

```
┌──────────────────────────────────────────────────────┐
│              Pointer receiver   vs   Value receiver         │
│                                                                  │
│   func (c *Counter) Inc()        func (c Counter) Inc()           │
│                                                                        │
│   Receives: address of c            Receives: a full COPY of c           │
│   Mutations: visible to caller       Mutations: invisible, lost            │
│   Cost: cheap (just an address)       Cost: copies the whole struct           │
└──────────────────────────────────────────────────────┘
```

---

## 6. Escape Analysis

This is the compiler decision, introduced back in Module 01, that decides
whether a value lives on the **stack** (cheap, auto-cleaned when the function
returns) or the **heap** (managed by the garbage collector) — and pointers
are exactly what trigger it.

```go
func newCounter() *Counter {
	c := Counter{} // looks local...
	return &c        // ...but its ADDRESS escapes the function via the return!
}
```

In a language like C, returning `&c` here would be a serious bug — `c` was a
stack variable, and the stack frame it lived in is gone the instant
`newCounter` returns, leaving `&c` pointing at garbage. **Go's compiler
prevents this automatically**: it performs escape analysis, notices `c`'s
address is returned (so it must outlive the function), and simply allocates
`c` on the **heap** instead of the stack in the first place. The garbage
collector then reclaims it once nothing points to it anymore.

```
┌────────────────────────────────────────────────────────┐
│   Compiler asks, for every local variable: "does its           │
│   address ever leave this function?"                              │
│                                                                        │
│   NO  → stays on the STACK                                             │
│           (fast: just moving the stack pointer, freed                    │
│            automatically when the function returns)                        │
│                                                                                │
│   YES → moves to the HEAP                                                      │
│           (slightly slower: a real allocation, tracked                           │
│            and eventually freed by the garbage collector)                          │
└────────────────────────────────────────────────────────┘
```

Check the compiler's actual decisions for any file:
```bash
go build -gcflags="-m" main.go
# ./main.go:5:2: moved to heap: c
```

---

## 7. Heap

The heap is memory that isn't tied to any single function call's lifetime —
values here persist until nothing refers to them anymore, at which point
Go's garbage collector reclaims the space. Every value reachable through a
pointer that "escaped" (per the section above) ends up here.

Heap allocation is more expensive than stack allocation (real bookkeeping is
involved, and the GC has to eventually walk and reclaim it), but it's what
makes returning pointers to local data completely safe in Go — you never
have to reason about "does this address still point at something valid?"
the way you would in C.

---

## 8. Stack

The stack is memory scoped to the current chain of function calls — each
function call gets its own **stack frame** holding its local variables,
pushed on entry and popped (instantly freed, no GC involved) on return.

```
┌───────────────────────────────────────────────┐
│   func c()  {  x := 1;  ... }    ◀── stack frame 3     │
│   func b()  {  y := 2;  c()  }    ◀── stack frame 2       │
│   func a()  {  z := 3;  b()  }     ◀── stack frame 1         │
│                                                                    │
│   Call order:  a → b → c   (frames pushed downward)                  │
│   Return order: c → b → a   (frames popped in reverse — LIFO,          │
│                               same shape as defer's LIFO order            │
│                               from Module 02)                                │
└───────────────────────────────────────────────┘
```

Local variables that **don't** escape (per escape analysis) live here, and
their memory is reclaimed automatically — instantly, with zero garbage
collector involvement — the moment their function returns. This is why
"prefer values that don't need to outlive their function" is a genuine
performance habit in Go: it's not just about avoiding pointers for their own
sake, it's about giving the compiler the chance to keep things on the cheap,
GC-free stack.

---

Onto the project — a Mini Database is a natural fit for pointers, since every
record needs to be mutable *in place* through whatever's holding a reference
to it, exactly like `Increment` above but at the scale of a real dataset.
