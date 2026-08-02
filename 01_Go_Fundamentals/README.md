# Go for Beginners — Module 01: Go Fundamentals

## Contents

1. **[01-go-fundamentals.md](./01-go-fundamentals.md)** — History, compilation,
   how the Go runtime works, memory layout, variables, constants, zero values,
   basic types, type conversion, formatting, input, packages, exported names,
   and scope. Diagrams included throughout.

2. **[exercises/temperature-converter/](./exercises/temperature-converter)** —
   Converts C/F/K by routing through a common base unit.

3. **[exercises/age-calculator/](./exercises/age-calculator)** — Calendar
   arithmetic with the `time` package; years/months/days vs. raw duration math.

4. **[exercises/unit-converter/](./exercises/unit-converter)** — Table-driven
   design with a `map[string]struct{...}` for length/weight/volume conversions.

See **[exercises/README.md](./exercises/README.md)** for a breakdown of exactly
which fundamentals each exercise drills, plus the common pattern shared by all
three.

## Suggested Order

```
Fundamentals guide ──▶ Temperature Converter ──▶ Age Calculator ──▶ Unit Converter
```

Each exercise is a small step up in difficulty and introduces exactly one or
two new ideas on top of the last.

## Quick Reference: Running Any Exercise

```bash
cd exercises/<exercise-folder>
go run main.go -val 100 -from c -to f   # example — flags differ per exercise,
                                          # see each folder's main.go comments
```

*Note: this module builds on Module 00 (Setup & Environment) — if you haven't
gone through that one yet, start there first.*
