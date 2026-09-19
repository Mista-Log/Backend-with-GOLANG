# Go Memory Model

**Source**: https://go.dev/ref/mem

Short, dense, and the definitive answer to "is this concurrent code
actually correct, or just currently working?" Read it after you've hit a
real race — it lands much harder with a concrete failure in mind.

## Questions worth bringing to it

- Which operations establish a happens-before edge? (channels, mutexes,
  `go` statements, `sync.Once`, atomics)
- Why is a data race undefined behavior rather than "you get one of the
  two values"?
- What exactly does `sync/atomic` guarantee, and what doesn't it?

## Notes in this folder

- `happens-before-edges.md`

## Related modules

Module 12 (concurrency, race detector), Module 22 (distributed consistency)
