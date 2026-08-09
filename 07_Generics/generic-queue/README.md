# Project 1 — Generic Queue

```bash
cd generic-queue
go run main.go
```

## What's Demonstrated Here

- **`Queue[T any]`** — one implementation, instantiated as `Queue[int]`,
  `Queue[string]`, and `Queue[Task]` (a custom struct) in the same program,
  with the compiler enforcing the right type at every call site. No `any`
  leaks out to callers, and no type assertions are needed anywhere in `main`.
- **The "comma ok" pattern, generic-safe** — `Dequeue` returns `(T, bool)`
  instead of just `T` (or `nil` on empty, which wouldn't even compile for a
  non-pointer `T`). This is the same idiom as map lookups
  (`v, ok := m[k]`) and channel receives, extended to work for *any* `T`,
  including value types like `int` that have no natural "empty" sentinel of
  their own.
- **`var zero T`** — the trick for producing "the zero value of whatever T
  turns out to be" inside a generic function, without knowing T's concrete
  type in advance. For `Queue[int]` this is `0`; for `Queue[Task]` it's a
  `Task{}` with every field at its own zero value.

```
┌──────────────────────────────────────────────────────┐
│   Queue[int]     items: []int      Enqueue/Dequeue deal      │
│                                      in int values directly      │
│                                                                      │
│   Queue[Task]    items: []Task     Enqueue/Dequeue deal            │
│                                      in Task values directly           │
│                                                                              │
│   Same struct definition, same methods — the compiler generates the           │
│   right specialized behavior for each instantiation.                            │
└──────────────────────────────────────────────────────┘
```

## Case Study: What This Replaced, Pre-Generics

Before Go 1.18, a genuinely reusable FIFO queue had two realistic options:

1. **Copy-paste a new `IntQueue`, `StringQueue`, `TaskQueue`...** for every
   element type you needed — works, but it's the same twelve lines of logic
   duplicated N times, and any bug fix has to be applied N times too.
2. **Write one `Queue` with `items []any`, and force every caller to type-assert:**
   ```go
   val := queue.Dequeue().(Task) // panics if you get the assertion wrong,
                                    // and the compiler can't catch a mismatch
                                    // for you at compile time
   ```

`Queue[T any]` gets you the reusability of option 2 with the safety of
option 1 — genuinely the reason generics were the most-requested Go feature
for years before they arrived.

## Try It Yourself
- Add a `Contains` method — you'll need to change the constraint from
  `T any` to `T comparable` (see the Generic Cache project for why), since
  checking equality requires it
- Add a `Clear()` method and a `ToSlice() []T` that returns a copy of the
  current contents in order
- Implement a **priority queue** variant: `Enqueue` still appends, but
  `Dequeue` removes the *highest-priority* `Task` instead of the front one —
  this one can't stay `[T any]`, since it needs to compare priorities; think
  about what constraint or extra parameter that requires
