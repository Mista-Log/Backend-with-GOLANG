# Go for Beginners — Module 07: Generics

## Contents

1. **[07-generics.md](./07-generics.md)** — Type parameters, constraints
   (interfaces that describe allowed type sets, not just required methods),
   the built-in `comparable` constraint, `any` in a generics context, custom
   constraints (including the `~underlying-type` syntax for named types),
   and generic data structures — plus the limitation that a method can't
   introduce a narrower constraint than its type already declared. Diagrams
   included throughout.

2. **[generic-queue/](./generic-queue)** — `Queue[T any]`, instantiated with
   `int`, `string`, and a custom `Task` struct in the same program. Covers
   the generic-safe "comma ok" pattern (`Dequeue() (T, bool)`) and `var zero T`.

3. **[generic-stack/](./generic-stack)** — `Stack[T any]` plus a standalone
   `MaxInStack[T Ordered]` function, with a case study on *why* it has to be
   standalone rather than a method. Includes a genuinely practical use case:
   a bracket-matching `IsBalanced` function built on `Stack[rune]`.

4. **[generic-cache/](./generic-cache)** — `Cache[K comparable, V any]`, the
   module's only *two*-type-parameter structure. Covers exactly why `K`
   needs `comparable` (it's used as a real map key) while `V` doesn't, plus
   lazy TTL expiry and simple FIFO capacity eviction.

## Suggested Order

```
Generics guide ──▶ Generic Queue ──▶ Generic Stack ──▶ Generic Cache
                     ([T any])         ([T any] +         ([K comparable,
                                       custom Ordered       V any] — two
                                       constraint)           parameters)
```

Each project's constraint choice builds on the last: Queue needs nothing
beyond `any`; Stack adds a custom constraint on a *standalone function*
rather than the type itself; Cache needs `comparable` on one of its *two*
type parameters, for a reason specific to how it's implemented internally
(a real Go map under the hood).

## Quick Reference: Running Any Project

```bash
cd <project-folder>
go run main.go
```

*Note: this module builds on Modules 00–06 — start there first if you
haven't already, especially Module 04 (map internals / `comparable`) and
Module 06 (interfaces), since constraints are themselves interfaces.*
