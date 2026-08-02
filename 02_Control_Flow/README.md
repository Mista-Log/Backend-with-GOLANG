# Go for Beginners — Module 02: Control Flow

## Contents

1. **[02-control-flow.md](./02-control-flow.md)** — `if` (with its init-statement
   form), `switch` (including condition-less and type switches), `for` (all four
   shapes), `range`, labels, `break`, `continue`, `goto`, and `defer` basics
   (including the LIFO-order and argument-evaluation-timing gotchas). Diagrams
   included throughout.

2. **[atm-cli/](./atm-cli)** — Single-account ATM with PIN retry (labeled loop),
   a deposit/withdraw/balance menu, and a `defer`-based session summary. Includes
   a case study on the `defer` / `os.Exit` interaction.

3. **[number-guessing/](./number-guessing)** — Guess-the-number with hints, a
   condition-only `for` loop chosen specifically so invalid input doesn't cost
   an attempt, and two independent labeled loops (per-round guessing, and an
   outer "play again?" loop).

4. **[banking-menu/](./banking-menu)** — Multi-account bank with nested menus.
   Two *separately scoped* labeled loops show the difference between "back to
   the main menu" and "exit the whole program." Also covers why map iteration
   order needs explicit sorting for anything user-facing.

## Suggested Order

```
Control flow guide ──▶ ATM CLI ──▶ Number Guessing ──▶ Banking Menu
```

Each project reuses labels/switch/defer from the last, then adds one new
wrinkle: ATM CLI introduces labeled break/continue and defer; Number Guessing
shows how loop *shape* changes what continue does; Banking Menu stacks two
independently-scoped labeled loops across function boundaries.

## Quick Reference: Running Any Project

```bash
cd <project-folder>
go run main.go
```

*Note: this module builds on Modules 00 (Setup & Environment) and 01 (Go
Fundamentals) — start there first if you haven't already.*
