# Module 01 Exercises

Each exercise is scoped tightly to a handful of fundamentals — resist the urge to
add extra features until you've run each one and read through the code once.

## 1. Temperature Converter

```bash
cd temperature-converter
go run main.go -val 100 -from c -to f
# 100.00°C = 212.00°F
```

**Drills:** basic types (`float64`), explicit type-free arithmetic, functions with
multiple single-purpose helpers, `switch` statements, formatted output (`%.2f`).

**Key idea — route through a base unit:** rather than writing all 6 direct
conversion formulas between C/F/K, the code always converts *into* Celsius first,
then *out* to the target. Converting between **N** units this way only needs
**2N** functions instead of **N×(N-1)** — a pattern that scales far better as you
add more units (see the Unit Converter exercise, where this really pays off).

```
        ┌─────────┐
  F ───▶│         │───▶ K
        │ Celsius │
  K ───▶│ (base)  │───▶ F
        └─────────┘
```

## 2. Age Calculator

```bash
cd age-calculator
go run main.go -year 1997 -month 8 -day 21
# Born:  August 21, 1997
# Today: July 31, 2026
# Age:   28 years, 11 months, 10 days
# Total days lived: 10570
```

**Drills:** the `time` package, `time.Date`/`time.Now`, calendar arithmetic vs.
plain duration math, and *why the two are different*.

**Key idea:** `now.Sub(birth)` gives you an exact `Duration` (nanosecond-precise),
which is perfect for "total days lived" — but it's the *wrong* tool for "how many
years/months/days" because calendar units aren't fixed-length (February has 28 or
29 days; months range from 28-31). The exercise manually "borrows" a month/year
the same way you'd borrow a ten in grade-school subtraction — this is a genuinely
useful mental model to carry into any date-math code you write later, in any
language.

## 3. Unit Converter

```bash
cd unit-converter
go run main.go -val 5 -from km -to mi
# 5 km = 3.107 mi

go run main.go -val 10 -from kg -to mi
# Error: cannot convert kg (weight) to mi (length) — different categories
```

**Drills:** `map[string]struct{...}` as a lookup table, struct literals, guarding
against nonsensical input (the category check), and generalizing the "route
through a base unit" idea from Exercise 1 to many more units at once.

**Key idea — table-driven design:** every unit is just one line in the `units`
map: a category label + "how many base units is 1 of me". Adding a new unit
(say, `yd` for yards) means adding **one line**, not touching any conversion
logic. This is the single most reusable pattern across all three exercises —
you'll see it again in bigger Go codebases as config-driven behavior, dispatch
tables, and test tables.

---

## Common Thread Across All Three

All three programs follow the same shape:
```
parse flags ──▶ validate input ──▶ do the calculation (pure function,
                                    no side effects, easy to test)
                                        │
                                        ▼
                                  print formatted result
```
Keeping the "calculate" step as a pure function (input in, value + error out,
nothing else touched) is why each of these would be easy to unit test — even
though this module hasn't covered `go test` yet, it's worth noticing the shape
now, because it's the shape testable Go code almost always has.
