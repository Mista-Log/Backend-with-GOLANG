# Project 1 — Validation Package

```
validation-package/
├── go.mod
├── main.go              package main — demo
└── validation/
    └── validation.go     package validation — the actual library
```

```bash
cd validation-package
go run .
```

## What's Demonstrated Here

- **A custom error type (`FieldError`) wrapping a sentinel error** — every
  validator returns either `nil` or a `*FieldError{Field, Err}`, where `Err`
  is one of the package's sentinel errors (`ErrRequired`, `ErrTooShort`,
  ...), sometimes further wrapped with `%w` to add specific numbers (see
  `MinLength`).
- **`FieldError.Unwrap() error`** — the one method that makes `errors.Is`
  and `errors.As` able to see *through* `FieldError` to the sentinel
  underneath. Without it, `FieldError` would be a dead end in the chain.
- **`errors.Join`** (Go 1.20+) — `All` merges every failing validator's
  error into one, but critically, `errors.Is`/`errors.As` still work
  correctly against the *merged* result, checking every branch, not just
  the first one.
- **`errors.Is` for "did this SPECIFIC kind of failure happen anywhere?"**
  and **`errors.As` for "give me the first FieldError so I can read its
  Field"** — used side by side in `main.go` so the difference stays
  concrete.
- **A recursive `Split` helper** — flattens a joined error tree back into a
  plain `[]error`, useful for building a field-by-field error report (like
  an API's JSON error response) instead of checking for one specific
  failure at a time.

```
┌──────────────────────────────────────────────────────────┐
│   validation.All(                                                │
│       Required("Name", "A")        ──▶ nil (passes)                 │
│       MinLength("Name", "A", 2)     ──▶ &FieldError{Name, ErrTooShort} │
│       Email("Email", "bad")          ──▶ &FieldError{Email, ErrInvalidEmail}│
│       Range("Age", 5, 13, 120)        ──▶ &FieldError{Age, ErrOutOfRange}    │
│   )                                                                              │
│        │                                                                           │
│        ▼                                                                             │
│   errors.Join(...)  →  ONE error value, but errors.Is/As still reach EACH             │
│                          individual FieldError inside it                                │
└──────────────────────────────────────────────────────────┘
```

## Case Study: Why Sentinel + Wrap, Instead of Just a Message String

An easier-looking `FieldError{Field, Message string}` (just a plain string
for the reason) would work fine for *displaying* errors, but it would make
`errors.Is(err, someSpecificFailure)` impossible — string comparison is
fragile (message wording changes break every caller checking against it) and
isn't what `errors.Is` does anyway. Wrapping a real sentinel error value
means:
- **Display still works** — `FieldError.Error()` still produces a normal,
  readable string via `fmt.Sprintf("%s: %s", e.Field, e.Err)`.
- **Programmatic checks are robust** — `errors.Is(err, validation.ErrOutOfRange)`
  keeps working even if you later reword `ErrOutOfRange`'s message, add more
  detail with `%w`, or nest the error inside several more layers of wrapping.
- **`errors.As` still gets you the structured `Field` name** when you need
  it, independent of whichever specific sentinel is underneath.

## Try It Yourself
- Add a `Numeric(field, value string) error` validator with its own sentinel
  (`ErrNotNumeric`), and add it to `ValidateSignup` for a new `Phone` field
- Change `Split` to return `map[string][]error` keyed by field name instead
  of a flat slice — useful for a real API's `{"errors": {"email": [...]}}`
  response shape
- Add a `MustBeUnique(field string, exists func(string) bool) error`
  validator that takes a function (a callback, tying back to Module 03's
  higher-order functions) to check uniqueness against, e.g., a database
