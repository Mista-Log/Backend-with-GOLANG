# Project 2 — Calculator CLI

This project introduces four things you'll use in nearly every Go program:
**the `flag` package, multiple return values, Go's explicit error handling,
and reading input with `bufio.Scanner`.**

## Two Ways to Run It

**One-shot mode** (good for scripting):
```bash
go run main.go -op add -a 4 -b 5
# 4.00 + 5.00 = 9.00

go run main.go -op div -a 10 -b 0
# Error: division by zero
```

**REPL mode** (good for playing around):
```bash
go run main.go
# Go Calculator REPL — type e.g. `add 4 5`, or `exit` to quit
# > add 4 5
# = 9.00
# > div 10 0
# Error: division by zero
# > exit
```

## Program Flow

```
┌─────────────────────────────────────────────────────────────┐
│                         main()                                │
│                            │                                  │
│               flag.Parse() reads -op -a -b                    │
│                            │                                  │
│              ┌─────────────┴─────────────┐                    │
│              ▼                           ▼                    │
│      -op not given                -op given                   │
│              │                           │                    │
│              ▼                           ▼                    │
│         runREPL()                  runOneShot()                │
│              │                           │                    │
│              ▼                           ▼                    │
│    loop: read line ──▶ parse ──▶ calculate() ──▶ print         │
│              │              (shared by both modes)             │
│              └──────── repeat until "exit" ────────┘           │
└─────────────────────────────────────────────────────────────┘
```

## The Core Idea: Go's Error Handling

Coming from languages with `try/catch`, Go's approach looks unusual at first:

```go
result, err := calculate(op, a, b)
if err != nil {
    // handle it right here, right now
}
```

Every function that can fail returns an **extra `error` value** alongside its
normal result. There's no hidden control flow — no exception can silently jump
past five stack frames before anyone notices. You see every possible failure
point explicitly, right where it happens.

```
┌──────────────────────────────────────────────────────────┐
│         Exceptions (Python/Java)   vs   Go errors          │
│                                                              │
│   try:                              result, err := calc()   │
│       result = risky()              if err != nil {         │
│   except Exception as e:                // handle here      │
│       handle(e)                     }                       │
│                                                              │
│   Failure can be thrown from        Failure is a normal,    │
│   ANYWHERE inside risky(),          explicit return value — │
│   caught far away, invisibly.       impossible to ignore    │
│                                      without the compiler    │
│                                      warning something's odd.│
└──────────────────────────────────────────────────────────┘
```

This program also shows the **sentinel error** pattern:
```go
var ErrDivideByZero = errors.New("division by zero")
```
A package-level error value that callers elsewhere could check precisely with
`errors.Is(err, ErrDivideByZero)` — useful once your program grows and different
callers need to react differently to different failure types (e.g., retry on a
network error, but not on a "division by zero").

## Case Study: Why `flag` Instead of Manual `os.Args` Parsing

You *could* parse `os.Args` by hand, but `flag` gives you for free:
- Automatic `-h` / `-help` output
- Type conversion (`flag.Float64` rejects non-numeric input with a clear error)
- Default values

This mirrors a broader Go philosophy: the standard library is unusually complete
for a systems language. Real production Go CLIs (`kubectl`, `docker`, `terraform`)
either use `flag` directly or a thin wrapper library (like `cobra`) built on the
same ideas — subcommands, flags, and help text as first-class citizens.

## Try It Yourself
- Add a `mod` (modulus) operation
- Add a `-precision` flag to control the number of decimal places in `%.2f`
- Make REPL mode remember the *last result* so you can type `add 5` to add 5
  to the previous answer (hint: track a `float64` variable outside the loop)
