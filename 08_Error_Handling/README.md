# Go for Beginners — Module 08: Error Handling

## Contents

1. **[08-error-handling.md](./08-error-handling.md)** — The `error` interface,
   custom error types, wrapping with `%w`, `errors.Is` (sentinel values,
   chain-aware) vs. `errors.As` (types, extracts structured data), and
   `panic`/`recover` (including exactly where `recover` legitimately belongs
   vs. where it doesn't). Diagrams included throughout.

2. **[validation-package/](./validation-package)** — A real `validation`
   package: sentinel errors (`ErrRequired`, `ErrTooShort`, ...) wrapped
   inside a custom `FieldError` type, merged with `errors.Join` (Go 1.20+),
   and still fully inspectable afterward via `errors.Is`/`errors.As` — plus
   a recursive `Split` helper for flattening a joined error tree into a
   field-by-field report.

3. **[retry-library/](./retry-library)** — A real `retry` package:
   exponential backoff, the functional-options pattern (tying back to Module
   03's higher-order functions), a `Permanent()` escape hatch checked via
   `errors.As` to stop retrying un-retryable failures immediately, and
   `panic`/`recover` used at a genuine boundary — protecting the retry loop
   from a panicking function.

## Suggested Order

```
Error handling guide ──▶ Validation Package ──▶ Retry Library
                           (errors.Is / errors.As,      (same tools, now driving
                            errors.Join)                  actual control-flow decisions,
                                                            plus panic/recover)
```

Validation Package focuses on *inspecting* errors after the fact (which
field failed, and how). Retry Library takes the same `errors.Is`/`errors.As`
tools and uses them to make real decisions mid-execution (stop retrying now,
or keep going) — plus adds the module's `panic`/`recover` material at a
legitimate system boundary.

## Quick Reference: Running Either Project

```bash
cd validation-package && go run .
cd retry-library && go run .
```

*Note: this module builds on Modules 00–07 — start there first if you
haven't already. `errors.Join` (used in Validation Package) requires Go
1.20+; everything else in this module works on any Go version covered so
far.*
