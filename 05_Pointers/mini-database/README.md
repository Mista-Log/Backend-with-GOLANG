# Project — Mini Database

```bash
cd mini-database
go run main.go
```

Set a couple of keys, then try option 3 (the risky live-pointer read) to watch
the database get silently corrupted, followed by option 6 to see a transaction
protect against exactly that kind of mistake via rollback.

## What's Demonstrated Here

| Feature | Pointer concept |
|---|---|
| `records map[string]*Record` | Storing pointers so `Set` can mutate a record in place through the map |
| `func (r *Record) update(...)` | **Pointer receiver** — mutates the actual stored struct |
| `GetLive` vs `GetCopy` | The exact same address vs. dereferenced-copy distinction from the guide's `&`/`*` section, applied to a real "should this be safe to hand out?" design decision |
| `Set`'s `r := &Record{...}` | **Escape analysis** — `r`'s address is stored in the map and must outlive `Set`, so the compiler puts it on the heap automatically |
| `Snapshot()` returning `map[string]Record` (not `*Record`) | The deliberate choice that makes `Rollback` actually work — see the case study below |

```
┌────────────────────────────────────────────────────────┐
│  db.records["x"]  ──▶  *Record{Value: "hello", Version: 1}   │
│                                                                  │
│  GetLive("x")   returns the SAME pointer  ──▶ mutations show up   │
│                                                  in the database      │
│                                                                          │
│  GetCopy("x")   dereferences + copies      ──▶ mutations are LOCAL,       │
│                                                  the database is untouched  │
└────────────────────────────────────────────────────────┘
```

## Case Study: Why `Snapshot()` Must Return Values, Not Pointers

This is the same "shared backing array" trap from Module 04's Book Library
project, but one level more consequential — get it wrong here and rollback
silently does nothing:

```go
// ❌ If Snapshot() did this instead:
func (db *DB) SnapshotBroken() map[string]*Record {
	snap := make(map[string]*Record, len(db.records))
	for k, r := range db.records {
		snap[k] = r // SAME pointer — not a copy!
	}
	return snap
}
// Rollback("x") would try to restore snap["x"], but snap["x"] IS
// db.records["x"] — there was never a separate copy to restore FROM.
// Any change made after Begin() already "changed the snapshot" too.
```

```go
// ✅ What Snapshot() actually does:
snap[k] = *r   // dereference r, then COPY the resulting Record value
```

Because `Record` contains only plain value fields (strings, an int, a
`time.Time`), `*r` gives a fully independent copy — no nested pointers or
slices inside `Record` that would need their own deeper copying. `Rollback`
then does the reverse: for each snapshotted value, it takes `restored :=
snapshotValue` (a fresh local variable, with its own fresh address) and
stores `&restored` back into the live map. That "fresh local variable, own
address" step matters too — reusing a loop variable's address across
iterations is a classic pointer bug in other languages; Go 1.22+ scopes loop
variables per-iteration specifically to make patterns like this safe by
default (see Module 03's closures-in-loops section for the historical
gotcha this fixed).

## Try It Yourself
- Confirm escape analysis yourself: run `go build -gcflags="-m" .` in this
  folder and find the line reporting that the `Record` inside `Set` moves to
  the heap
- Add a `History(key string) []Record`, tracked by having `update` append a
  *copy* of the record's prior state to a `[]Record` field before changing it
  — another place where "copy, not pointer" is the deliberately correct choice
- Add a `CompareAndSwap(key, oldValue, newValue string) bool` that only
  updates if the current value still matches `oldValue` — a small taste of
  the concurrency-safety patterns pointers underpin in later modules
