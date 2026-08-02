# Project 1 — ATM CLI

```bash
cd atm-cli
go run main.go
```

Try entering the wrong PIN a couple of times, then the correct one (`1234`), then
play with deposit/withdraw/balance before exiting.

## What's Demonstrated Here

```
┌─────────────────────────────────────────────────────────────┐
│  authenticate()                                                │
│  attempts: for i := 1; i <= 3; i++ {                            │
│      switch pin {                                                │
│      case correctPIN:                                            │
│          break attempts   ◀── labeled break: exits the FOR loop, │
│      }                          not just the switch               │
│  }                                                                 │
│                                                                     │
│  main()                                                              │
│  menu: for {                                                          │
│      switch choice {                                                   │
│      case "4":                                                          │
│          break menu       ◀── same pattern again, from the main menu    │
│      }                                                                    │
│  }                                                                          │
└─────────────────────────────────────────────────────────────┘
```

- **Labeled `break`** — `authenticate` uses `break attempts` to escape the retry
  loop the instant the PIN is correct, from *inside* a `switch` case. This is
  exactly the scenario labels exist for: a bare `break` here would only exit the
  `switch`, leaving the `for` loop still running.
- **Labeled `continue`** — used explicitly in the PIN loop and the deposit/
  withdraw cases (`continue menu`) to jump straight back to the next menu
  prompt after an invalid amount, skipping any code below it in that iteration.
- **`switch` on a string** — both the PIN check and the menu choice are string
  switches with a clean `default` case for anything unrecognized.
- **`defer`** — the session-summary line is deferred once, near the top of
  `main`, and fires no matter which path ends the program: successful exit
  (option 4), or `os.Exit`-free normal return. (Note: `os.Exit` in
  `authenticate`'s failure path is the *one* case that would skip deferred
  calls — see the case study below.)

## Case Study: The `defer` / `os.Exit` Gotcha

This project deliberately hides a subtle, very real Go trap. Look at:

```go
if !authenticate(reader) {
	os.Exit(1)
}
```

`os.Exit` terminates the program **immediately** — it does **not** run any
deferred functions. So if authentication fails, the "Session ended..." message
never prints, even though `defer` was already registered. This is one of the
most common surprises for people new to Go: `defer` guarantees "runs before
this function returns normally (or panics)" — but `os.Exit` sidesteps that
entirely by killing the process outright, no unwinding at all.

**The fix**, if you want the deferred message to always print, is to avoid
`os.Exit` for controlled shutdowns and instead `return` an exit code up to
`main`, or restructure so the failure path also flows through a normal return.
Try modifying `authenticate` to return the exit code instead of calling
`os.Exit` directly, and confirm the deferred message now always prints.

## Try It Yourself
- Fix the `os.Exit` / `defer` interaction described above
- Add a "transfer to another account" option (you'll need a second `account`)
- Add a maximum single-withdrawal limit (e.g., can't withdraw more than 300 at
  once, even with sufficient balance)
- Track a transaction history (`[]string`) and print it as a receipt on exit
