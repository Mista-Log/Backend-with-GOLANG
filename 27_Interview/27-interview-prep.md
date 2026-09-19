# 27. Interview Preparation

The final module. Everything before this built engineering skill; this one
is about **demonstrating** it under interview conditions — a different
skill, with its own preparation.

---

## DSA in Go

Go's standard library is deliberately smaller than Python's or Java's for
data structures, so know which primitives you'll actually build on:

```
┌──────────────────────────────────────────────────────────┐
│   Array/List      → []T slice (Module 04)                                │
│   Hash map         → map[K]V (Module 04) — K must be comparable              │
│   Set                → map[T]struct{} — struct{} takes ZERO bytes                │
│   Stack               → []T with append / s[:len(s)-1] (Module 07)                   │
│   Queue                → []T with append / s[1:], or container/list                      │
│   Heap/Priority Queue   → container/heap (implement its 5-method interface)                  │
│   Linked list            → container/list, or a hand-rolled *Node struct                         │
│   Sorting                 → sort.Slice / slices.Sort (Go 1.21+)                                      │
└──────────────────────────────────────────────────────────┘
```

**Go-specific gotchas interviewers actually probe:**

```go
// Set idiom — struct{} over bool, since struct{} allocates nothing:
seen := map[string]struct{}{}
seen["x"] = struct{}{}
if _, ok := seen["x"]; ok { /* present */ }

// Sorting a slice of structs:
sort.Slice(people, func(i, j int) bool { return people[i].Age < people[j].Age })

// Go 1.21+ generics-based helpers (know both — interviewers may expect either):
slices.Sort(nums)
slices.Contains(nums, 42)
maps.Keys(m) // returns an iterator in Go 1.23; slices.Collect(maps.Keys(m)) for a slice
```

---

## LeetCode

Patterns worth having fluent, with the Go-specific detail that matters
most for each:

```
┌──────────────────────────────────────────────────────────┐
│   Two pointers        → watch slice bounds; Go panics on out-of-range     │
│                            (no silent undefined behavior)                      │
│   Sliding window        → a map[byte]int for character counts is the             │
│                              common shape                                            │
│   Binary search           → sort.SearchInts, or hand-rolled: use                         │
│                                mid := lo + (hi-lo)/2 (overflow-safe habit,                  │
│                                even though Go's int is 64-bit)                                  │
│   BFS/DFS                   → BFS uses a queue ([]T with s[1:]); DFS uses                           │
│                                  recursion or an explicit stack                                        │
│   Dynamic programming         → make([]int, n+1) with a capacity hint                                     │
│                                    (Module 23's pre-allocation habit)                                        │
│   Backtracking                  → REMEMBER: append may share a backing array                                    │
│                                      (Module 04) — copy the slice before                                           │
│                                      storing a result, or every stored result                                         │
│                                      mutates together. THE classic Go                                                    │
│                                      LeetCode bug.                                                                          │
└──────────────────────────────────────────────────────────┘
```

**The backtracking trap, concretely** — this is the single most common Go
mistake in interview coding:
```go
// ❌ BROKEN — every appended result aliases the same backing array
results = append(results, current)

// ✅ CORRECT — copy first
cp := make([]int, len(current))
copy(cp, current)
results = append(results, cp)
```

---

## Concurrency Questions

Go interviews ask these far more than generic-language interviews do —
concurrency is the language's signature feature. Common questions, with
the answer's core:

```
┌──────────────────────────────────────────────────────────┐
│   "Goroutine vs. OS thread?"                                             │
│      ~2KB growable stack vs. ~1MB fixed; scheduled by Go's runtime           │
│      (M:N), not the OS directly (Module 12)                                      │
│                                                                                       │
│   "Buffered vs. unbuffered channel?"                                                   │
│      Unbuffered: send BLOCKS until a receiver is ready — a synchronization               │
│      point. Buffered: blocks only when full (Module 12)                                      │
│                                                                                                  │
│   "What happens if you close a channel twice? Send on a closed channel?"                          │
│      Both PANIC. Receiving from a closed channel returns the zero value                              │
│      immediately, with ok == false                                                                      │
│                                                                                                             │
│   "How do you stop a goroutine?"                                                                             │
│      You can't kill one externally — it must return on its own. Signal it                                       │
│      via context cancellation (Module 13) or a done channel                                                        │
│                                                                                                                       │
│   "Mutex vs. channel — when each?"                                                                                     │
│      Mutex: protecting shared STATE. Channel: transferring OWNERSHIP of                                                   │
│      data between goroutines. "Share memory by communicating" (Module 12)                                                    │
│                                                                                                                                 │
│   "What's a nil channel do?"                                                                                                     │
│      Sends and receives BLOCK FOREVER — occasionally useful in a select,                                                            │
│      to disable one case dynamically                                                                                                   │
└──────────────────────────────────────────────────────────┘
```

Three classic whiteboard problems are implemented in
`interview-practice/concurrency/`: a worker pool, alternating goroutines
(print 1,2,3... from two goroutines in strict alternation), and a
rate-limited fan-in.

---

## System Design

Module 26 covered this in depth, including four full case studies. The
interview-specific advice:

```
┌──────────────────────────────────────────────────────────┐
│   1. CLARIFY requirements first — scale, read/write ratio, consistency   │
│        needs. Never start drawing boxes immediately.                          │
│   2. Estimate roughly (back-of-envelope): QPS, storage, bandwidth.              │
│   3. High-level design — boxes and arrows, no detail yet.                           │
│   4. DEEP DIVE on 2-3 components the interviewer steers you toward.                    │
│   5. Discuss TRADE-OFFS explicitly — "I'd choose X over Y because...".                    │
│        This is what's actually being scored, more than the diagram.                          │
└──────────────────────────────────────────────────────────┘
```

**As a Go engineer specifically**, you have a genuine edge: when the
design calls for a worker pool, a circuit breaker, or an event-driven
flow, you can say "here's how I'd actually implement that" — Modules 12,
20, 21, 22, and 26 all built these for real.

---

## Behavioral

Use **STAR**: Situation, Task, Action, Result. Prepare 5-6 stories you can
adapt, covering:

```
┌────────────────────────────────────────────────────┐
│   - A hard technical problem you solved                                │
│   - A time you disagreed with a teammate/decision                          │
│   - A production incident you handled (or contributed to causing —             │
│      owning a mistake honestly is a strong signal, not a weak one)                 │
│   - A time you had to learn something unfamiliar fast                                 │
│   - Something you shipped end to end, and what you'd do differently now                   │
└────────────────────────────────────────────────────┘
```

**Result matters most and is most often skipped** — quantify it where you
honestly can ("cut p99 latency from 800ms to 120ms," "eliminated a class
of bug that had caused three incidents that quarter").

If you write publicly about engineering, that's genuinely useful material
here — a post explaining a system you designed doubles as a prepared,
well-structured answer you've already thought through carefully.

---

Onto the internals sections — the Go-specific questions that separate
"writes Go" from "understands Go," followed by runnable practice code.
