# Project 3 — Generic Cache

```bash
cd generic-cache
go run main.go
```

## What's Demonstrated Here

- **Two type parameters (`Cache[K comparable, V any]`)** — the key and value
  types are independently generic. `Cache[string, User]` and
  `Cache[int, float64]` are both used in the demo, with zero shared code
  changes between them.
- **Why `K` needs `comparable` but `V` doesn't** — `K` is used as an actual
  Go map key internally (`map[K]entry[V]`), which requires `==` support,
  exactly the rule from Module 04's map-internals section. `V` is only ever
  stored and returned, never compared, so it stays at the more permissive
  `any` — same reasoning as `Queue[T any]` and `Stack[T any]` in the other
  two projects.
- **Lazy TTL expiry** — `Get` checks `hasExpiry` and `time.Now().After(...)`
  on access, deleting the entry right then if it's stale. No background
  goroutine sweeping for expired entries — that's a reasonable next step,
  but it needs concurrency-safety tools from a later module first.
- **FIFO capacity eviction** — `order []K` tracks insertion order
  specifically so `evictIfNeeded` knows which entry is oldest; `Delete`
  keeps `order` in sync using `==` on `K`, which — again — is only legal
  because `K` is constrained to `comparable`.

```
┌──────────────────────────────────────────────────────────┐
│   type Cache[K comparable, V any] struct {                       │
│       data  map[K]entry[V]                                          │
│       order []K                                                        │
│   }                                                                       │
│                                                                              │
│   Cache[string, User]   →  data is map[string]entry[User]                     │
│   Cache[int, float64]   →  data is map[int]entry[float64]                        │
│                                                                                        │
│   K MUST satisfy comparable (used as a real map key + compared with ==)                 │
│   V has NO constraint (only ever stored/returned, never compared)                          │
└──────────────────────────────────────────────────────────┘
```

## Case Study: Why This Project Needed Its Own Constraint Choice

Generic Queue and Generic Stack both used `[T any]` because neither ever
needs to compare, hash, or otherwise operate on the elements it holds —
purely storage. `Cache` looks superficially similar (it's also "just
storage"), but its *key* has a hidden requirement the other two don't:
**it has to work as a Go map key**, which immediately pulls in
`comparable` — not because the cache's logic wants to compare keys for its
own sake, but because `map[K]...` itself demands it. This is a good general
instinct to build: when you reach for a data structure that uses `map[X]...`
or `X == X` internally, `X`'s type parameter needs `comparable` (or
something stricter); when a type parameter is only ever stored, copied, or
passed along untouched, `any` is the right (and most reusable) choice.

## Try It Yourself
- Swap the FIFO eviction for a true **LRU** (least-recently-used): update
  `order` (or move the touched key to the end of it) inside `Get`, not just
  `Set`, so reading an entry counts as "recently used" too
- Add a `SetIfAbsent(key K, value V) bool` that only sets when the key isn't
  already present (or has expired), returning whether it actually set
  anything — a small building block toward safe concurrent caches later
- Add a `Cleanup()` method that proactively removes all expired entries in
  one pass, instead of relying purely on lazy expiry during `Get`
