# Go for Beginners — Module 22: Distributed Systems

## Contents

1. **[22-distributed-systems.md](./22-distributed-systems.md)** — CAP
   Theorem, consistency models, replication (sync vs. async), partitioning,
   consensus, leader election, event sourcing, CQRS, the Saga Pattern, the
   Outbox Pattern, and distributed locks. Diagrams throughout.

2. **[distributed-patterns/](./distributed-patterns)** — Seven runnable
   reference implementations, each built so the behavior is directly
   observable: Event Sourcing with optimistic concurrency, CQRS with a
   visibly-lagging read model, an orchestrated Saga with real compensations
   running in reverse on failure, the Outbox Pattern with a genuine
   duplicate delivery, Distributed Locks with real TTL expiry and
   failover, Leader Election with a real crash-and-recover cycle, and a
   Replication simulator where a stale read is a real timing effect, not a
   simulated one.

## Setup

```bash
cd distributed-patterns
go run ./cmd/demo
```

No external dependencies — pure standard library.

*Note: this module builds on Module 12 (concurrency — every lease/lock
uses the same Mutex-protected patterns), Module 17 (transactions — the
Outbox Pattern's atomicity guarantee), Module 20 (message queues — the
Outbox Relay and Saga's compensations), and Module 21 (the ledger's
append-only, replay-derived-balance principle, generalized here to Event
Sourcing).*
