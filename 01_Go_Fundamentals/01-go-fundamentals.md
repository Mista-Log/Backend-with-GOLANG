# 01. Go Fundamentals

This module is the "how Go actually works" layer underneath the syntax — once
this clicks, everything else you write will make a lot more sense.

---

## 1. History

- **2007** — Design begins at Google, driven by frustration with C++ build times
  and Java's verbosity for the kind of large distributed systems Google runs.
- **Creators**: Robert Griesemer, Rob Pike, Ken Thompson (Thompson co-created Unix
  and the B language, one of C's ancestors — Go's lineage is very "systems
  programming royalty").
- **2009** — Announced publicly, open-sourced.
- **2012** — Go 1.0, the first version with a compatibility promise (code written
  for 1.0 still compiles today).
- **2018** — Go Modules introduced (Go 1.11), solving dependency management.
- **2022** — Generics added (Go 1.18) — the most requested feature in the
  language's history, added carefully and late on purpose.

Go's guiding design principle throughout: **simplicity over cleverness**. The
spec is famously short — you can read the entire language specification in an
afternoon, unlike C++ or Rust's much larger specs.

---

## 2. Compilation

Go is **ahead-of-time (AOT) compiled** straight to native machine code — no
bytecode, no VM, no JIT warm-up.

```
┌─────────────────────────────────────────────────────────────┐
│                    Go Compilation Pipeline                    │
│                                                                 │
│   .go source                                                   │
│       │                                                        │
│       ▼                                                        │
│   lexer/parser  ──▶  builds an Abstract Syntax Tree (AST)      │
│       │                                                        │
│       ▼                                                        │
│   type checker  ──▶  every type resolved & verified at         │
│       │                compile time (this is why Go has        │
│       │                so few runtime type errors)             │
│       ▼                                                        │
│   SSA generation  ──▶  intermediate representation, optimized  │
│       │                                                         │
│       ▼                                                        │
│   machine code  ──▶  native instructions for GOOS/GOARCH        │
│       │                                                         │
│       ▼                                                        │
│   linked static binary  ──▶  ONE file, includes the Go          │
│                                runtime + garbage collector       │
│                                statically linked in — this is    │
│                                why Go binaries "just work" with  │
│                                no separate runtime install       │
└─────────────────────────────────────────────────────────────┘
```

Compare to interpreted/JIT languages:
```
Python:  source ──▶ bytecode ──▶ interpreted line-by-line at runtime
Java:    source ──▶ bytecode ──▶ JVM interprets, then JIT-compiles hot paths
Go:      source ──▶ native machine code, once, ahead of time
```

This is *why* `go run` still feels instant despite being a "compiled" language —
Go's compiler is unusually fast, deliberately engineered for large codebases at
Google where slow C++ builds were the original motivating pain point.

---

## 3. How Go Works (Runtime)

Even though there's no separate VM, every Go binary embeds a small **runtime**
that handles three jobs your code doesn't have to think about:

1. **Garbage collection** — reclaims memory you're no longer using (concurrent,
   low-latency, tuned for Go's goroutine-heavy style).
2. **Goroutine scheduling** — Go's lightweight "green threads" are scheduled by
   the runtime onto a small number of real OS threads (this is how Go programs
   run thousands of concurrent goroutines cheaply).
3. **Memory allocation** — decides whether a value goes on the stack (cheap,
   auto-cleaned) or the heap (needs GC) via **escape analysis** at compile time.

```
┌───────────────────────────────────────────────────────────┐
│                     Goroutines ──▶ OS Threads                │
│                                                                │
│   goroutine  goroutine  goroutine  goroutine  ... (cheap,     │
│      │           │          │          │        ~2KB stack   │
│      └─────┬─────┴────┬─────┴────┬─────┘        each)        │
│             ▼          ▼          ▼                          │
│         OS thread   OS thread  OS thread   (the Go scheduler  │
│                                              multiplexes many  │
│                                              goroutines onto   │
│                                              few OS threads)   │
└───────────────────────────────────────────────────────────┘
```

---

## 4. Memory Layout

Every Go program's memory is split into regions with very different lifetimes:

```
┌────────────────────────────────────────────────────────────┐
│  Stack           │  Grows/shrinks per function call.         │
│                   │  Local variables live here IF the         │
│                   │  compiler proves they never "escape"      │
│                   │  the function (fast — no GC involved).    │
├────────────────────────────────────────────────────────────┤
│  Heap             │  Anything that might outlive its function  │
│                   │  (e.g. returned as a pointer) is placed    │
│                   │  here instead. Managed by the garbage      │
│                   │  collector.                                │
├────────────────────────────────────────────────────────────┤
│  Data segment     │  Package-level vars with an initial value  │
├────────────────────────────────────────────────────────────┤
│  Text/Code segment│  The compiled machine instructions          │
└────────────────────────────────────────────────────────────┘
```

Check what the compiler decided with:
```bash
go build -gcflags="-m" main.go
# will print lines like: "./main.go:10:6: x escapes to heap"
```

You rarely need to think about this day to day, but knowing it exists explains
*why* returning a pointer to a local variable is completely safe in Go (unlike
C, where that's a classic bug) — Go's compiler automatically moves that variable
to the heap for you.

---

## 5. Variables

```go
var name string = "Ibrahim"  // explicit type
var age = 29                  // type inferred from the value
count := 0                    // short declaration — ONLY inside functions

var (
	x = 1
	y = 2
	z = 3
)
```

`:=` is the one you'll use 95% of the time inside functions — but it can't be
used at package level, and it always declares at least one *new* variable on
the left-hand side.

---

## 6. Constants

```go
const Pi = 3.14159
const MaxRetries = 3

// Grouped, auto-incrementing constants via iota — Go's answer to enums:
type Weekday int

const (
	Sunday Weekday = iota // 0
	Monday                 // 1
	Tuesday                // 2
	Wednesday              // 3
)
```

Constants must be knowable at compile time — no function calls, no runtime
values. `iota` resets to 0 at each `const` block and increments by one per line,
which is Go's idiomatic way of building enum-like values without a dedicated
`enum` keyword (Go doesn't have one).

---

## 7. Zero Values

Unlike languages where an uninitialized variable is garbage/undefined, Go
**guarantees every variable a sane default** the moment it's declared:

| Type | Zero Value |
|---|---|
| `int`, `float64` | `0` |
| `bool` | `false` |
| `string` | `""` (empty string, not nil) |
| pointers, slices, maps, channels, funcs, interfaces | `nil` |
| struct | every field set to its own zero value |

```go
var count int      // 0
var name string     // ""
var ready bool       // false
var items []int      // nil (but len(items) == 0 works fine, no crash)
```

This eliminates an entire category of bugs common in C ("uninitialized memory
read") — in Go, "did you forget to initialize this?" simply isn't a question you
have to ask.

---

## 8. Basic Types

```
bool

string

int   int8   int16   int32   int64
uint  uint8  uint16  uint32  uint64  uintptr

byte    // alias for uint8
rune    // alias for int32, represents a Unicode code point

float32  float64
complex64  complex128
```

`int` and `uint` are platform-dependent (32 or 64 bit depending on the target
architecture) — use plain `int` unless you specifically need a guaranteed
width, in which case reach for `int32`/`int64` etc.

---

## 9. Type Conversion

Go has **no implicit conversion** — not even between `int` and `float64`. You
must convert explicitly:

```go
var i int = 42
var f float64 = float64(i)   // explicit — required
var u uint = uint(f)          // explicit — required

// i + f  ← this is a COMPILE ERROR: mismatched types
```

This is another "no silent surprises" design choice — in languages that allow
implicit widening/narrowing, subtle bugs creep in from precision loss you
didn't ask for. Go makes you spell it out every time.

String ⇄ number conversions go through the `strconv` package, not a type
conversion:
```go
n, err := strconv.Atoi("42")     // string -> int
s := strconv.Itoa(42)             // int -> string
f, err := strconv.ParseFloat("3.14", 64)
```

---

## 10. Formatting

The `fmt` package's verbs are worth memorizing early — you'll type these daily:

| Verb | Meaning |
|---|---|
| `%v` | default format for any value |
| `%+v` | like `%v` but includes struct field names |
| `%T` | the Go type of the value |
| `%d` | integer |
| `%f` | float (`%.2f` = 2 decimal places) |
| `%s` | string |
| `%t` | bool |
| `%q` | a quoted string (adds `"..."`) |

```go
fmt.Printf("%v %T\n", 42, 42)        // 42 int
fmt.Printf("%.2f\n", 3.14159)         // 3.14
fmt.Println("plain print, no verbs needed")
s := fmt.Sprintf("age: %d", 29)       // build a string instead of printing
```

---

## 11. Input

```go
import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your name: ")
	name, _ := reader.ReadString('\n')
	fmt.Println("Hello,", name)
}

// Or, for single simple values:
var age int
fmt.Scan(&age)   // note the & — Scan needs a pointer to write into
```

---

## 12. Packages

Every `.go` file belongs to exactly one package, declared at the top. A
directory = a package (all files in the same folder must declare the same
package name). `package main` is special — it's the only kind that can be
compiled into a runnable binary; every other package name produces an
importable library.

```
myapp/
├── main.go        package main
├── utils/
│   └── math.go    package utils
└── models/
    └── user.go    package models
```

```go
import "myapp/utils"

utils.Add(2, 3)
```

---

## 13. Exported Names

Go has no `public`/`private` keywords. Instead, **capitalization is the access
control**:

```go
func Add(a, b int) int { ... }   // Capital A — EXPORTED, visible outside the package
func add(a, b int) int { ... }   // lowercase a — unexported, package-private
```

```
┌───────────────────────────────────────────────────┐
│  Capitalized  →  Exported    →  usable as pkg.Name  │
│  lowercase    →  unexported  →  only inside the      │
│                                  same package         │
└───────────────────────────────────────────────────┘
```

This applies to functions, types, struct fields, and variables/constants alike.
It's a small rule that has a big effect: you can tell a type's entire public
API just by scanning for capital letters, no separate keyword needed.

---

## 14. Scope

Go uses classic **lexical (block) scoping** — a variable is visible from its
declaration to the end of the enclosing `{ }` block:

```go
func demo() {
	x := 1
	{
		y := 2
		fmt.Println(x, y) // both visible here
	}
	fmt.Println(x) // y is now out of scope — compile error if referenced
}
```

Package-level scope sits above all of this — anything declared outside a
function (`var`, `const`, `func`, `type`) is visible throughout the whole
package, and additionally outside it if exported.

```
┌───────────────────────────────────────────┐
│  Package scope                              │
│   ┌──────────────────────────────────────┐  │
│   │  Function scope                        │  │
│   │   ┌─────────────────────────────────┐  │  │
│   │   │  Block scope (if/for/{ })        │  │  │
│   │   └─────────────────────────────────┘  │  │
│   └──────────────────────────────────────┘  │
└───────────────────────────────────────────┘
```

One classic Go gotcha: a shadowed variable inside a nested `if err := ...; err != nil`
creates a *new* `err` local to that block — easy to miss when you expect it to
reuse the outer one.

---

Onto the exercises — these are deliberately small so you can focus entirely on
variables, types, conversion, and formatting without any extra complexity.
