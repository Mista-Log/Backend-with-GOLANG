# 04. Data Structures

This is the module where Go programs stop being "a script with some math" and
start looking like real software — modeling data with structs, and storing
collections of it in slices and maps.

---

## 1. Arrays

A fixed-length, fixed-type sequence — the length is part of the type itself.

```go
var nums [5]int              // [0 0 0 0 0] — zero-valued, length locked at 5
scores := [3]int{90, 85, 77}
matrix := [2][3]int{{1, 2, 3}, {4, 5, 6}} // 2D array

fmt.Println(len(scores)) // 3
```

`[5]int` and `[10]int` are **different types** — you can't assign one to the
other, and a function expecting `[5]int` will reject a `[10]int`. This
rigidity is exactly why arrays are rarely used directly in everyday Go code —
**slices** (next section) are almost always what you actually want.

```
┌───────────────────────────────────────────┐
│  [5]int  is a DIFFERENT TYPE from  [10]int    │
│  the way  int  is different from  string        │
└───────────────────────────────────────────┘
```

---

## 2. Slices

A slice is a flexible, growable *view* over an underlying array — this is the
data structure you'll reach for constantly.

```go
nums := []int{1, 2, 3}         // slice literal — no length in the brackets
nums = append(nums, 4)          // grows as needed

s := make([]int, 3)              // length 3, zero-valued: [0 0 0]
s := make([]int, 3, 10)           // length 3, but capacity 10 (see Advanced section)

sub := nums[1:3]                   // slicing: elements at index 1 and 2 (not 3)
```

```
┌────────────────────────────────────────────────────┐
│  nums := []int{10, 20, 30, 40, 50}                       │
│  nums[1:3]  →  [20, 30]        (start inclusive, end       │
│                                  exclusive — same as most     │
│                                  languages' slicing)             │
│  nums[:2]   →  [10, 20]        (omit start = from 0)              │
│  nums[2:]   →  [30, 40, 50]    (omit end = to the end)              │
└────────────────────────────────────────────────────┘
```

A 2D slice (slice of slices) is how you model a grid without a fixed array size:
```go
grid := make([][]int, 3)
for i := range grid {
	grid[i] = make([]int, 3)
}
```

**See the Advanced section below** for what a slice actually *is* under the
hood (pointer + length + capacity) — that's essential for understanding why
slices sometimes behave surprisingly when passed to functions or copied.

---

## 3. Maps

Go's built-in hash table — unordered key/value storage.

```go
ages := map[string]int{"Alice": 30, "Bob": 25}
ages["Carol"] = 40                              // add/update
delete(ages, "Bob")                              // remove

age, ok := ages["Alice"]  // "comma ok" — ok is false if the key doesn't exist
if !ok {
	fmt.Println("not found")
}

empty := make(map[string]int) // empty map, ready to use
var nilMap map[string]int      // a NIL map — reading is safe (returns zero
                                 // value), but WRITING to it panics!
```

```
┌────────────────────────────────────────────────────┐
│  var m map[string]int     →  m is nil                  │
│  m["x"]                    →  0  (safe — zero value)      │
│  m["x"] = 1                 →  PANIC: assignment to entry    │
│                                  in nil map                     │
│                                                                    │
│  m := make(map[string]int)  →  m is a real, empty map            │
│  m["x"] = 1                   →  works fine                        │
└────────────────────────────────────────────────────┘
```

Remember from Module 02: **map iteration order is randomized on purpose** —
sort your keys first if you need a stable, user-facing order.

---

## 4. Structs

Structs group related fields into one named type — Go's answer to "a class
without methods or inheritance."

```go
type Person struct {
	Name string
	Age  int
}

p := Person{Name: "Ada", Age: 28}   // struct literal, field names explicit (preferred)
p2 := Person{"Ada", 28}              // positional — works, but fragile if fields reorder

p.Age = 29                            // fields are accessed/set with dot notation

fmt.Printf("%v\n", p)   // {Ada 29}
fmt.Printf("%+v\n", p)  // {Name:Ada Age:29} — field names included
```

Structs are **value types** — assigning or passing one copies it entirely,
unless you explicitly use a pointer:
```go
func birthday(p Person) {
	p.Age++ // modifies a COPY — has no effect on the caller's Person
}

func birthdayPtr(p *Person) {
	p.Age++ // modifies the ORIGINAL, through the pointer
}

birthday(p)     // p.Age unchanged
birthdayPtr(&p)  // p.Age incremented
```

This is why you'll see `func (p *Person) SomeMethod()` (pointer receiver) far
more often than `func (p Person) SomeMethod()` (value receiver) once methods
are involved — pointer receivers let a method actually mutate the struct.

---

## 5. Nested Structs

A struct field can itself be another struct — this is how you model
"has-a" relationships (an Order *has an* Address, not *is an* Address).

```go
type Address struct {
	City    string
	Country string
}

type Employee struct {
	Name    string
	Address Address // nested — accessed as employee.Address.City
}

e := Employee{
	Name: "Kemi",
	Address: Address{City: "Lagos", Country: "Nigeria"},
}
fmt.Println(e.Address.City) // Lagos
```

```
┌───────────────────────────────────────────┐
│  Employee                                     │
│   ├── Name    "Kemi"                            │
│   └── Address  Address{                            │
│                    City:    "Lagos",                   │
│                    Country: "Nigeria",                    │
│               }                                              │
│                                                                  │
│  Access: e.Address.City                                          │
└───────────────────────────────────────────┘
```

---

## 6. Embedding

Embedding looks similar to nesting but drops the field name — and when you do
that, the outer struct **promotes** the embedded struct's fields and methods
to its own top level. This is Go's deliberate alternative to class inheritance.

```go
type Animal struct {
	Name string
}

func (a Animal) Describe() string {
	return a.Name + " is an animal"
}

type Dog struct {
	Animal    // EMBEDDED — no field name, just the type
	Breed string
}

d := Dog{Animal: Animal{Name: "Rex"}, Breed: "Labrador"}
fmt.Println(d.Name)        // "Rex" — PROMOTED straight from Animal, no d.Animal.Name needed
fmt.Println(d.Describe())   // "Rex is an animal" — PROMOTED method too
fmt.Println(d.Animal.Name)  // also valid — the explicit path still works
```

```
┌──────────────────────────────────────────────────────┐
│  Nesting (named field)     vs      Embedding (no name)    │
│                                                              │
│  type Employee struct {          type Dog struct {            │
│      Address Address                 Animal                     │
│  }                                    Breed string                 │
│                                    }                                   │
│  e.Address.City                    d.Name          ◀── PROMOTED,      │
│  (must go through                     (no .Animal.        no need to    │
│   the field name)                      needed!)              qualify it   │
└──────────────────────────────────────────────────────┘
```

**Important: this is composition, not real inheritance.** A `Dog` does not
*become* an `Animal` — you can't pass a `Dog` anywhere that specifically
requires an `Animal` value (there's no polymorphism through embedding alone;
that comes from **interfaces**, a later module). Embedding only gives you
promoted field/method access as a convenience. If both the embedded type and
the outer type define a field/method with the same name, the outer type's own
version wins — the embedded one is simply shadowed, accessible only via its
explicit path (`d.Animal.Name`).

---

## Advanced: How Slices and Maps Actually Work

### Slice Internals

A slice value is a small struct with three fields, **not** the data itself:

```
┌─────────────────────────────────────────────┐
│                    slice header                  │
│   ┌─────────┬────────┬──────────┐                  │
│   │ pointer │ length │ capacity │                    │
│   └────┬────┴────────┴──────────┘                      │
│        │                                                  │
│        ▼                                                    │
│   underlying array:  [10][20][30][ ][ ]                       │
│                        len=3 ─┘        capacity=5                │
└─────────────────────────────────────────────┘
```

This matters because **assigning or passing a slice copies the header, not
the underlying array** — so two slices can point at the *same* backing array:

```go
a := []int{1, 2, 3}
b := a            // b shares the SAME underlying array as a
b[0] = 999
fmt.Println(a[0]) // 999 — a was affected too!
```

### Capacity

**Capacity** is how many elements the underlying array *could* hold before a
new array is needed — it's always ≥ length.

```go
s := make([]int, 3, 5) // len=3, cap=5
fmt.Println(len(s), cap(s)) // 3 5

s = append(s, 10) // fits within existing capacity — no reallocation
fmt.Println(len(s), cap(s)) // 4 5
```

### Append

`append` is where capacity actually matters:

```
┌────────────────────────────────────────────────────────┐
│  If len < cap:                                              │
│      append writes into the EXISTING array's next slot,        │
│      just bumps length — cheap, no allocation                     │
│                                                                       │
│  If len == cap (array is full):                                        │
│      append allocates a BRAND NEW, bigger array (Go typically           │
│      doubles capacity for smaller slices), copies everything             │
│      over, THEN adds the new element                                        │
│                                                                                 │
│  This is why appending to a slice that's shared with another                     │
│  slice can suddenly "stop" affecting that other slice — once                        │
│  a reallocation happens, they point at DIFFERENT arrays.                              │
└────────────────────────────────────────────────────────┘
```

```go
a := make([]int, 3, 3) // len=3, cap=3 — completely full
b := a
a = append(a, 4)         // cap exceeded -> a gets a NEW underlying array
a[0] = 999
fmt.Println(b[0])         // still the OLD value — b still points at the old array
```

### Copy

`copy(dst, src)` copies elements from `src` into `dst`, up to whichever is
shorter — genuinely copying data (unlike a plain slice assignment, which only
copies the header):

```go
src := []int{1, 2, 3}
dst := make([]int, 3)
n := copy(dst, src) // n == 3, dst is now an independent [1 2 3]
dst[0] = 999
fmt.Println(src[0]) // still 1 — dst has its OWN array now
```

### Map Internals

A Go map is a hash table under the hood: keys are hashed to find a "bucket,"
and each bucket holds a handful of key/value pairs.

```
┌───────────────────────────────────────────────────────┐
│   key "Alice"  ──▶ hash function ──▶ bucket #2             │
│   key "Bob"    ──▶ hash function ──▶ bucket #7                │
│                                                                    │
│   [bucket 0] [bucket 1] [bucket 2: Alice→30] [bucket 3] ...        │
│                                                                        │
│   Lookup, insert, and delete are all O(1) on AVERAGE — this            │
│   is why maps are the go-to structure for "look something up            │
│   by key" instead of scanning a slice with a loop.                        │
└───────────────────────────────────────────────────────┘
```

Two practical consequences worth remembering:
- **Maps are reference types like slices** — passing a map to a function
  lets that function mutate the original (no copy of the underlying data
  happens, unlike passing a struct by value).
- **A struct used as a map key must be entirely comparable** — no slices,
  maps, or functions among its fields — since Go needs to hash the whole key.

---

Onto the projects — Inventory System leans on maps-of-structs for O(1)
lookups by ID, Student Management leans on nested structs and embedding, and
Book Library leans on slices, capacity, and append behavior for its checkout
history.
