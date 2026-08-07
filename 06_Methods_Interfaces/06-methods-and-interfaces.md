# 06. Methods & Interfaces

This is arguably the most important module so far — interfaces are how Go
achieves polymorphism without classes or inheritance, and they shape how
almost every real Go codebase is organized.

---

## 1. Methods

A method is a function with a **receiver** — you've been writing these since
Module 05, now let's name the parts precisely.

```go
type Rectangle struct {
	Width, Height float64
}

func (r Rectangle) Area() float64 { // (r Rectangle) is the receiver
	return r.Width * r.Height
}

rect := Rectangle{Width: 3, Height: 4}
fmt.Println(rect.Area()) // 12
```

```
┌─────────────────────────────────────────────────────┐
│  func   (r Rectangle)   Area()   float64   { ... }         │
│           │                 │        │                        │
│        receiver        method    return type                    │
│      "this method       name                                      │
│       belongs to                                                     │
│       Rectangle"                                                       │
└─────────────────────────────────────────────────────┘
```

Any named type can have methods — not just structs:
```go
type Celsius float64

func (c Celsius) ToFahrenheit() float64 {
	return float64(c)*9/5 + 32
}

temp := Celsius(100)
fmt.Println(temp.ToFahrenheit()) // 212
```

(Recall from Module 05: choose pointer vs. value receivers based on whether
the method needs to mutate, and stay consistent per type.)

---

## 2. Interfaces

An interface defines a **set of method signatures** — any type that has all
of those methods automatically satisfies the interface. There's no `implements`
keyword; **satisfaction is implicit**, decided purely by which methods a type
happens to have.

```go
type Shape interface {
	Area() float64
}

type Circle struct {
	Radius float64
}

func (c Circle) Area() float64 {
	return 3.14159 * c.Radius * c.Radius
}

// Rectangle from above already has Area() float64 too — so BOTH Circle and
// Rectangle satisfy Shape, without either of them ever mentioning "Shape".

func printArea(s Shape) {
	fmt.Printf("Area: %.2f\n", s.Area())
}

printArea(Circle{Radius: 5})
printArea(Rectangle{Width: 3, Height: 4})
```

```
┌──────────────────────────────────────────────────────────┐
│   type Shape interface { Area() float64 }                       │
│                                                                      │
│   Circle    has Area() float64  →  satisfies Shape  (automatically)   │
│   Rectangle has Area() float64  →  satisfies Shape  (automatically)     │
│   Square    has Area() float64  →  satisfies Shape  (automatically)       │
│                                                                                │
│   None of these types wrote "implements Shape" anywhere —                       │
│   Go checks method sets structurally, at compile time.                            │
└──────────────────────────────────────────────────────────┘
```

This is called **structural typing** (sometimes "duck typing, but checked at
compile time"): *"if it has the right methods, it fits the interface"* — very
different from Java/C#, where a class must explicitly declare
`implements Shape`. The practical upshot: you can define a small interface
*after the fact*, in your own package, that existing types (even ones from
the standard library or a third-party package you don't control) already
satisfy — a genuinely powerful decoupling tool.

**Interfaces are satisfied by the receiver type you actually use.** If
`Area()` has a *pointer* receiver (`func (c *Circle) Area() float64`), then
only `*Circle` (not plain `Circle`) satisfies `Shape` — worth double-checking
when something "should" satisfy an interface but the compiler disagrees.

---

## 3. Composition

Go doesn't have class inheritance — instead, you build bigger behavior by
**composing** smaller pieces together. You've already seen struct embedding
(Module 04) as one form of this; interfaces add another.

```go
type Reader interface {
	Read() string
}

type Writer interface {
	Write(data string)
}

// A bigger interface, COMPOSED from two smaller ones:
type ReadWriter interface {
	Reader
	Writer
}
```

Any type with both a `Read() string` and a `Write(data string)` method
automatically satisfies `ReadWriter` — no extra work needed. This
"small interfaces, composed into bigger ones" style is deeply idiomatic Go;
the standard library's `io` package is built almost entirely this way
(`io.Reader`, `io.Writer`, `io.ReadWriter`, `io.Closer`, `io.ReadCloser`...).

```
┌────────────────────────────────────────────────────┐
│   Reader { Read() string }                                │
│   Writer { Write(data string) }                              │
│                                                                  │
│   ReadWriter { Reader; Writer }   ◀── just LISTS the smaller      │
│                                        interfaces — no new              │
│                                        methods of its own needed          │
└────────────────────────────────────────────────────┘
```

---

## 4. Embedding

Struct embedding (Module 04) is also how you compose **implementations**, not
just interface definitions — embedding a struct that already satisfies an
interface means the outer struct satisfies it too, for free:

```go
type Logger struct{}

func (l Logger) Log(msg string) {
	fmt.Println("[LOG]", msg)
}

type Service struct {
	Logger // embedded — promotes Log(msg string)
	Name   string
}

s := Service{Name: "billing"}
s.Log("started") // promoted straight from Logger — [LOG] started
```

If something expects a `Logger`-satisfying interface (say, `type LogWriter
interface { Log(string) }`), `Service` satisfies it too — purely because it
embeds something that does. This is how Go achieves a lot of what other
languages reach for inheritance to do: not "Service IS-A Logger," but
"Service HAS-A Logger, and gets its behavior promoted along with it."

---

## 5. Type Assertions

Given a value of interface type, a **type assertion** lets you check (and
extract) its concrete underlying type:

```go
var s Shape = Circle{Radius: 5}

c, ok := s.(Circle) // "comma ok" form — ok is false if s isn't actually a Circle
if ok {
	fmt.Println("radius:", c.Radius)
}

c2 := s.(Circle) // WITHOUT ok — panics if the assertion is wrong! Use with care.
```

This is commonly used to check whether a value satisfies a *more specific*
interface than the one you're holding, so you can opt into extra behavior
only when it's available:

```go
type Shape interface {
	Area() float64
}

type Describable interface {
	Describe() string
}

func printShape(s Shape) {
	fmt.Printf("Area: %.2f\n", s.Area())
	if d, ok := s.(Describable); ok { // does this Shape ALSO satisfy Describable?
		fmt.Println(d.Describe())
	}
}
```

```
┌────────────────────────────────────────────────────────┐
│   value, ok := x.(SomeType)                                   │
│                                                                    │
│   ok == true   →  x's concrete type really is SomeType,             │
│                     value now holds it, safe to use                    │
│   ok == false  →  x is something else; value is SomeType's               │
│                     zero value, NO panic, safe to continue                 │
└────────────────────────────────────────────────────────┘
```

---

## 6. Empty Interface

`interface{}` (or its alias `any`, Go 1.18+) has **zero methods** — which
means, structurally, **every single type satisfies it**. It's Go's
"this could be anything" escape hatch.

```go
func describe(v any) {
	fmt.Printf("value: %v, type: %T\n", v, v)
}

describe(42)          // value: 42, type: int
describe("hello")      // value: hello, type: string
describe(Circle{Radius: 5}) // value: {5}, type: main.Circle
```

`map[string]any` is a common pattern for "arbitrary JSON-shaped data" before
you know its exact structure. The trade-off: once something is `any`, you've
lost all compile-time type safety on it — getting the real value back out
requires a type assertion or type switch (Module 02 covered the type-switch
syntax), and the compiler can no longer catch a wrong assumption for you.
**Use `any` sparingly** — a specific interface with the exact methods you
need is almost always preferable when you can define one.

---

## 7. Reflection Basics

Reflection is inspecting a value's type and structure **at runtime**, via the
`reflect` package — useful for generic tooling (JSON encoders, ORMs,
validators) that need to work with types they've never seen before.

```go
import "reflect"

func inspect(v any) {
	t := reflect.TypeOf(v)
	val := reflect.ValueOf(v)
	fmt.Println("Type:", t)
	fmt.Println("Kind:", t.Kind()) // the underlying category: struct, int, slice...

	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			fmt.Printf("  %s: %v\n", field.Name, val.Field(i))
		}
	}
}

inspect(Rectangle{Width: 3, Height: 4})
// Type: main.Rectangle
// Kind: struct
//   Width: 3
//   Height: 4
```

```
┌────────────────────────────────────────────────────────┐
│   reflect.TypeOf(v)    →  the TYPE information (struct name,   │
│                             field names, method set...)             │
│   reflect.ValueOf(v)   →  the actual VALUE, inspectable/           │
│                             settable field by field                   │
└────────────────────────────────────────────────────────┘
```

**Use reflection sparingly.** It bypasses compile-time type checking, runs
slower than direct field access, and makes code harder to follow — the Go
proverb (from Rob Pike himself) is *"reflection is never clear."* Reach for
it only when you're building something genuinely generic (a serializer, a
test-assertion helper) — everyday application code almost never needs it,
interfaces alone cover the vast majority of "I need this to work with
multiple types" needs.

---

Onto the projects — all three center on **interfaces enabling polymorphism**:
Payment Gateway swaps payment methods behind one interface, Notification
Service composes multiple notifiers together, and Storage Drivers builds
small, `io`-style composable interfaces for pluggable backends.
