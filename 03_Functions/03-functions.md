# 03. Functions

Functions are where Go's design philosophy shows up most clearly: no default
arguments, no overloading, no exceptions — but genuinely excellent support for
multiple return values, closures, and treating functions as ordinary values.

---

## 1. Function Declaration

```go
func add(a int, b int) int {
	return a + b
}

// Consecutive parameters of the same type can share one type annotation:
func add(a, b int) int {
	return a + b
}

// No return value at all:
func greet(name string) {
	fmt.Println("Hello,", name)
}
```

```
┌─────────────────────────────────────────────────┐
│  func   name   ( params )   returnType   { body }  │
│   │       │         │            │           │      │
│  keyword  │    each param:   what comes   the code    │
│         chosen    name Type    back out    that runs    │
│           by you                                          │
└─────────────────────────────────────────────────┘
```

Go has **no function overloading** — you can't declare `add(int, int)` and
`add(float64, float64)` side by side under the same name. This is a deliberate
simplicity trade-off; the usual workarounds are a differently-named function,
or (later) generics.

---

## 2. Multiple Returns

This is one of Go's most-used features, seen constantly in the standard
library:

```go
func divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}

result, err := divide(10, 2)
if err != nil {
	// handle it
}
```

Multiple returns are also how Go implements the "comma ok" idiom for safe
lookups and type assertions:
```go
value, ok := myMap["key"]         // ok is false if the key doesn't exist
n, err := strconv.Atoi("42")       // err is non-nil if parsing failed
v, ok := someInterface.(string)    // ok is false if the assertion fails
```

---

## 3. Named Returns

You can name your return values in the function signature — they're declared
as variables (initialized to their zero value) at the top of the function, and
a bare `return` sends back whatever they currently hold.

```go
func divide(a, b float64) (result float64, err error) {
	if b == 0 {
		err = errors.New("division by zero")
		return // returns (0, err) — result is still its zero value
	}
	result = a / b
	return // returns (result, nil)
}
```

Named returns are especially useful combined with `defer`, since a deferred
function can modify a named return value *after* the `return` statement has
run but *before* the function actually exits:

```go
func mayPanic() (result int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from panic: %v", r)
		}
	}()
	// ... risky code that might panic ...
	return 42, nil
}
```

Use named returns sparingly for short functions where the names add real
clarity (e.g., documenting what each return value *means*) — for long
functions they can make the flow harder to follow, since a bare `return`
hides exactly what's being sent back.

---

## 4. Variadic Functions

A parameter prefixed with `...` accepts zero or more arguments, collected
into a slice inside the function:

```go
func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

sum()              // 0
sum(1, 2, 3)         // 6
sum([]int{4, 5, 6}...)  // 15 — spread a slice into variadic args with `...`
```

`fmt.Println(a, b, c ...interface{})` is itself variadic — this is exactly why
you can pass it any number of arguments of any type. A variadic parameter must
be the **last** parameter in the list, and there can only be one.

```
┌───────────────────────────────────────────────────────┐
│  func sum(nums ...int) int                                │
│                                                              │
│  call: sum(1, 2, 3)                                          │
│              │                                                 │
│              ▼                                                  │
│  inside the function, nums is just: []int{1, 2, 3}                │
└───────────────────────────────────────────────────────┘
```

---

## 5. Anonymous Functions

A function literal with no name, usable immediately or stored in a variable:

```go
square := func(x int) int {
	return x * x
}
fmt.Println(square(5)) // 25

// Immediately Invoked Function Expression (IIFE) — defines AND calls at once:
result := func(x, y int) int {
	return x + y
}(3, 4) // result == 7
```

Anonymous functions are the building block for closures and for passing
one-off logic into higher-order functions (`sort.Slice`, goroutines launched
with `go func() {...}()`, etc).

---

## 6. Closures

A closure is a function that **captures variables from its surrounding
scope**, and keeps a live reference to them — not a copy — even after the
outer function has returned.

```go
func makeCounter() func() int {
	count := 0
	return func() int {
		count++
		return count
	}
}

counter := makeCounter()
fmt.Println(counter()) // 1
fmt.Println(counter()) // 2
fmt.Println(counter()) // 3

counter2 := makeCounter() // a completely separate `count` — closures don't share state
fmt.Println(counter2())   // 1
```

```
┌────────────────────────────────────────────────────────┐
│  makeCounter() call #1                                     │
│      count := 0   ──┐                                        │
│      returns func   │  the returned function keeps a LIVE     │
│                      │  reference to this specific `count`     │
│                      ▼  variable, even though makeCounter()     │
│                    [count: 0] ◀── lives on the heap now,          │
│                                    NOT the stack, because it       │
│                                    escaped its original function    │
│                                                                        │
│  makeCounter() call #2  ──▶  a totally separate [count: 0]              │
│                                (its own closure, own memory)              │
└────────────────────────────────────────────────────────┘
```

This connects directly back to **Module 01's memory layout section**: the
compiler's escape analysis notices `count` is still reachable after
`makeCounter` returns (via the closure), so it automatically moves `count` to
the heap instead of the stack — no manual memory management needed on your
part.

**Classic closure-in-a-loop gotcha** (fixed in Go 1.22+, but good to know):
```go
// Go 1.21 and earlier — ALL closures captured the SAME shared `i`:
funcs := []func(){}
for i := 0; i < 3; i++ {
	funcs = append(funcs, func() { fmt.Println(i) })
}
// calling each of these used to print 3, 3, 3 (the final value of i)

// Go 1.22+ — the loop variable is now scoped PER ITERATION,
// so this same code correctly prints 0, 1, 2.
```

---

## 7. Recursion

A function that calls itself, with a **base case** to stop the recursion:

```go
func factorial(n int) int {
	if n <= 1 {
		return 1 // base case — stops the recursion
	}
	return n * factorial(n-1) // recursive case
}

factorial(5) // 5 * 4 * 3 * 2 * 1 = 120
```

```
┌──────────────────────────────────────────────────────┐
│  factorial(5)                                            │
│    = 5 * factorial(4)                                      │
│           = 4 * factorial(3)                                 │
│                  = 3 * factorial(2)                            │
│                         = 2 * factorial(1)                       │
│                                = 1  ◀── base case, unwinding      │
│                                        starts here and multiplies  │
│                                        back UP the call stack        │
└──────────────────────────────────────────────────────┘
```

Every recursive call adds a frame to the call stack — deep recursion (tens of
thousands of levels) can exhaust it, which is why Go programmers often prefer
an iterative (`for` loop) version for anything that might recurse very deeply,
reaching for recursion mainly when the problem is naturally tree/graph-shaped
(traversing directory trees, JSON structures, parsing expressions).

---

## 8. Higher-Order Functions

A function that takes another function as a parameter, returns one, or both.

```go
// Takes a function as a parameter:
func apply(nums []int, f func(int) int) []int {
	result := make([]int, len(nums))
	for i, n := range nums {
		result[i] = f(n)
	}
	return result
}

doubled := apply([]int{1, 2, 3}, func(n int) int { return n * 2 })
// doubled == [2, 4, 6]

// Returns a function (this is also a closure, see section 6):
func multiplier(factor int) func(int) int {
	return func(n int) int { return n * factor }
}

double := multiplier(2)
triple := multiplier(3)
fmt.Println(double(5), triple(5)) // 10 15
```

The standard library leans on this constantly — `sort.Slice(data, less func(i, j int) bool)`,
`http.HandleFunc(pattern string, handler func(...))`, and every `context`-based
cancellation callback all take a function as an argument.

```
┌────────────────────────────────────────────────────────┐
│                 Function as a first-class value           │
│                                                              │
│   func(int) int   ──▶  can be:                               │
│                          - stored in a variable                │
│                          - passed as an argument                 │
│                          - returned from another function          │
│                          - stored in a slice/map (a dispatch        │
│                            table — see the Calculator API project)   │
└────────────────────────────────────────────────────────┘
```

---

Onto the projects — the Math Library leans hard on recursion and variadic
functions; the Calculator API leans on higher-order functions and closures via
a dispatch-table design.
