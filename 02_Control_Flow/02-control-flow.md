# 02. Control Flow

Go deliberately has **one loop keyword** (`for`) and no ternary operator — fewer
ways to do the same thing, which is very much on-brand for the language. This
module covers every branching and looping construct you have.

---

## 1. `if`

```go
if age >= 18 {
	fmt.Println("adult")
} else if age >= 13 {
	fmt.Println("teen")
} else {
	fmt.Println("child")
}
```

No parentheses required around the condition — Go's formatter (`gofmt`) will
actually remove them if you add them out of habit.

**The distinctive Go feature: an "init statement" scoped to the if/else chain.**
```go
if err := doSomething(); err != nil {
	fmt.Println("failed:", err)
} else {
	// err is still in scope here too — it's just always false-path guaranteed
}
// err does NOT exist out here — it was scoped to the if/else block only
```
This `if <init>; <condition>` pattern is everywhere in idiomatic Go — it keeps a
variable you only need for one check from leaking into the rest of the function.

```
┌───────────────────────────────────────────────────┐
│  if val, ok := m["key"]; ok {                        │
│      // val and ok both scoped to this if/else chain  │
│  }                                                     │
│  // neither val nor ok exist here                      │
└───────────────────────────────────────────────────┘
```

---

## 2. `switch`

Go's `switch` is far more flexible than C/Java's — **no fallthrough by default**
(each case implicitly breaks), and cases can be arbitrary expressions, not just
constants.

```go
switch day {
case "Sat", "Sun":
	fmt.Println("weekend")
default:
	fmt.Println("weekday")
}

// Condition-less switch — a clean replacement for long if/else chains:
switch {
case score >= 90:
	fmt.Println("A")
case score >= 80:
	fmt.Println("B")
default:
	fmt.Println("C or below")
}

// Type switch — checks the dynamic type inside an interface value:
var x interface{} = 42
switch v := x.(type) {
case int:
	fmt.Println("int:", v)
case string:
	fmt.Println("string:", v)
default:
	fmt.Println("other type")
}
```

Want old C-style fallthrough behavior? Opt in explicitly:
```go
switch n := 1; n {
case 1:
	fmt.Println("one")
	fallthrough
case 2:
	fmt.Println("also prints — fallthrough was explicit")
}
```

---

## 3. `for`

Go has exactly one looping keyword, but four shapes:

```go
for i := 0; i < 5; i++ {        // classic three-part loop
	fmt.Println(i)
}

for count < 10 {                 // "while" loop — just drop two of the three parts
	count++
}

for {                              // infinite loop — use break to exit
	if done {
		break
	}
}

for i := range 5 {                // range over an integer (Go 1.22+) — 0,1,2,3,4
	fmt.Println(i)
}
```

```
┌──────────────────────────────────────────────────────┐
│           for init; condition; post { }                  │
│                                                            │
│   init  ──▶  condition? ──no──▶ exit loop                 │
│                 │yes                                       │
│                 ▼                                          │
│              body runs                                      │
│                 │                                            │
│                 ▼                                            │
│               post  ──▶ back to condition check              │
└──────────────────────────────────────────────────────┘
```

---

## 4. `range`

`range` iterates over slices, arrays, maps, strings, channels, and (since Go
1.22) plain integers and functions.

```go
nums := []int{10, 20, 30}
for i, v := range nums {
	fmt.Println(i, v)   // index, value
}

m := map[string]int{"a": 1, "b": 2}
for k, v := range m {
	fmt.Println(k, v)   // NOTE: map iteration order is randomized on purpose
}

for i, r := range "héllo" {   // iterates RUNES (Unicode code points), not bytes
	fmt.Println(i, string(r))
}

for _, v := range nums {      // blank identifier `_` to discard the index
	fmt.Println(v)
}
```

**Common gotcha:** ranging over a string gives you byte *indices* but full
*rune* values — because multi-byte UTF-8 characters mean the index can jump by
more than 1 between iterations. This is Go being correct about Unicode by
default, but it surprises people coming from ASCII-only languages.

---

## 5. Labels

A label lets `break`/`continue` target an **outer** loop from inside a nested
one — Go has no other way to "break twice."

```go
outer:
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if j == 1 {
				continue outer   // skip to the NEXT i, not just next j
			}
			fmt.Println(i, j)
		}
	}
```

```
┌─────────────────────────────────────────────────┐
│  outer: for i ...                                   │
│           for j ...                                  │
│             continue outer  ──▶ jumps all the way    │
│                                  back to the outer     │
│                                  loop's next iteration, │
│                                  skipping the rest of   │
│                                  the inner loop entirely │
└─────────────────────────────────────────────────┘
```

---

## 6. `break`

Exits the nearest enclosing `for`, `switch`, or `select` — or, with a label, a
named outer one.

```go
for i := 0; i < 10; i++ {
	if i == 5 {
		break   // stops the loop entirely at i == 5
	}
	fmt.Println(i)
}
```

Careful: inside a `switch` that's inside a `for`, a bare `break` exits the
**switch**, not the loop — this is a classic beginner trip-up coming from
languages where `break` always means "exit the loop."

```go
for i := 0; i < 3; i++ {
	switch i {
	case 1:
		break // only breaks the switch — the for loop keeps going!
	}
	fmt.Println("i is", i)
}
```

---

## 7. `continue`

Skips the rest of the current iteration and jumps to the loop's next one.

```go
for i := 0; i < 10; i++ {
	if i%2 != 0 {
		continue // skip odd numbers
	}
	fmt.Println(i) // only evens print: 0 2 4 6 8
}
```

---

## 8. `goto`

Jumps directly to a labeled line in the same function. Go allows it, but it's
rare in practice — almost always a `for`/`if`/labeled-break can express the same
logic more clearly.

```go
i := 0
loop:
	if i < 3 {
		fmt.Println(i)
		i++
		goto loop
	}
```

Rules: you can't jump *into* a block you weren't already in (e.g., into the
middle of a `for` loop from outside it), and you can't skip over a variable
declaration into its scope. The one place `goto` shows up somewhat legitimately
in real code is centralizing cleanup logic in older C-style code — but in Go,
**`defer` (next section) replaces that use case almost entirely.**

---

## 9. `defer` Basics

`defer` schedules a function call to run **right before the enclosing function
returns** — regardless of *how* it returns (normal return, or a `panic`).

```go
func readFile() {
	f, _ := os.Open("data.txt")
	defer f.Close()   // guaranteed to run when readFile() returns, no matter what

	// ... use f here ...
}   // f.Close() runs automatically right here
```

**Three rules worth internalizing:**

1. **Arguments are evaluated immediately, but the call happens later:**
   ```go
   i := 1
   defer fmt.Println("deferred:", i)  // captures i=1 RIGHT NOW
   i = 2
   fmt.Println("immediate:", i)        // prints 2
   // ... function continues, then at return: prints "deferred: 1"
   ```

2. **Multiple defers run in LIFO order** (last deferred, first executed) — like
   a stack:
   ```go
   defer fmt.Println("1")
   defer fmt.Println("2")
   defer fmt.Println("3")
   // prints: 3, 2, 1
   ```

3. **`defer` is what makes cleanup reliable** — it's the idiomatic replacement
   for `finally` blocks in try/finally languages, used constantly for closing
   files, unlocking mutexes, and closing database connections.

```
┌────────────────────────────────────────────────────────┐
│  func do() {                                              │
│      defer fmt.Println("1")   ─┐                          │
│      defer fmt.Println("2")   ─┤─  pushed onto a stack     │
│      defer fmt.Println("3")   ─┘                          │
│      // ... rest of function runs ...                       │
│      return                                                  │
│  }                                                             │
│  // on return, stack unwinds (LIFO): prints 3, then 2, then 1  │
└────────────────────────────────────────────────────────┘
```

---

Onto the projects — all three of these are menu/loop-driven programs, which is
exactly where `for`, `switch`, `break`/`continue`, labels, and `defer` earn
their keep together.
