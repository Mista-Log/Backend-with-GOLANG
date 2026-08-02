# Project 2 — Number Guessing

```bash
cd number-guessing
go run main.go
```

## What's Demonstrated Here

- **Condition-only `for` (the "while loop" shape)** — `playRound` deliberately
  avoids the classic three-part `for i := 1; i <= n; i++` form. Why: with the
  three-part form, `continue` still runs the post-statement (`i++`) before
  looping back, so an invalid guess would silently cost the player an attempt.
  Switching to `for attempt <= maxAttempts { ... attempt++ }` and only
  incrementing at the *end* of a valid guess fixes that — a small but genuinely
  useful lesson in how `continue` interacts with each loop shape differently.
- **Condition-less `switch`** — the low/high/correct/out-of-range branching
  reads like a clean if/else chain but as a `switch` (`switch { case ...: }`),
  which is idiomatic Go for exactly this kind of multi-branch comparison.
- **Labeled `continue` and `break`** — both target the `guessing:` loop by
  name from inside the `switch`, for the same reason as the ATM project: a
  bare `break` there would only exit the `switch`.
- **A second, outer labeled loop (`rounds:`)** — wraps the whole game so
  "play again?" can `continue rounds` (start a fresh round) or `break rounds`
  (fall through to the final score) from inside a nested `switch`.

```
┌────────────────────────────────────────────────────────┐
│  rounds: for {                                             │
│      playRound(reader)   ◀── contains its own guessing:     │
│                               loop, entirely self-contained  │
│      switch again {                                            │
│      case "y": continue rounds                                  │
│      default:  break rounds                                       │
│      }                                                               │
│  }                                                                     │
│  // final score prints here, after breaking out of rounds               │
└────────────────────────────────────────────────────────┘
```

## Try It Yourself
- Add a difficulty flag (`-max 1000`) that changes the range and recalculates
  a sensible `maxAttempts` (hint: `log2(max)` gives you the optimal binary-
  search attempt count)
- Track and print the best (fewest-guesses) win across all rounds
- Add a "give up" command mid-guess that reveals the number and ends the round
  early (another good use for a labeled `break`)
