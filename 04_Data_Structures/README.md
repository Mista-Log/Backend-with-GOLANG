# Go for Beginners — Module 04: Data Structures

## Contents

1. **[04-data-structures.md](./04-data-structures.md)** — Arrays, slices, maps,
   structs, nested structs, and embedding (with the promotion vs. shadowing
   rules) — plus an **Advanced** section on slice internals (the pointer +
   length + capacity header), capacity, append's reallocation behavior,
   `copy()`, and map internals (hashing, and why struct map keys must be
   fully comparable). Diagrams included throughout.

2. **[inventory-system/](./inventory-system)** — Two maps of struct pointers
   (`Product`, and `Perishable` which *embeds* `Product`). Covers promoted
   fields/methods, mutating through a map of pointers, and building a sorted
   view across two maps at once.

3. **[student-management/](./student-management)** — A *nested* struct
   (`Address`) side by side with an *embedded* one (`GraduateStudent` embeds
   `Student`), including a promoted **method** (`AddGrade`), not just fields.

4. **[book-library/](./book-library)** — Deliberately slice-based (not
   map-based) so you can directly watch `append`'s capacity-doubling
   reallocation happen, and see why `copy()` — not plain assignment — is
   needed for a truly independent snapshot.

## Suggested Order

```
Data structures guide ──▶ Inventory System ──▶ Student Management ──▶ Book Library
                            (maps + embedding)   (nested vs embedded,     (slices, capacity,
                                                   promoted methods)        append, copy)
```

Each project isolates a different piece of this module on purpose: Inventory
System and Student Management both use maps but focus on embedding from two
angles (fields-only vs. fields-and-methods); Book Library switches to a slice
specifically to make the Advanced section's internals visible while you use
the program.

## Quick Reference: Running Any Project

```bash
cd <project-folder>
go run main.go
```

*Note: this module builds on Modules 00–03 — start there first if you
haven't already.*
