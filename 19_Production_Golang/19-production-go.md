# 19. Production Go

Every module before this taught a language feature or a library. This one
is different: it's about the habits and structural decisions that separate
a working Go program from one a team can safely operate, extend, and debug
at 3 AM when something breaks.

---

## Project Structure

Go doesn't enforce a project layout the way some frameworks do, but a
de facto standard has emerged across the ecosystem — the
[`golang-standards/project-layout`](https://github.com/golang-standards/project-layout)
conventions, used (in whole or part) by most serious Go projects:

```
myservice/
├── cmd/
│   └── server/
│       └── main.go          ◀── the ENTRY POINT — thin, just wires things together
├── internal/                   ◀── Module 09's compiler-enforced privacy: nothing
│   ├── config/                    outside this module can import anything under here
│   ├── domain/                  the CORE business types and logic (entities, rules)
│   ├── service/                   use-case / business-logic orchestration
│   ├── repository/                 data access implementations
│   └── transport/
│       └── http/                    HTTP handlers, routing, middleware
├── pkg/                         ◀── code SAFE for other modules to import (rarer —
│                                     most projects need little or nothing here)
├── migrations/                 (Module 17)
├── go.mod
└── go.sum
```

```
┌──────────────────────────────────────────────────────────┐
│   cmd/server/main.go   →  ONLY wiring: load config, construct              │
│                              dependencies, start the server, wait for          │
│                              shutdown. NO business logic lives here.               │
│                                                                                        │
│   internal/            →  everything that's genuinely "this project's own,"             │
│                              protected by Module 09's compiler-enforced rule                │
│                                                                                                  │
│   pkg/                  →  reserved for code you WANT other projects/modules                      │
│                              to import — most services never need this at all                        │
└──────────────────────────────────────────────────────────┘
```

**`cmd/` supports multiple binaries from one module** — a service and a
companion CLI admin tool, for instance, each getting their own
`cmd/<name>/main.go`, sharing all the same `internal/` code.

---

## Clean Architecture

A layering discipline (popularized by Robert C. Martin) built around one
core rule: **dependencies point inward, never outward** — your core
business logic should never import a framework, a database driver, or an
HTTP library.

```
┌──────────────────────────────────────────────────────────┐
│                                                                  │
│         ┌─────────────────────────────────────────┐               │
│         │         Frameworks & Drivers                │  ◀── outermost:    │
│         │    (HTTP handlers, database, CLI)              │       Gin, sqlx,     │
│         │    ┌─────────────────────────────────┐        │       cobra, etc.       │
│         │    │      Interface Adapters             │        │                          │
│         │    │  (repository implementations,         │        │                          │
│         │    │   HTTP request/response mapping)         │        │                          │
│         │    │    ┌─────────────────────────┐        │        │                          │
│         │    │    │     Use Cases                │        │        │                          │
│         │    │    │  (application-specific         │        │        │                          │
│         │    │    │   business rules)                 │        │        │                          │
│         │    │    │    ┌─────────────────┐        │        │        │                          │
│         │    │    │    │    Entities         │        │        │        │                          │
│         │    │    │    │ (core business        │        │        │        │                          │
│         │    │    │    │  objects & rules)        │        │        │        │                          │
│         │    │    │    └─────────────────┘        │        │        │                          │
│         │    │    └─────────────────────────┘        │        │                          │
│         │    └─────────────────────────────────┘        │                          │
│         └─────────────────────────────────────────┘                          │
│                                                                                    │
│   DEPENDENCIES only ever point INWARD (outer rings depend on inner ones,             │
│   never the reverse) — Entities know NOTHING about Use Cases, which know                │
│   NOTHING about HTTP or your database.                                                    │
└──────────────────────────────────────────────────────────┘
```

This is Module 06's interfaces and Module 17's repository pattern, applied
at the scale of a whole application: your core logic depends on
*interfaces* it defines, and the outer layers *implement* those interfaces
— exactly the inversion that lets your business rules be tested (Module 14)
without a real database or HTTP server anywhere in the loop.

---

## Hexagonal (Ports and Adapters)

A close cousin of Clean Architecture, often described as functionally
equivalent — same inward-pointing-dependency principle, different
vocabulary and a slightly different visual framing:

```
┌──────────────────────────────────────────────────────────┐
│                                                                  │
│                    ┌───────────────────────┐                       │
│        HTTP ───────▶│                         │◀─────── CLI            │
│      (adapter)      │                         │      (adapter)            │
│                    │      CORE DOMAIN            │                          │
│    Database ───────▶│      (the "hexagon")          │◀─────── Message           │
│    (adapter)        │                             │        Queue                 │
│                    │                               │      (adapter)                 │
│  Test Fakes ───────▶│                                 │◀─────── External API             │
│  (adapter)          └───────────────────────┘        (adapter)                        │
│                                                                                              │
│   "PORTS" are the interfaces the core domain DEFINES (what it needs FROM                       │
│   the outside world, and what it OFFERS to it). "ADAPTERS" are the concrete                       │
│   implementations plugged into those ports — swap any adapter (a real                                │
│   database for a test fake, an HTTP API for a CLI) without touching the                                │
│   core at all.                                                                                            │
└──────────────────────────────────────────────────────────┘
```

The practical Go implementation of both Clean and Hexagonal architecture
looks nearly identical: interfaces defined near your core logic,
implementations living in outer packages that depend on (import) the core
— never the other way around. Don't get hung up on which name to call it;
the **inward-dependency rule** is the part that actually matters.

---

## DDD (Domain-Driven Design)

A broader methodology for modeling complex business domains, contributing
a few concepts that show up constantly in Go codebases regardless of
whether a team calls itself "doing DDD":

```
┌────────────────────────────────────────────────────┐
│   ENTITY           → has a distinct IDENTITY that persists over            │
│                        time, even as its attributes change                    │
│                        (an Order — the same order, even after its                │
│                         status changes from "pending" to "shipped")                │
│                                                                                        │
│   VALUE OBJECT       → has NO identity — defined entirely by its                        │
│                          attributes; two with the same values ARE                          │
│                          the same thing (Module 05's comparable structs:                      │
│                          a Money{amount: 500, currency: "USD"} is                                │
│                          interchangeable with any other identical one)                              │
│                                                                                                          │
│   AGGREGATE            → a cluster of entities/value objects treated                                       │
│                            as ONE unit for data changes, with a single                                        │
│                            "aggregate root" controlling access (an Order                                         │
│                            aggregate might contain OrderLine value objects —                                        │
│                            you never modify a line directly, only through                                              │
│                            the Order)                                                                                     │
│                                                                                                                              │
│   BOUNDED CONTEXT        → a boundary within which a specific model and                                                       │
│                              vocabulary applies consistently — "Customer"                                                        │
│                              might mean something different in your Billing                                                         │
│                              context than in your Shipping context, and                                                                │
│                              that's fine, AS LONG AS each context is internally                                                            │
│                              consistent and the translation between them                                                                      │
│                              is explicit                                                                                                          │
└────────────────────────────────────────────────────┘
```

In practice, DDD's biggest everyday contribution to Go code is discipline
about **where business rules live**: inside entity/aggregate methods
(`order.Cancel()`, which enforces "can't cancel an already-shipped order"
internally), not scattered across handlers or database queries — directly
continuous with Module 06's methods-carry-behavior habit.

---
## Configuration

Production services need settings that vary by environment (database URL,
log level, port) without code changes between them. A typical Go config
struct, populated from multiple sources with a clear precedence order:

```go
type Config struct {
	Port        int    `mapstructure:"port"`
	DatabaseURL string `mapstructure:"database_url"`
	LogLevel    string `mapstructure:"log_level"`
}
```

```
┌──────────────────────────────────────────────────────────┐
│              Typical precedence, highest wins                       │
│                                                                          │
│   1. Command-line flags        (--port=9000)          HIGHEST PRIORITY    │
│   2. Environment variables      (PORT=9000)                                  │
│   3. Config file                 (config.yaml: port: 9000)                      │
│   4. Hardcoded defaults           (port: 8080)          LOWEST PRIORITY            │
│                                                                                         │
│   This lets the SAME binary run correctly in dev (config file), CI                        │
│   (env vars), and production (a mix, often env vars + secrets) without                       │
│   any code changes — just different configuration SOURCES.                                      │
└──────────────────────────────────────────────────────────┘
```

---

## Environment Variables

The simplest, most universal configuration mechanism — and the backbone of
the ["twelve-factor app"](https://12factor.net) methodology's config
principle: **store config in the environment**, strictly separate from code.

```go
port := os.Getenv("PORT")
if port == "" {
	port = "8080" // a sensible default
}

// Go 1.21+ convenience for exactly this pattern... doesn't exist natively,
// but it's common enough that most config libraries (Section "Libraries"
// below) build this "env var, or a default" lookup in directly.
```

```
┌────────────────────────────────────────────────────┐
│   Why environment variables specifically (not just a config          │
│   file) for PRODUCTION settings:                                        │
│                                                                              │
│   - No code or config FILE needs to change between environments —             │
│      the same built binary/image runs everywhere                                │
│   - Never accidentally committed to version control (a config FILE                 │
│      with real secrets in it is a classic way secrets leak into git history)          │
│   - Every deployment platform (Docker, Kubernetes, systemd, cloud             │
│      hosting) has first-class support for setting them per-environment            │
└────────────────────────────────────────────────────┘
```

---

## Dependency Injection

**Dependency injection** means a component receives its dependencies from
the outside (typically via constructor parameters) instead of constructing
them itself — you've been doing this since Module 06's
`NewGateway(processor PaymentProcessor)` and every `New*` constructor since.

```go
// Manual DI — completely ordinary Go, no framework needed:
func main() {
	cfg := loadConfig()
	db := connectDB(cfg.DatabaseURL)
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	handler := transport.NewUserHandler(userService)

	http.ListenAndServe(cfg.Port, handler)
}
```

```
┌──────────────────────────────────────────────────────────┐
│   main() constructs EVERYTHING, in DEPENDENCY ORDER, and passes            │
│   each thing into whatever needs it — this is "manual" or                    │
│   "constructor" dependency injection, and it's genuinely how MOST               │
│   Go services are wired, with no DI framework at all.                              │
│                                                                                        │
│   As a project grows to dozens of components, this wiring code can                       │
│   get long and repetitive — that's the specific problem wire and fx                         │
│   (Libraries section below) exist to help with, NOT a fundamentally                            │
│   different way of doing DI.                                                                      │
└──────────────────────────────────────────────────────────┘
```

---

## Logging

Module 16 introduced `log/slog` for structured logging; production
services extend this with **log levels** (so verbosity is adjustable
without redeploying) and consistent, structured fields across every log
line:

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo, // DEBUG logs are suppressed unless this is lowered
}))

logger.Info("order processed", "orderID", order.ID, "amount", order.Total)
logger.Error("payment failed", "orderID", order.ID, "error", err)
```

```
┌────────────────────────────────────────────────────┐
│   DEBUG  → verbose, developer-only detail                                 │
│   INFO    → normal operational events worth recording                       │
│   WARN     → something unexpected, but not yet a failure                       │
│   ERROR     → an operation failed                                                │
│                                                                                       │
│   Production services typically run at INFO or WARN — DEBUG is turned              │
│   on TEMPORARILY while investigating a specific issue, then turned                    │
│   back down, since it's usually too voluminous for continuous use.                       │
└────────────────────────────────────────────────────┘
```

---

## Observability

The "three pillars," each answering a different question when something
goes wrong:

```
┌──────────────────────────────────────────────────────────┐
│   LOGS      → "what HAPPENED?" — discrete events, often with              │
│                 rich context, but expensive to search across                 │
│                 huge volumes                                                    │
│                                                                                     │
│   METRICS    → "what's the current STATE/TREND?" — numeric                          │
│                  time series (request rate, error rate, latency                        │
│                  percentiles) — cheap to store and query even at                          │
│                  huge scale, but low detail per data point                                   │
│                                                                                                   │
│   TRACES      → "what PATH did this ONE request take?" — follows a                                  │
│                   single request across multiple services/functions,                                   │
│                   showing exactly where time was spent                                                    │
│                                                                                                                │
│   Together: metrics tell you SOMETHING is wrong and roughly where;                                               │
│   traces show you the exact request that hit it; logs give you the                                                  │
│   detailed story of what that request actually did.                                                                    │
└──────────────────────────────────────────────────────────┘
```

---

## Metrics

`prometheus/client_golang` is the dominant Go metrics library, exposing an
HTTP endpoint Prometheus (or a compatible system) periodically scrapes:

```go
import "github.com/prometheus/client_golang/prometheus"

var requestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{Name: "http_requests_total"},
	[]string{"method", "path", "status"},
)

func init() {
	prometheus.MustRegister(requestsTotal)
}

// inside a middleware, after each request:
requestsTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(status)).Inc()
```

```
┌────────────────────────────────────────────────────┐
│   Counter    → only ever goes UP (total requests, total errors)          │
│   Gauge       → goes up AND down (current connections, queue depth)          │
│   Histogram    → tracks a DISTRIBUTION (request latency buckets — lets            │
│                    you compute p50/p95/p99 percentiles later)                        │
└────────────────────────────────────────────────────┘
```

---

## Health Checks

Two conventionally distinct endpoints, since they answer genuinely
different questions that orchestrators (Kubernetes, load balancers) act on
differently:

```go
mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK) // "is the PROCESS alive at all?"
})

mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(); err != nil { // "is it ready to serve REAL traffic?"
		http.Error(w, "database unreachable", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
})
```

```
┌────────────────────────────────────────────────────┐
│   /healthz (LIVENESS)  → fails ONLY if the process itself is             │
│                            broken/deadlocked — Kubernetes RESTARTS            │
│                            the container on repeated failure                     │
│                                                                                       │
│   /readyz  (READINESS)   → fails if a DEPENDENCY (database, cache,                     │
│                              downstream service) isn't reachable — the                     │
│                              orchestrator STOPS ROUTING traffic here                          │
│                              temporarily, WITHOUT restarting the container,                        │
│                              since the process itself is fine                                          │
└────────────────────────────────────────────────────┘
```

Conflating these is a common mistake: a liveness check that also verifies
the database causes Kubernetes to endlessly restart a perfectly healthy
process just because the *database* is temporarily down — restarting the
app does nothing to fix that, and makes recovery slower once the database
comes back.

---

## Graceful Shutdown

When a process receives a termination signal (`SIGTERM` — how Kubernetes
and most process managers ask a process to stop), a production service
should finish in-flight requests before actually exiting, not drop them
mid-response.

```go
srv := &http.Server{Addr: ":8080", Handler: mux}

go func() {
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}()

sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
<-sigCh // BLOCKS here until a signal arrives

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
srv.Shutdown(ctx) // stops accepting NEW connections, waits for IN-FLIGHT ones,
                    // up to the context's deadline
```

```
┌──────────────────────────────────────────────────────────┐
│   SIGTERM received                                                    │
│        │                                                                  │
│        ▼                                                                    │
│   srv.Shutdown(ctx)                                                            │
│        │                                                                          │
│        ├─▶ stop accepting NEW connections immediately                               │
│        ├─▶ let IN-FLIGHT requests finish naturally                                     │
│        └─▶ if they don't finish within ctx's timeout, force-close anyway                  │
│                                                                                               │
│   This is Module 13's context deadline pattern, applied to the whole                           │
│   PROCESS lifecycle instead of one request — "give in-flight work a                               │
│   bounded amount of time to wrap up, then move on regardless."                                       │
└──────────────────────────────────────────────────────────┘
```

---

## Secrets

Passwords, API keys, and signing keys (Module 18's JWT secret) need
handling distinct from ordinary configuration:

```
┌────────────────────────────────────────────────────┐
│   NEVER commit secrets to version control — not even in a               │
│     "private" repo, and not even temporarily "just for now"                  │
│                                                                                   │
│   NEVER log secrets — a stray fmt.Println(cfg) that includes an                     │
│     API key ends up in log aggregation systems, often retained                        │
│     far longer and viewed by more people than intended                                   │
│                                                                                                │
│   PREFER a secrets manager (AWS Secrets Manager, HashiCorp Vault,                                │
│     Google Secret Manager) or your platform's built-in secret                                       │
│     injection (Kubernetes Secrets) over plain environment variables                                    │
│     where available — these add access auditing, rotation, and                                            │
│     encryption at rest that plain env vars don't provide on their own                                        │
│                                                                                                                    │
│   ROTATE secrets periodically, and IMMEDIATELY if one is ever                                                        │
│     suspected to have leaked (an accidental commit, a compromised                                                        │
│     laptop) — Module 18's refresh-token-family revocation is the                                                            │
│     same underlying instinct: assume compromise, cut off everything                                                            │
│     descended from it, force a clean re-issue                                                                                     │
└────────────────────────────────────────────────────┘
```

---

Onto the Libraries section — the concrete tools most real Go production
services reach for to implement everything above, followed by a reference
project structure tying every topic together into one runnable skeleton.
## Libraries

### zap

Uber's high-performance structured logger — historically the most common
choice specifically for performance-sensitive services, since it avoids
reflection and allocations far more aggressively than most alternatives.

```go
import "go.uber.org/zap"

logger, _ := zap.NewProduction() // JSON output, INFO level, production defaults
defer logger.Sync()

logger.Info("order processed",
	zap.Int("orderID", order.ID),
	zap.Float64("amount", order.Total),
)
```

Since Go 1.21, `log/slog` (standard library, Module 16) covers most of what
zap historically was needed for, with no extra dependency — many new
projects now default to `slog` unless they have specifically measured a
need for zap's extra performance.

### logrus

An older, once extremely popular structured logger, notable today mainly
because it's still present in a huge number of existing codebases —
**the project itself is now in maintenance mode**, and its own README
recommends new projects consider `slog` or `zap` instead.

```go
import "github.com/sirupsen/logrus"

log := logrus.New()
log.WithFields(logrus.Fields{"orderID": order.ID}).Info("order processed")
```

### viper

The dominant configuration library, handling exactly the multi-source
precedence chain from this module's Configuration section (flags → env
vars → config file → defaults) in one place:

```go
import "github.com/spf13/viper"

viper.SetDefault("port", 8080)
viper.SetConfigName("config")
viper.AddConfigPath(".")
viper.AutomaticEnv() // env vars automatically override file/default values

viper.ReadInConfig()
port := viper.GetInt("port")
```

### cobra

The standard library for building CLI applications with subcommands, flags,
and help text — this is what `kubectl`, `docker`, `hugo`, and most serious
Go CLI tools are built with.

```go
import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "myservice",
	Short: "A production service",
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	Run: func(cmd *cobra.Command, args []string) {
		runMigrations()
	},
}

func main() {
	rootCmd.AddCommand(migrateCmd)
	rootCmd.Execute()
}
```
```bash
myservice migrate          # runs the subcommand above
myservice --help            # auto-generated help text
```
`viper` and `cobra` are frequently used together (`viper` for config,
`cobra` for the CLI shell around it) — both from the same author, designed
to interoperate directly.

### wire

A **compile-time** dependency injection code generator from Google —
you describe which constructors produce which types, and `wire` generates
the actual "manual DI" wiring code (exactly the `main.go` style from this
module's Dependency Injection section) for you, rather than doing it at
runtime via reflection.

```go
// wire.go (not compiled into the final binary — a code-gen input)
func InitializeService() (*Service, error) {
	wire.Build(NewConfig, NewDB, NewUserRepository, NewUserService)
	return nil, nil // wire replaces this with REAL generated code
}
```
```bash
wire ./...   # generates wire_gen.go with the actual constructor calls, in order
```
The generated code is plain, readable Go — `wire`'s value is entirely in
**not hand-writing and maintaining** that wiring as a project's dependency
graph grows, while keeping everything resolved at compile time (a
missing/mismatched dependency is a build error, not a runtime panic).

### fx

Uber's **runtime** dependency injection framework — instead of generating
code, you register constructors and `fx` builds and wires the object graph
when the application starts, using reflection.

```go
import "go.uber.org/fx"

func main() {
	fx.New(
		fx.Provide(NewConfig, NewDB, NewUserRepository, NewUserService),
		fx.Invoke(StartServer),
	).Run()
}
```

```
┌──────────────────────────────────────────────────────────┐
│   Manual DI:  you write ALL the wiring code yourself, by hand              │
│                                                                                 │
│   wire:        you describe the graph, wire GENERATES the wiring CODE            │
│                  (compile-time — a build step, output is normal Go)                 │
│                                                                                          │
│   fx:           you describe the graph, fx BUILDS it at RUNTIME via                       │
│                   reflection (no generated code, but errors in the graph                     │
│                   surface at STARTUP, not at compile time)                                      │
│                                                                                                     │
│   For small-to-medium services, manual DI (this module's default                                     │
│   throughout) is genuinely often the simplest, most debuggable choice —                                 │
│   reach for wire or fx once wiring code becomes a real maintenance burden                                  │
│   across dozens of components.                                                                                │
└──────────────────────────────────────────────────────────┘
```

---

## Reference Project Structure

Rather than a numbered project (this module's topics are structural habits
more than a single algorithm to implement), **`production-service/`**
alongside this guide is a small, runnable skeleton applying every section
above at once: a layered `cmd/`/`internal/` structure, config loaded from
env vars with defaults, manual dependency injection wiring a domain type
through a repository and an HTTP layer, `log/slog` structured logging,
Prometheus metrics, liveness/readiness health checks, and graceful
shutdown on `SIGTERM`. See its own README for a full walkthrough with
diagrams.
