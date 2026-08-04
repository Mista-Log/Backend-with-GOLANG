# Project 3 — Book Library

```bash
cd book-library
go run main.go
```

Add 5-6 books in a row and watch option 1's capacity printout after each one —
`cap` won't grow by exactly 1 each time; it jumps ahead. Then try option 7 to
see `copy()` prove its independence from the live catalog.

## What's Demonstrated Here

Unlike Inventory System and Student Management (both map-based), this
project's catalog is a **plain `[]Book` slice** — a deliberate choice so
append/capacity/copy behavior is front and center, not hidden behind a hash
table.

- **Linear search (`findIndex`)** — the direct trade-off of choosing a slice
  over a map: O(n) lookup instead of O(1). Worth comparing directly against
  Inventory System's `map[string]*Product` lookups.
- **Watching capacity grow** — `printCapacityInfo` after every `addBook` call
  shows `len` and `cap` side by side, so you can watch Go's reallocation
  strategy happen in real time instead of just reading about it.
- **`copy()` vs. plain assignment** — `snapshot()` uses `copy()` to build a
  genuinely independent slice; option 7 then deliberately mutates the live
  catalog afterward and shows the snapshot is unaffected.

## Case Study: Watching Append Reallocate, Live

Add books one at a time via option 1 and read the capacity line after each:

```
books slice: len=1 cap=1
books slice: len=2 cap=2
books slice: len=3 cap=4      ◀── jumped from 2 to 4, not 3
books slice: len=4 cap=4
books slice: len=5 cap=8      ◀── jumped from 4 to 8
```

(Exact numbers can vary slightly by Go version, but the doubling pattern
holds.) Every time `len` catches up to `cap`, the *next* `append` has to
allocate a new, bigger backing array and copy every existing element into it
— which is why Go doesn't just grow by 1 each time: that would mean a fresh
allocation-and-copy on *every single append*, which gets expensive fast for
large slices. Doubling means the *average* cost of an append stays cheap
(this is the same amortized-growth idea behind dynamic arrays in most
languages — Python lists, Java's `ArrayList`, C++'s `std::vector`).

## Case Study: Why `snapshot()` Can't Just Be `frozen := lib.books`

```go
frozen := lib.books          // ❌ copies the slice HEADER only
frozen[0].Available = 999     // mutates the SAME underlying array —
                                // lib.books[0].Available is now 999 too!

frozen := make([]Book, len(lib.books))   // ✅ a real, separate backing array
copy(frozen, lib.books)                    // actually copies every Book struct over
frozen[0].Available = 999                    // only affects frozen — lib.books is untouched
```

This is the single most common slice bug for people new to Go: a slice
variable *looks* like "the data," but it's really a small header pointing at
shared data. `append` sometimes silently gives you a new backing array (when
capacity runs out) and sometimes doesn't (when there's room) — which is
exactly why relying on two slices staying in sync, or not, is fragile unless
you're deliberate about it with `copy()`.

## Try It Yourself
- Pre-allocate `lib.books` with `make([]Book, 0, 100)` in `main` before the
  loop starts, then compare the capacity printout pattern — no more doubling
  jumps, since there's already room
- Add a `removeBook(isbn string) error` — note that removing from the
  *middle* of a slice (not just the end) needs either `append(s[:i], s[i+1:]...)`
  or a manual shift; this is a good place to see slicing and append combined
- Extend `Book` with a `[]string` field for reviews, and use `append` to add
  to it — practice nested append (a slice field inside a struct inside a
  slice)
