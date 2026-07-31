# 00. Setup & Environment — Go for Beginners

Welcome to Go! This guide covers everything in your module outline, with diagrams and
hands-on examples. Work through it top to bottom — each section builds on the last.

---

## 1. Introduction

Go (also called "Golang") is a compiled, statically-typed language created at Google in
2009 by Rob Pike, Ken Thompson, and Robert Griesemer. It was designed to fix pain points
of large-scale software engineering: slow builds, unclear dependencies, and clunky
concurrency.

**Why people learn Go:**
- Compiles to a single static binary — no runtime or VM needed to deploy
- Built-in concurrency via goroutines and channels
- Small, simple language spec (you can learn the whole syntax in a weekend)
- Fast compile times, garbage collected, strongly typed
- Dominant in cloud infrastructure — Docker, Kubernetes, Terraform, and most of the
  modern DevOps toolchain are written in Go

```
┌─────────────────────────────────────────────────────────┐
│                     Go Toolchain Flow                    │
│                                                            │
│   main.go  ──▶  go build  ──▶  single binary  ──▶  run   │
│                     │                                     │
│                     ▼                                     │
│           (no external runtime needed —                   │
│            unlike Python/Node/JVM)                        │
└─────────────────────────────────────────────────────────┘
```

---

## 2. Installing Go

### macOS
```bash
brew install go
```

### Linux (Debian/Ubuntu)
```bash
# Download the latest tarball from go.dev/dl, then:
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.23.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

### Windows
Download the `.msi` installer from **go.dev/dl** and run it. It sets up `PATH`
automatically.

### Verify
```bash
go version
# go version go1.23.0 linux/amd64
```

---

## 3. GOPATH vs GOROOT

This trips up almost every beginner, so here's the mental model:

```
┌────────────────────────────┐   ┌────────────────────────────┐
│          GOROOT             │   │           GOPATH            │
│  "Where Go itself lives"     │   │  "Where YOUR Go work lives"  │
│                              │   │                              │
│  /usr/local/go               │   │  ~/go                       │
│   ├── bin/  (go, gofmt)      │   │   ├── bin/  (installed CLIs)│
│   ├── src/  (stdlib source)  │   │   ├── pkg/  (compiled cache)│
│   └── pkg/                   │   │   └── src/  (legacy, pre-  │
│                              │   │        modules code)        │
│  You almost NEVER touch this │   │  Mostly obsolete since Go   │
│  by hand.                    │   │  Modules (2018+), but       │
│                              │   │  `go install` still puts    │
│                              │   │  binaries in $GOPATH/bin    │
└────────────────────────────┘   └────────────────────────────┘
```

- **GOROOT** = where the Go installation itself (compiler, stdlib) lives. Set once,
  rarely touched.
- **GOPATH** = historically, where all your Go code *had* to live (`$GOPATH/src/...`).
  Since **Go Modules** (Go 1.11+, default since 1.16), you no longer need your code
  inside GOPATH. GOPATH today mostly just matters for:
  - `$GOPATH/bin` — where `go install` puts compiled CLI tools
  - `$GOPATH/pkg/mod` — the module download cache

Check both any time:
```bash
go env GOROOT
go env GOPATH
```

---

## 4. Go Modules

Modules are Go's dependency management system — think `package.json` + `npm`, or
`requirements.txt` + `pip`, bundled into the `go` CLI itself.

```bash
mkdir myapp && cd myapp
go mod init github.com/yourname/myapp   # creates go.mod
go get github.com/gorilla/mux           # adds a dependency
go mod tidy                              # cleans up unused/missing deps
```

**`go.mod`** — declares your module and dependencies:
```go
module github.com/yourname/myapp

go 1.23

require github.com/gorilla/mux v1.8.1
```

**`go.sum`** — cryptographic checksums of every dependency, for reproducible builds
(like a lockfile). You never edit this by hand.

```
┌──────────────────────────────────────────────────────────┐
│                   Module Resolution Flow                  │
│                                                             │
│   go build/run                                             │
│        │                                                   │
│        ▼                                                   │
│   reads go.mod ──▶ checks $GOPATH/pkg/mod (cache)          │
│        │                     │                             │
│        │              found? │  not found?                 │
│        │                     ▼        ▼                    │
│        │              use cached   download from proxy     │
│        │                            (proxy.golang.org)      │
│        └──────────────▶ verify against go.sum ──▶ compile  │
└──────────────────────────────────────────────────────────┘
```

---

## 5. VS Code

1. Install the **Go extension** (by the Go Team at Google) from the marketplace.
2. Open your project folder — VS Code will prompt to install supporting tools
   (`gopls`, `dlv`, `staticcheck`, etc.). Click **Install All**.
3. Key features you get:
   - Autocomplete + inline docs via `gopls` (Go's official language server)
   - Format-on-save using `gofmt`
   - Inline debugging via Delve (see below)
   - "Run test" / "Run" lens above every `func Test...` and `func main()`

Recommended `settings.json` additions:
```json
{
  "go.formatTool": "gofmt",
  "go.lintTool": "staticcheck",
  "editor.formatOnSave": true
}
```

## 6. GoLand

JetBrains' dedicated Go IDE. Heavier than VS Code but has deeper refactoring tools,
a built-in debugger UI, and excellent test-coverage visualization out of the box —
no extension installs needed. Good choice if you're already in the JetBrains
ecosystem (IntelliJ, PyCharm, etc.) and want one consistent UI.

---

## 7. CLI

The `go` command is your one entry point for almost everything:

| Command | Purpose |
|---|---|
| `go run main.go` | Compile + execute immediately (no binary left behind) |
| `go build` | Compile to a binary in the current directory |
| `go install` | Compile + install the binary into `$GOPATH/bin` |
| `go fmt ./...` | Auto-format all files |
| `go vet ./...` | Static analysis — catches suspicious code |
| `go test ./...` | Run all tests |
| `go mod tidy` | Sync go.mod with actual imports |
| `go doc fmt.Println` | Show docs for a function/package right in the terminal |

---

## 8. Debugger

Go programs can be debugged two main ways:

1. **`fmt.Println` / `log.Println` debugging** — quick and dirty, totally normal for
   small programs.
2. **Real breakpoint debugging** — via **Delve**, either from the terminal or through
   your editor's debug UI (VS Code / GoLand both wrap Delve under the hood).

## 9. Delve

Delve (`dlv`) is Go's purpose-built debugger (regular gdb doesn't understand
goroutines well).

```bash
go install github.com/go-delve/delve/cmd/dlv@latest

dlv debug main.go
```

Inside the Delve prompt:
```
(dlv) break main.main       # set a breakpoint
(dlv) continue              # run until breakpoint
(dlv) next                  # step over
(dlv) step                  # step into
(dlv) print myVariable      # inspect a variable
(dlv) goroutines            # list all running goroutines
```

---

## 10. Go Workspace

A **Go workspace** (`go.work`, Go 1.18+) lets you develop across *multiple modules*
locally at once — e.g., a library and the app consuming it — without publishing the
library first.

```bash
mkdir myworkspace && cd myworkspace
go work init ./mylib ./myapp
```

`go.work`:
```
go 1.23

use (
    ./mylib
    ./myapp
)
```

Now `myapp` resolves `mylib` from your local disk instead of the network — useful
during active co-development, not needed for single-module projects (like the ones
below).

---

## 11. Environment Variables

Check everything Go knows about your setup:
```bash
go env
```

Commonly touched ones:

| Variable | Meaning |
|---|---|
| `GOOS` | Target operating system (`linux`, `darwin`, `windows`) |
| `GOARCH` | Target CPU architecture (`amd64`, `arm64`) |
| `GO111MODULE` | Legacy toggle for modules (`on` by default now, rarely needed) |
| `GOPROXY` | Where `go get` downloads modules from (default: `proxy.golang.org`) |
| `CGO_ENABLED` | Whether cgo (calling C code) is allowed (`1`/`0`) |

Set one for a single command:
```bash
GOOS=windows GOARCH=amd64 go build -o app.exe main.go
```

---

## 12. Build Commands

```bash
go build                      # binary named after the module/folder
go build -o bin/app main.go   # custom output path/name
go build -ldflags="-s -w"     # strip debug info, smaller binary
go build -race                # instrument for the race detector (dev only)
```

---

## 13. Cross Compilation

This is one of Go's superpowers — no extra toolchain needed to target another OS/CPU:

```
┌────────────────────────────────────────────────────────────┐
│              Cross-Compiling FROM macOS (M-series)          │
│                                                               │
│   GOOS=linux   GOARCH=amd64 go build -o app-linux    main.go│
│   GOOS=windows GOARCH=amd64 go build -o app.exe      main.go│
│   GOOS=darwin  GOARCH=arm64 go build -o app-mac-arm  main.go│
│                                                               │
│   Each command produces a NATIVE binary for that target —   │
│   no Docker, no VM, no extra SDK required.                  │
└────────────────────────────────────────────────────────────┘
```

Full target list: `go tool dist list`

---

You now have a working Go setup end to end. Next: the three projects, in increasing
order of difficulty — **Hello World → Calculator CLI → File Organizer**.
