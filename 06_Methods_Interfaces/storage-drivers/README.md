# Project 3 — Storage Drivers

```bash
cd storage-drivers
go run main.go
```

This creates a real file, `storage-data.json`, in this folder — open it after
running to see what `FileStorage.Close()` actually flushed to disk.

## What's Demonstrated Here

- **`Store`, composed from three small interfaces** (`Getter`, `Setter`,
  `Deleter`, plus a `Keys` method) — same "small pieces, composed" idiom as
  the guide's `ReadWriter`, now with three pieces instead of two.
- **`Closer` as a genuinely optional interface** — `MemoryStorage` has
  nothing to clean up, so it simply doesn't implement `Close()` at all.
  `FileStorage` buffers writes in memory and only touches disk in `Close()`,
  so it needs to satisfy `Closer` — and `closeIfPossible` discovers that via
  a type assertion, without either backend needing to know about the other.
- **A driver registry (`driverRegistry`)** — a `map[string]func(string) (Store, error)`,
  the same dispatch-table idea from Module 03's Calculator API, here mirroring
  a real, well-known Go idiom: the standard library's `database/sql` package
  lets you `sql.Open("postgres", ...)` or `sql.Open("sqlite3", ...)` by
  registering driver constructors under a name — exactly this pattern, at
  production scale.
- **Reflection for value inspection** — `describeValue` reports the actual
  concrete type behind each `any`-typed stored value, since a generic
  key/value store can't know in advance what's been stored where.

```
┌──────────────────────────────────────────────────────────┐
│   type Store interface {                                        │
│       Getter                                                       │
│       Setter                                                         │
│       Deleter                                                          │
│       Keys() []string                                                    │
│   }                                                                          │
│                                                                                  │
│   MemoryStorage  ──▶ satisfies Store ONLY                                          │
│   FileStorage    ──▶ satisfies Store AND Closer                                       │
│                                                                                            │
│   closeIfPossible(s Store) checks s.(Closer) — MemoryStorage: no-op,                        │
│   FileStorage: flushes buffered writes to disk                                                │
└──────────────────────────────────────────────────────────┘
```

## Case Study: Why `FileStorage` Buffers Instead of Writing on Every `Set`

An alternative design would have `Set` and `Delete` write the whole file to
disk immediately, every single time. That's simpler, but far more expensive
if a caller sets ten keys in a row — ten full file rewrites instead of one.
Buffering in memory and flushing once, in `Close()`, trades a small risk
(changes are lost if the program crashes before `Close()` runs — the `dirty`
flag exists specifically to skip a wasted write when nothing actually
changed) for a real performance win. This is the exact same trade-off
covered in Module 02's ATM CLI case study: **`defer` is the idiomatic way to
guarantee `Close()` actually runs**, even on an early return —
`defer closeIfPossible(store)` right after `openStore` succeeds would be the
production-ready version of this program's cleanup, instead of the explicit
call at the end of `demo` used here for clarity.

## Try It Yourself
- Change `main` to use `defer closeIfPossible(fileStore)` immediately after
  `openStore` succeeds, instead of calling it explicitly at the end of `demo`
- Add a `"redis"` (or any other name) entry to `driverRegistry` that just
  returns an error for now (`"not yet implemented"`) — confirm `openStore`
  handles it the same way it handles a genuinely unknown driver name today
- Add a `Size() int` method to `Store` (byte size of all stored data) —
  you'll need `reflect` or `encoding/json`'s marshaled length to estimate it
  for arbitrary `any` values
