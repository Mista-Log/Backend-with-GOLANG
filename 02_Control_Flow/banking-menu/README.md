# Project 3 — Banking Menu

```bash
cd banking-menu
go run main.go
```

Create a couple of accounts, deposit into one, transfer between them, then list
accounts to see the running balances.

## Architecture: Two Independent Labeled Loops

```
┌───────────────────────────────────────────────────────────┐
│  main:  for {              ◀── the MAIN menu loop            │
│      switch choice {                                            │
│      case "3":                                                    │
│          manageAccount(...)  ◀── calls into a SEPARATE function,   │
│                                    which has its OWN "account:" loop  │
│      case "4":                                                         │
│          break main         ◀── exits the main loop entirely            │
│      }                                                                     │
│  }                                                                           │
│                                                                                 │
│  func manageAccount(...) {                                                       │
│  account: for {              ◀── a DIFFERENT labeled loop, in a           │
│      switch choice {              different function's stack frame         │
│      case "4":                                                               │
│          break account       ◀── returns to manageAccount's CALLER          │
│                                    (the main loop), NOT to program exit        │
│      }                                                                           │
│  }                                                                                 │
│  }                                                                                   │
└───────────────────────────────────────────────────────────┘
```

This is the key idea this project adds on top of the ATM CLI: labels are
**scoped to the function they're written in** — `manageAccount`'s `account:`
label has nothing to do with `main`'s `main:` label, even though both use
`break` inside a nested `switch` for the same reason. "Back to main menu"
(exiting `manageAccount`, returning to the caller) and "Exit the whole
program" (exiting `main`'s loop) are two genuinely different actions, and the
two separate labels are what make that distinction possible without any extra
boolean flags or sentinel return values.

## Case Study: Why `sortedOwners()` Exists

```go
for owner := range b.accounts {   // DON'T rely on this order being consistent
    ...
}
```

Go **intentionally randomizes** map iteration order, specifically so nobody
accidentally writes code that depends on an order the language never promised
in the first place (older languages that iterated maps in insertion or hash-
bucket order ended up with subtle bugs when that "incidental" order changed
between versions). Any time you need a **stable, user-facing** order from a
map — like listing accounts — the idiomatic fix is exactly what
`sortedOwners()` does: pull the keys into a slice, `sort.Strings` it, then
iterate the slice instead of the map directly.

## Try It Yourself
- Add a `delete` option to the main menu (careful: deleting the map key
  `owner` you're currently looping over inside `range` is legal in Go and
  won't panic, but think through what should happen if you're mid-transfer)
- Add transaction history per account (a `[]string` field, appended to on
  every deposit/withdraw/transfer), with a "view history" option in
  `manageAccount`
- Add an account PIN, prompted once per `manageAccount` call, reusing the
  labeled-retry-loop pattern from the ATM CLI project
