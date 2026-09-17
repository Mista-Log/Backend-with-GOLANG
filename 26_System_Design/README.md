# Go for Beginners — Module 26: System Design

## Contents

1. **[26-system-design.md](./26-system-design.md)** — Scalability, load
   balancing, caching strategies, Redis, CDNs, message queues (cross-
   referencing Module 20), database sharding and replication (cross-
   referencing Module 22), rate-limiting algorithm comparison, distributed
   caching, circuit breakers, retry, and backpressure.

2. **[resilience-toolkit/](./resilience-toolkit)** — Four small, reusable
   packages: a Circuit Breaker (full three-state machine), a token-bucket
   Rate Limiter, exponential-backoff Retry, and a bounded worker pool
   demonstrating real, structural Backpressure — all runnable with
   observable, timed output proving each behavior actually happens.

3. **[design-case-studies.md](./design-case-studies.md)** — Four full
   system design write-ups: **Design YouTube, Design Uber, Design
   WhatsApp, Design Stripe** — each with requirements, a high-level
   architecture diagram, and a deep dive on the 2-3 hardest components,
   mapping every piece to the specific earlier module or project
   (Module 12's worker pools, Module 20's choreography, Module 21's
   fintech-backend, Module 25's Chat Server) that already implements it.

## Setup

```bash
cd resilience-toolkit
go run ./cmd/demo
```

No external dependencies.

*Note: this module is the synthesis point for Modules 12, 15, 16, 18,
20, 21, 22, and 25 — the case studies are meant to be read alongside
whichever of those you want to refresh, not as a replacement for them.*
