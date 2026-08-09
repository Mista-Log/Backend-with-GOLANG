# 08. Error Handling

Go treats errors as **ordinary values**, not a separate control-flow
mechanism — you've been using this since Module 02, and this module fills in
the rest of the toolkit: building your own error types, wrapping errors with
context while preserving the original, and the `panic`/`recover` escape hatch
for the genuinely exceptional cases errors aren't meant for.

---

## 1. Errors

`error` is a built-in **interface** — just one method:

```go
type error interface {
	Error() string
}
```

That's it. Anything with an `Error() string` method satisfies `error` —
exactly the same implicit, structural satisfaction from Module 06.

```go
err := errors.New("something went wrong")
fmt.Println(err)         // something went wrong
fmt.Println(err.Error())  // something went wrong — Println calls Error() for you

err2 := fmt.Errorf("failed to process order %d", 42) // formatted, same idea as fmt.Sprintf
```

```
┌────────────────────────────────────────────┐
│   type error interface { Error() string }        │
│                                                        │
│   errors.New("msg")     →  a basic error                 │
│   fmt.Errorf("...%d", n) →  a formatted error                │
│                                                                  │
│   BOTH are just values satisfying the error interface —          │
│   nothing about "error" is magic beyond that one method.            │
└────────────────────────────────────────────┘
```

---

## 2. Custom Errors

Any type with an `Error() string` method is a valid error — this lets you
attach structured data (not just a string) to a failure.

```go
type NotFoundError struct {
	Resource string
	ID       int
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with ID %d not found", e.Resource, e.ID)
}

func findUser(id int) (*User, error) {
	// ... lookup fails ...
	return nil, &NotFoundError{Resource: "user", ID: id}
}
```

Callers that only care about the message can treat this exactly like any
other error (`if err != nil`, `fmt.Println(err)`), but callers that need the
structured data can get it back out via a type assertion or, better,
`errors.As` (see below).

**Sentinel errors** — package-level `var` values, checked by identity — are
the simplest form, seen back in Module 03's calculator:
```go
var ErrNotFound = errors.New("not found")

if err == ErrNotFound { // works, but errors.Is (below) is the more robust habit
	...
}
```

---

## 3. Wrapping

Wrapping preserves an original error while adding context, using `%w`
(instead of `%s` or `%v`) in `fmt.Errorf`:

```go
func loadConfig() error {
	_, err := os.Open("config.yaml")
	if err != nil {
		return fmt.Errorf("loading config: %w", err) // WRAPS err, doesn't discard it
	}
	return nil
}
```

```
┌──────────────────────────────────────────────────────┐
│   os.Open fails with:  *PathError{"config.yaml", ...}        │
│                              │                                  │
│                              ▼                                    │
│   fmt.Errorf("loading config: %w", err)                              │
│                              │                                         │
│                              ▼                                           │
│   a NEW error whose message is "loading config: open config.yaml:          │
│   no such file or directory" — but which STILL remembers the original        │
│   *PathError underneath, reachable via errors.Unwrap / errors.Is / errors.As    │
└──────────────────────────────────────────────────────┘
```

Wrapping can chain arbitrarily deep — each layer adds its own context while
keeping every earlier layer reachable:
```go
err := fmt.Errorf("layer 3: %w", fmt.Errorf("layer 2: %w", fmt.Errorf("layer 1: %w", baseErr)))
// err.Error() == "layer 3: layer 2: layer 1: <baseErr's message>"
```

**Use `%w` instead of `%v`/`%s` specifically when the caller might need to
check *what kind* of error this ultimately was** — logging code, for
instance, usually just wants the message and can use `%v`; code that needs
to react differently to different failure types needs `%w`.

---

## 4. `errors.Is`

`errors.Is(err, target)` checks whether `target` appears **anywhere in
`err`'s wrap chain** — not just at the top level, and not by comparing error
*messages*, but by walking the chain via `Unwrap()` and checking each layer.

```go
var ErrPermissionDenied = errors.New("permission denied")

func openFile() error {
	return fmt.Errorf("accessing resource: %w", ErrPermissionDenied)
}

err := openFile()
if errors.Is(err, ErrPermissionDenied) {
	fmt.Println("caller can react specifically to this, even though the")
	fmt.Println("actual err value is a totally different, wrapped error")
}
```

```
┌──────────────────────────────────────────────────────┐
│   err  →  "accessing resource: permission denied"          │
│              │                                                │
│              │ Unwrap()                                         │
│              ▼                                                    │
│           ErrPermissionDenied   ◀── errors.Is walks DOWN through      │
│                                       every wrap layer looking             │
│                                       for a match                            │
└──────────────────────────────────────────────────────┘
```

This is why sentinel errors (`var ErrX = errors.New(...)`) plus `errors.Is`
is the idiomatic replacement for `err == ErrX` — direct `==` comparison
breaks the moment the error gets wrapped even once, but `errors.Is` doesn't.

---

## 5. `errors.As`

`errors.As(err, &target)` walks the same wrap chain, but instead of checking
for one specific error *value*, it looks for the first error in the chain
matching a specific error *type*, and — if found — assigns it into `target`
so you can access its fields.

```go
err := findUser(42) // returns a wrapped *NotFoundError somewhere in its chain

var notFound *NotFoundError
if errors.As(err, &notFound) {
	fmt.Println("resource:", notFound.Resource) // fields are accessible now
	fmt.Println("id:", notFound.ID)
}
```

```
┌──────────────────────────────────────────────────────┐
│   errors.Is(err, target)   →  "is THIS SPECIFIC error value      │
│                                  anywhere in the chain?"                │
│                                                                            │
│   errors.As(err, &target)  →  "is there an error of THIS TYPE             │
│                                  anywhere in the chain? if so,               │
│                                  give it to me so I can use its                │
│                                  fields."                                        │
└──────────────────────────────────────────────────────┘
```

Rule of thumb: `errors.Is` for sentinel *values* ("did this specific known
failure happen?"), `errors.As` for custom error *types* whose fields you
need ("did some kind of NotFoundError happen, and if so, which resource?").

---

## 6. `panic`

`panic` immediately stops normal execution — running any deferred calls
along the way — and starts unwinding the call stack, printing a stack trace
and crashing the program **unless something recovers it** (next section).

```go
func divide(a, b int) int {
	if b == 0 {
		panic("division by zero") // NOT the same as returning an error!
	}
	return a / b
}
```

**`panic` is not Go's version of exceptions for ordinary failure handling.**
Reach for a returned `error` for anything an ordinary caller might
reasonably need to handle — a missing file, invalid input, a failed network
call. Reserve `panic` for **programmer errors** that indicate a bug (an
out-of-bounds slice index, a nil pointer dereference, a broken invariant
your own code should have prevented) — situations where continuing to run
would be actively wrong, not just inconvenient. The standard library itself
panics for exactly this class of problem (indexing past a slice's length,
for instance) and expects you to fix the bug, not routinely recover from it.

```
┌────────────────────────────────────────────────────┐
│   panic("...")                                            │
│        │                                                    │
│        ▼                                                      │
│   deferred calls in the current function run (LIFO, same         │
│   as always), THEN control unwinds to the CALLER, whose            │
│   deferred calls run too, and so on up the stack...                   │
│        │                                                                │
│        ▼                                                                  │
│   if nothing calls recover() along the way, the program CRASHES,           │
│   printing a full stack trace                                                │
└────────────────────────────────────────────────────┘
```

---

## 7. `recover`

`recover` stops a panic mid-unwind and lets the program keep running — but
it only works when called **directly inside a deferred function**, nowhere
else.

```go
func safeDivide(a, b int) (result int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from panic: %v", r) // NAMED return, set here
		}
	}()
	return a / b, nil // a real Go division by zero panics — this recovers it
}

result, err := safeDivide(10, 0)
fmt.Println(result, err) // 0, "recovered from panic: runtime error: integer divide by zero"
```

```
┌────────────────────────────────────────────────────┐
│   func f() {                                              │
│       defer func() {                                         │
│           if r := recover(); r != nil {                         │
│               // handle it — the panic STOPS unwinding HERE          │
│           }                                                            │
│       }()                                                                │
│       panic("boom")                                                        │
│   }                                                                           │
│   // f() returns NORMALLY after recover — the caller of f() never             │
│   // sees a panic at all                                                        │
└────────────────────────────────────────────────────┘
```

This pairs with **named returns** (Module 03) for exactly the reason shown
above: the deferred function runs *after* the `return` statement has
notionally happened, so modifying a named return value inside it is the way
to turn a recovered panic into a normal, returned `error`.

**Where `recover` genuinely belongs:** at the boundary of a system that must
stay up even if something inside it panics unexpectedly — an HTTP server
recovering per-request so one bad handler doesn't crash the whole process, a
worker pool that shouldn't die because one job panicked, or a library
function converting an internal panic into a returned error for its callers
(exactly the pattern above). **It does not belong** sprinkled throughout
ordinary business logic as a substitute for `if err != nil` — that inverts
Go's entire error-handling philosophy and hides real bugs instead of
surfacing them.

---

Onto the projects — Validation Package leans on custom error types and
`errors.As` for extracting structured field-level failures; Retry Library
leans on wrapping, `errors.Is` for distinguishing retryable from permanent
failures, and `recover` for containing panics inside retried work.
