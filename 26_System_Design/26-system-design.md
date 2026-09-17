# 26. System Design

Modules 20-22 gave you the pieces (services, ledgers, event sourcing,
consensus). This module is about **assembling** them under real
constraints — traffic that outgrows one machine, failures that cascade if
you're not careful, and the vocabulary system design interviews and real
architecture reviews expect you to use fluently.

---

## Scalability

```
┌──────────────────────────────────────────────────────────┐
│   VERTICAL (scale UP):    bigger machine — simple, but a hard             │
│                              ceiling and a single point of failure             │
│                                                                                     │
│   HORIZONTAL (scale OUT):   more machines running the SAME code —                     │
│                                needs the app to be STATELESS (Module 15's                │
│                                servers never stored session state locally,                    │
│                                precisely to make this possible later)                             │
└──────────────────────────────────────────────────────────┘
```

## Load Balancing

```
┌──────────────────────────────────────────────────────────┐
│   Round robin:       simple, even, ignores actual server load         │
│   Least connections:   routes to whichever backend has the FEWEST         │
│                           active connections right now                        │
│   Consistent hashing:    same key ALWAYS routes to the same backend —            │
│                            crucial for caching, and adding/removing a               │
│                            backend only remaps a SMALL fraction of keys                │
└──────────────────────────────────────────────────────────┘
```

## Caching

Module 16's `QueryCache` was one layer of this; real systems cache at
several layers at once (browser -> CDN -> gateway -> application -> database).

```
┌────────────────────────────────────────────────────┐
│   CACHE-ASIDE:    check cache first; on a MISS, read the source,          │
│                     then populate the cache (Module 16's approach)             │
│   WRITE-THROUGH:    every write updates cache AND source together —              │
│                        always fresh, slower writes                                   │
│   WRITE-BEHIND:       writes hit the cache immediately, flushed to the                  │
│                          source ASYNCHRONOUSLY — fast, but needs Module                     │
│                          21/22's Outbox Pattern to avoid losing data on a crash                 │
└────────────────────────────────────────────────────┘
```

## Redis

```go
rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
rdb.Set(ctx, "user:42:profile", jsonBytes, 10*time.Minute)
val, err := rdb.Get(ctx, "user:42:profile").Result() // err == redis.Nil on a cache miss
```
Beyond simple key/value: Hashes (partial object updates), Sorted Sets
(leaderboards), and Streams (Module 20's Redis Streams — a durable,
consumer-group-aware log).

## CDN

```
┌──────────────────────────────────────────────────────────┐
│   User in Tokyo requests video.mp4                                    │
│   -> nearest EDGE server: cached? serve directly, origin never touched     │
│                            not cached? fetch from origin ONCE, cache it,        │
│                            serve every SUBSEQUENT nearby request from there        │
└──────────────────────────────────────────────────────────┘
```
Static-ish content (images, video, JS bundles) caches beautifully at the
edge; genuinely live data (a chat message) doesn't cache at all — which is
exactly why that still needs WebSockets (Module 25) or a queue instead.

## Message Queues

Covered in depth in Module 20; the system-design framing: without a queue,
Service A blocks on Service B's availability, and B being slow makes A
slow too (a cascading failure — see Circuit Breaker, below). With a queue,
A publishes and moves on; a traffic spike just makes the queue longer
instead of overwhelming B directly (Backpressure, also below).

## Database Sharding

The same idea as Module 22's Partitioning, under its common name when
applied specifically to a database — splitting one dataset horizontally
across multiple database instances. The real complexity tax: any query
joining or aggregating ACROSS shards can no longer be one simple query —
it must query every shard and merge in application code.

## Replication

Covered in depth in Module 22 (leader-follower, sync vs async, CAP). The
system-design point: **sharding and replication solve different problems
and are normally used together** — sharding spreads DIFFERENT data across
nodes (for scale); replication copies the SAME data across nodes (for
durability/availability). A real large system shards AND replicates each
shard independently.

## Rate Limiting

Module 16 built a token bucket; the fuller comparison:

```
┌────────────────────────────────────────────────────┐
│   Token bucket:    allows BURSTS up to bucket size, refills steadily          │
│   Leaky bucket:      processes at a FIXED steady rate, queuing (not               │
│                        rejecting) anything faster — smooths bursts into                │
│                        added latency instead                                              │
│   Fixed window:        simple, but allows ~2x the limit right at a window                    │
│                          boundary (a burst spanning the end of one window                       │
│                          and the start of the next)                                                │
│   Sliding window:         fixes the boundary problem, at the cost of                                  │
│                             tracking more state per client                                               │
└────────────────────────────────────────────────────┘
```

## Distributed Cache

A cache that's itself sharded/replicated across nodes (Redis Cluster is
the common real example), so the cache layer isn't its own single point
of failure once traffic outgrows one instance — the same consistent-
hashing mechanism from Load Balancing, now routing cache KEYS to nodes
instead of requests to servers.

## Circuit Breaker

Prevents cascading failures: if a downstream service is failing, stop
calling it for a while (fail fast) instead of piling on requests it can't
handle anyway.

```
┌──────────────────────────────────────────────────────────┐
│      CLOSED                 OPEN                  HALF-OPEN               │
│   (calls flow          (rejected instantly,    (ONE test call              │
│    normally)              no network call)         let through)               │
│      │  failures exceed      │  after a timeout      │  succeeds? -> CLOSED      │
│      │  threshold             │  elapses               │  fails?    -> OPEN again   │
│      ▼                        ▼                        ▼                        │
│    OPEN <───────────────────────────────────────── (loops back around)            │
└──────────────────────────────────────────────────────────┘
```

## Retry

Module 08 built this in full (exponential backoff, the `Permanent` escape
hatch). System-design point: retry and circuit breaker are complementary —
retry handles one transient blip; the breaker watches the AGGREGATE
failure rate and stops retrying altogether once it's clear the dependency
is actually down, not just having one bad request.

## Backpressure

Signaling upstream that a system can't keep up, so producers slow down
instead of overwhelming it. A bounded channel (`make(chan T, N)`, Module
12) already implements this: once full, a send blocks, forcing the
producer to slow to the consumer's real pace — structurally, no extra code.

```
┌──────────────────────────────────────────────────────────┐
│   Without it: A floods B faster than B can process -> B's queue          │
│                 grows unbounded -> B runs out of memory -> CRASH               │
│   With it:      B signals "I'm full" (bounded queue, HTTP 429, consumer          │
│                    lag) -> A slows down or sheds load BEFORE B fails                 │
└──────────────────────────────────────────────────────────┘
```

---

Onto the project — a small, genuinely reusable **resilience toolkit**
(Circuit Breaker, token-bucket Rate Limiter, Retry, and a bounded worker
pool demonstrating Backpressure) — followed by
**[design-case-studies.md](./design-case-studies.md)**: four full system
design write-ups (Design YouTube, Design Uber, Design WhatsApp, Design
Stripe), each mapping every component to the specific earlier module and
project that already builds it.
