# Interview Practice

```bash
cd interview-practice
go run ./cmd/demo     # every gotcha and solution, demonstrated
go test ./...          # the DSA test suite
go test -race ./...     # prove the concurrency solutions are race-free
```

## What's Here

- **`internals/`** — runnable proofs of the five classic Go gotchas: the
  nil-interface trap, slice aliasing (and the full-slice-expression fix),
  nil vs. empty slices, randomized map iteration, and loop-variable
  capture. Don't just read that these happen — run it and watch them.
- **`concurrency/`** — the four problems that come up most in Go
  interviews: a worker pool (with results reassembled in order, since
  that's the inevitable follow-up question), strictly alternating
  goroutines, fan-in with context cancellation, and the timeout pattern
  (with the buffered channel that prevents a goroutine leak).
- **`dsa/`** — LeetCode patterns in idiomatic Go, including
  `Permutations`, which handles the backtracking slice-aliasing trap
  correctly — and `dsa_test.go` includes a test specifically designed to
  **fail** if that copy were removed.

## Using This to Prepare

Run `go run ./cmd/demo` once to see everything work, then **close it and
rewrite each function from scratch**. Reading a correct solution feels
like learning; reproducing it under time pressure is the actual skill
being tested.

For the concurrency problems especially, practice explaining *why* out
loud — "I used an unbuffered channel here because the handoff itself is
the synchronization" is the kind of sentence that distinguishes a strong
Go interview from a passing one.
