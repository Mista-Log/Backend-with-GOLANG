# Go for Beginners — Module 09: Modules & Packages

## Contents

1. **[09-modules-and-packages.md](./09-modules-and-packages.md)** — What's
   actually inside `go.mod` and `go.sum` (and the difference in what each is
   *for*), how Go's Minimum Version Selection resolves dependency versions,
   semantic versioning (including Go's structural `/v2`+ import-path rule
   for major versions), working with private modules (`GOPRIVATE` and
   friends), the compiler-enforced `internal/` directory convention, and a
   set of package-design habits (naming, feature-based organization over
   layered organization, avoiding import cycles, exporting deliberately).
   Diagrams included throughout.

## No Dedicated Projects This Module

This module doesn't have its own hands-on projects — by this point, several
earlier modules already produced real, multi-package, `go.mod`-backed
projects:

- **Module 03's Math Library** (`mathlib` package + demo)
- **Module 08's Validation Package** (`validation` package + demo)
- **Module 08's Retry Library** (`retry` package + demo)

The best way to apply this module is to revisit one of those with its
package-design section in mind — in particular, try moving one of them to
use an `internal/` directory for anything that isn't meant to be part of its
public API, and confirm the compiler actually enforces the boundary.

## Quick Reference: Useful Commands Covered in This Module

```bash
go mod init github.com/yourname/myapp   # create go.mod
go get github.com/gorilla/mux@v1.8.1     # add/upgrade a specific version
go get -u ./...                           # upgrade everything to latest minor/patch
go mod tidy                                # sync go.mod/go.sum with actual imports
go mod why github.com/some/pkg               # explain why a dependency is present
go mod graph                                  # print the full dependency graph
```

*Note: this module builds on Modules 00–08 — start there first if you
haven't already, especially Module 00's introduction to `go.mod`/`go mod
init`/`go mod tidy`, which this module goes considerably deeper on.*
