# 09. Modules & Packages

Module 00 introduced `go mod init` and `go get` just enough to get a project
running. This module goes deeper: what's actually inside `go.mod`/`go.sum`,
how Go's versioning works, how to work with private code, the special
meaning of an `internal/` directory, and how to actually organize a
multi-package project well.

---

## 1. `go.mod`

Every module's root has exactly one `go.mod` — it's the single source of
truth for the module's identity, minimum Go version, and dependencies.

```go
module github.com/yourname/myapp

go 1.23

require (
	github.com/gorilla/mux v1.8.1
	github.com/stretchr/testify v1.9.0
)

require golang.org/x/text v0.14.0 // indirect
```

```
┌────────────────────────────────────────────────────────┐
│   module github.com/yourname/myapp                             │
│   └── the module's IMPORT PATH — how other code refers to it,      │
│         and (by convention) where `go get` would fetch it FROM        │
│                                                                            │
│   go 1.23                                                                   │
│   └── the MINIMUM Go version this module requires — the compiler               │
│         rejects building it with an older toolchain                              │
│                                                                                        │
│   require (...)                                                                          │
│   └── every DIRECT dependency and its exact version                                        │
│                                                                                                  │
│   // indirect                                                                                      │
│   └── a dependency your code doesn't import directly, but one of YOUR              │
│         direct dependencies does — Go tracks these too, for reproducibility           │
└────────────────────────────────────────────────────────┘
```

Common commands, most already seen in Module 00, now with the rest of the
picture:
```bash
go mod init github.com/yourname/myapp   # creates go.mod
go get github.com/gorilla/mux@v1.8.1     # add/upgrade a specific version
go get -u ./...                           # upgrade everything to latest MINOR/PATCH
go mod tidy                                # add missing requires, remove unused ones
go mod why github.com/some/pkg               # explain WHY a dependency is present
go mod graph                                  # print the full dependency graph
```

---

## 2. `go.sum`

`go.sum` is a **lockfile of cryptographic checksums** — one entry per module
version your build ever touches, direct or indirect:

```
github.com/gorilla/mux v1.8.1 h1:TuBL49tXwgrFYWhqrNgrUNEY92u81SPhu7sTdzQEiWY=
github.com/gorilla/mux v1.8.1/go.mod h1:DVbg23sWSpFRCP0SfiEN6jmj59UnW/n46BH5rLB71So=
```

Its only job is **verification**, not version selection — `go.mod` decides
*which* versions to use, `go.sum` proves the code you actually downloaded
for those versions is byte-for-byte what everyone else building this module
also gets. If a dependency's contents ever changed at the same version
number (a supply-chain red flag), the checksum mismatch fails the build
loudly instead of silently trusting different code than intended.

```
┌────────────────────────────────────────────────────────┐
│   go.mod   →  WHICH versions to use                            │
│   go.sum   →  PROOF those exact versions haven't been tampered      │
│                 with, verified on every download                        │
│                                                                              │
│   Both are committed to version control. Both are meant to be                  │
│   machine-generated (`go get`/`go mod tidy`) — hand-editing go.sum                │
│   is essentially never correct.                                                     │
└────────────────────────────────────────────────────────┘
```

---

## 3. Versioning

Go's module system resolves dependency versions using **Minimum Version
Selection (MVS)**: for any dependency needed (directly or transitively) at
multiple different versions across your whole dependency graph, Go picks the
**highest** of those minimums — never silently jumping to the newest version
available on the internet unless you ask it to.

```
┌────────────────────────────────────────────────────────┐
│   Your module requires:        pkgA v1.2.0                     │
│   pkgA v1.2.0 itself requires: pkgB v1.0.0                       │
│   pkgC (also a dependency)     requires: pkgB v1.1.0                │
│                                                                          │
│   Go picks pkgB v1.1.0 — the HIGHEST of the minimums actually            │
│   required anywhere in the graph, not necessarily pkgB's latest             │
│   release on the internet (which might be v1.5.0)                             │
└────────────────────────────────────────────────────────┘
```

This is deliberately more conservative than some other ecosystems' resolvers
— **builds stay reproducible** by default, and upgrading is always an
explicit action (`go get pkg@version` or `go get -u`), never something that
happens quietly as a side effect of adding an unrelated dependency.

---

## 4. Semantic Versioning

Go module versions follow **semver**: `vMAJOR.MINOR.PATCH` (e.g., `v1.8.1`).

```
┌────────────────────────────────────────────────┐
│        v 1  .  8  .  1                                │
│          │     │     │                                    │
│        MAJOR  MINOR  PATCH                                   │
│                                                                  │
│   MAJOR — incremented for BREAKING changes                          │
│   MINOR — incremented for new, BACKWARD-COMPATIBLE features             │
│   PATCH — incremented for backward-compatible BUG FIXES                    │
└────────────────────────────────────────────────┘
```

Go enforces one semver rule structurally, not just by convention: **a module
whose major version is 2 or higher must include that major version in its
import path**, e.g. `github.com/some/pkg/v2`. This means `v1.x` and `v2.x`
of the same module are, to Go's toolchain, essentially **different modules**
that can even be imported side by side in the same program — a deliberate
design choice, since a v2 release is explicitly allowed to break
compatibility with v1, and the import path makes that break visible and
resolvable rather than silently ambiguous.

```go
import (
	oldpkg "github.com/some/pkg"      // v1.x — no version suffix
	newpkg "github.com/some/pkg/v2"    // v2.x — explicit /v2 in the path
)
```

`v0.x.y` versions are understood by convention to mean "still unstable, no
compatibility promises yet" — many real-world Go modules stay on `v0` for a
long time before committing to a `v1` API contract.

---

## 5. Private Modules

By default, `go get` fetches from the public Go module proxy
(`proxy.golang.org`) and verifies checksums against a public checksum
database (`sum.golang.org`) — neither of which can see or verify a private
company repository. Two environment variables tell the toolchain to bypass
both for matching module paths:

```bash
export GOPRIVATE="github.com/yourcompany/*"
# or, more granularly:
export GONOSUMCHECK="github.com/yourcompany/*"   # skip checksum DB only
export GOPROXY="https://proxy.golang.org,direct"  # fall back to a direct
                                                     # VCS fetch for anything
                                                     # not found on the proxy
```

```
┌────────────────────────────────────────────────────────┐
│   go get github.com/yourcompany/internal-lib                     │
│                                                                        │
│   WITHOUT GOPRIVATE:  tries the public proxy first → fails/hangs        │
│                         (the repo isn't public) or leaks the module's      │
│                         existence to a third party                            │
│                                                                                    │
│   WITH GOPRIVATE set: skips the public proxy AND checksum database for      │
│                         matching paths, fetches directly via your normal        │
│                         git credentials (SSH key, access token, etc.)              │
└────────────────────────────────────────────────────────┘
```

You'll also typically need normal `git`-level authentication configured
(an SSH key, or a `.netrc`/credential helper for HTTPS) — `GOPRIVATE` tells
Go *not* to route through the public infrastructure; it doesn't grant access
by itself.

---

## 6. Internal Packages

Any package living inside a directory literally named `internal/` gets a
**compiler-enforced** visibility rule: it can only be imported by code
rooted at (or below) the parent of that `internal/` directory. This isn't a
convention or a linter suggestion — the `go build` toolchain itself refuses
the import.

```
myapp/
├── go.mod                     module github.com/you/myapp
├── main.go
├── internal/
│   └── billing/
│       └── billing.go          package billing
└── pkg/
    └── publicapi/
        └── publicapi.go          package publicapi
```

```go
// main.go — INSIDE myapp, so this import is ALLOWED:
import "github.com/you/myapp/internal/billing"

// some OTHER module trying the same import — REJECTED at build time:
// "use of internal package github.com/you/myapp/internal/billing not allowed"
```

```
┌──────────────────────────────────────────────────────┐
│   github.com/you/myapp/                                    │
│       internal/billing        ◀── importable ONLY from            │
│                                     github.com/you/myapp/...           │
│                                     and its subdirectories                │
│                                                                                │
│   Anyone outside that tree — a different module entirely, or                    │
│   even a DIFFERENT top-level package within a monorepo that                        │
│   isn't under myapp/ — gets a hard compile error, not just a                          │
│   documentation note saying "please don't import this."                                  │
└──────────────────────────────────────────────────────┘
```

This is the single most useful tool for a growing codebase: it lets you
build genuinely shared internal helpers (database connection pooling,
internal auth logic, shared config parsing) without accidentally creating
a public API surface that other teams — or external consumers of your
module — start depending on and that you can then never change without
breaking someone.

---

## 7. Package Design

A few habits that separate a well-organized Go project from an
awkward one:

**Package names are short, lowercase, and singular** — `user`, not `Users`
or `user_utils`. The convention is that callers write `user.New(...)`, so
the package name itself is effectively a prefix; a name like `utils` or
`common` tends to become a dumping ground precisely because it doesn't
describe anything specific.

**A package's name shouldn't repeat in its own identifiers** — inside
package `user`, write `user.New()` and `user.Validate()`, not
`user.NewUser()` and `user.ValidateUser()`. From the *caller's* side,
`user.NewUser()` reads redundantly (`user.NewUser` says "user" twice); `user.New()`
reads cleanly.

**Group by what a package *does*, not by what *kind of thing* it is.** A
`models/`, `handlers/`, `services/` split (organizing by technical layer)
tends to force every real feature to spread across three-plus packages and
invites import cycles once those layers need to reference each other. Most
idiomatic Go codebases instead group by domain/feature — a `billing/`
package containing its own types, its own logic, and its own handlers
together — which keeps related code physically close and each package's
purpose obvious from its name alone.

**Keep the dependency graph flowing one direction.** Go doesn't allow
import cycles at all (package A importing B importing A is a compile
error), so package design in Go tends to naturally push you toward a clean,
layered dependency graph — but it's still possible to design yourself into
a painful *near*-cycle (two packages that keep needing "just one more thing"
from each other). When that happens, it's usually a sign a third package
should be extracted holding whatever both sides actually share.

**Export deliberately, not by default.** Recall Module 03's capitalization
rule — every exported name is effectively a promise to every caller, inside
or outside your codebase, that its behavior is stable. Keep helper functions
and internal types lowercase unless something outside the package genuinely
needs them; it's far easier to export something later (backward compatible)
than to un-export something already relied upon (a breaking change).

```
┌────────────────────────────────────────────────────────┐
│   Organize by FEATURE, not by LAYER:                            │
│                                                                     │
│   ❌ layered                        ✅ feature-based                  │
│   /models                            /billing                          │
│     invoice.go                         invoice.go   (type + logic)       │
│     user.go                            handlers.go   (its own handlers)    │
│   /handlers                          /user                                    │
│     invoice_handler.go                 user.go                                  │
│     user_handler.go                    handlers.go                                │
│   /services                                                                            │
│     invoice_service.go              Related code stays PHYSICALLY together,               │
│     user_service.go                  and each package's purpose is obvious                  │
│                                       just from its name.                                       │
└──────────────────────────────────────────────────────────┘
```

---

This module has no dedicated projects of its own — everything from Module 00
onward has already been organized as real, `go.mod`-backed modules (several
with their own internal library packages, like Module 03's `mathlib` and
Module 08's `validation`/`retry`), so the best next step is revisiting one of
those with this module's package-design habits in mind, rather than building
something new from scratch.
