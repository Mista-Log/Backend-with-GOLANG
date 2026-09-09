# Distributed Patterns — Reference Implementations

Seven focused, runnable implementations of the guide's most concrete
patterns — each built so the behavior being taught is **directly
observable** in the demo's output, not just asserted in a comment.

## Setup

```bash
cd distributed-patterns
go run ./cmd/demo
```

No external dependencies — pure standard library.

---

## 1. Event Sourcing

```
┌──────────────────────────────────────────────────────────┐
│   Append: AccountOpened → MoneyDeposited(500) → MoneyWithdrawn(120)   │
│        │                                                                │
│        ▼                                                                  │
│   Replay folds them: 0 + 500 - 120 = 380                                   │
│                                                                                │
│   A SECOND Append attempt with a STALE expectedVersion is REJECTED —            │
│   optimistic concurrency control, the same instinct as a database's                │
│   optimistic locking (Module 17) applied to an event log instead of a               │
│   row version column.                                                                  │
└──────────────────────────────────────────────────────────┘
```

## 2. CQRS

The demo makes the read model's staleness impossible to miss: it checks
the read model **before** the projector has ever run (not found at all),
**after** a `CatchUp()` (correct), then makes another write and checks
**again before the next CatchUp** (stale — still showing the old value).
This is CQRS's eventual-consistency trade-off, made visible rather than
theoretical.

## 3-4. Saga Pattern

```
┌──────────────────────────────────────────────────────────┐
│   Happy path:        place order → charge → reserve stock  → ALL succeed  │
│                                                                                │
│   Failure path:       place order → charge → reserve stock (FAILS:               │
│                                                    not enough stock)                 │
│                          │                                                            │
│                          ▼                                                              │
│                     compensate charge (refund) → compensate order (cancel)                │
│                          │                                                                   │
│                          ▼                                                                     │
│                     order ends up CANCELLED, payment ends up REFUNDED —                           │
│                     exactly as if the whole thing had never been attempted                           │
└──────────────────────────────────────────────────────────┘
```

The demo prints every step and compensation as it runs, in order — watch
the failure path's compensations print in the **reverse** order the
original steps completed in.

## 5. Outbox Pattern

The demo deliberately triggers the "crashed after publishing, before
marking sent" scenario via `SimulateCrashBeforeMarkingSent`, then calls
`Poll()` again — you'll see the same event delivered **twice** in the
output. That's not a bug in the demo; it's the guide's Outbox section's
core point made concrete: at-least-once delivery is the honest guarantee,
and a real subscriber needs the same idempotent handling Module 21's
webhook receiver already implemented.

## 6. Distributed Locks

```
┌──────────────────────────────────────────────────────────┐
│   worker-A: TryLock  → true   (lease acquired, TTL 200ms)               │
│   worker-B: TryLock  → false  (A's lease still valid)                       │
│                                                                                  │
│   ... 250ms pass, A's lease expires ...                                            │
│                                                                                        │
│   worker-B: TryLock  → true   (A's lease expired — B acquires it)                       │
│   worker-A: Unlock   → false  (A no longer holds it — B does now;                          │
│                                  Unlock correctly refuses to release                           │
│                                  a lock it doesn't actually hold)                                 │
└──────────────────────────────────────────────────────────┘
```

## 7. Leader Election

Two nodes race to become leader; the demo doesn't hardcode which one wins
(it reads `CurrentLeader()` to find out), then **stops whichever one
actually won**, simulating a crash. Watch the real ~150-190ms gap between
the stop and the survivor's failover message — that gap **is** the lease
TTL trade-off from the guide, made visible as real elapsed time instead of
an abstract description.

## 8. Replication

```
┌──────────────────────────────────────────────────────────┐
│   WriteSync("entry-1")   → takes ~150ms (waits for the SLOW follower)      │
│   slowFollower.Read() immediately after → [entry-1]  (already there)          │
│                                                                                    │
│   WriteAsync("entry-2")  → returns almost instantly                                  │
│   slowFollower.Read() IMMEDIATELY after → [entry-1]  (entry-2 MISSING —                 │
│                                              a genuine stale read, not simulated,          │
│                                              because the follower's goroutine                │
│                                              really hasn't finished its 150ms                  │
│                                              simulated network delay yet)                         │
│   slowFollower.Read() 200ms later        → [entry-1, entry-2]  (caught up)                          │
└──────────────────────────────────────────────────────────┘
```

This is the single clearest demonstration in the whole project of what
"eventual consistency" actually *feels like* from calling code: the same
`Read()` call, on the same follower, genuinely returns different answers
depending purely on *when* you happen to call it relative to an async write.

---

## Try It Yourself

- Add a `Snapshot` type to `eventsourcing` that caches a folded
  `AccountState` at a given version, and change `Replay` to start from the
  nearest snapshot instead of the beginning — then time replaying an
  account with 10,000 events with and without snapshotting
- Change the saga's failure demo so the THIRD step succeeds but a
  hypothetical fourth step fails — confirm all three prior steps'
  compensations run, in reverse, not just the two adjacent to the failure
- Make `distributedlock.LockStore` and `leaderelection.LeaseStore` share
  literally the same underlying type (they're structurally identical) —
  a good exercise in recognizing when two "different" patterns are really
  the same mechanism wearing different vocabulary, exactly as the guide
  calls out
- Add a THIRD follower to the replication demo with latency in between the
  other two, and change `WriteSync` to only wait for a **quorum** (majority)
  of followers instead of all of them — a small, concrete taste of what
  Raft/Paxos-style consensus systems actually optimize for
