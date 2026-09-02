# Go for Beginners — Module 19: Production Go

## Contents

1. **[19-production-go.md](./19-production-go.md)** — Standard Go project
   layout (`cmd/`/`internal/`/`pkg/`), Clean Architecture and Hexagonal
   Architecture (functionally the same inward-dependency rule, different
   vocabulary), DDD concepts that show up in everyday Go (entities, value
   objects, aggregates, bounded contexts), configuration precedence,
   environment variables and the twelve-factor methodology, dependency
   injection (manual, and what wire/fx actually add), logging levels,
   the three pillars of observability, Prometheus metrics types, the
   liveness-vs-readiness health check distinction, graceful shutdown, and
   secrets handling — then a **Libraries** section covering zap, logrus,
   viper, cobra, wire, and fx with real code and a clear "when to reach for
   which" comparison. Diagrams throughout every section.

2. **[production-service/](./production-service)** — Not a numbered
   exercise but a **reference architecture**: a small, runnable skeleton
   applying every guide topic at once — a layered `cmd/`/`internal/`
   structure with the dependency direction verifiable by grep, env-var
   configuration, manual dependency injection, `log/slog` structured
   logging, real Prometheus metrics on a separate port, liveness/readiness
   health checks, and graceful shutdown on `SIGTERM`/`SIGINT` you can
   trigger and watch complete correctly. Its README traces one request
   through every layer, diagrams the shutdown sequence step by step, and
   shows exactly how the architecture makes the service testable without a
   server or database in the loop.


## Suggested Order

```
Production Go guide ──▶ production-service (reference architecture)
```

## Suggested Order

```
Production Go guide ──▶ production-service (reference architecture)
```


This module has no numbered projects — its topics are structural habits,
not algorithms to implement from scratch. The reference project exists to
make those habits concrete and copyable, not to be "completed" like earlier
modules' exercises.

## Setup

```bash
cd production-service
go mod tidy    # fetches prometheus/client_golang — needs internet access
go run ./cmd/server
```

*Note: this module builds on nearly everything before it — Module 06
(interfaces, the ports/adapters foundation), Module 09 (project structure,
`internal/`), Module 12 (graceful shutdown's signal handling), Module 13
(context deadlines, applied to shutdown instead of a single request),
Module 14 (why this architecture is testable), Module 15 (HTTP,
middleware), and Module 17 (the repository pattern this module's domain
layer extends). If any single piece feels unfamiliar, it's worth a quick
revisit to that module rather than pushing through.*
