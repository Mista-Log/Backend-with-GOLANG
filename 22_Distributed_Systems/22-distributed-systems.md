# 22. Distributed Systems

Module 20 built multiple services; Module 21 built one system spanning
many of them. This module is about the hard problems that only show up
once state is **spread across machines that can independently fail**: what
guarantees you can actually keep, and the specific patterns real systems
use to keep them anyway.

---

## CAP Theorem

For any distributed data system, when a **network partition** happens
(some nodes can't reach others), you must choose between **Consistency**
(every read sees the latest write) and **Availability** (every request
gets a response) — you cannot have both during the partition.

```
┌──────────────────────────────────────────────────────────┐
│                                                                  │
│         Consistency          Availability          Partition        │
│                                                        Tolerance         │
│              \                    |                    /                  │
│               \                   |                   /                     │
│                \                  |                  /                        │
│                 \                 |                 /                           │
│                  \                |                /                              │
│                   PICK ANY TWO... except partition tolerance                         │
│                   ISN'T OPTIONAL in a real network.                                     │
│                                                                                             │
│   A real network WILL partition eventually (a switch fails, a cable is                        │
│   cut, a data center loses connectivity) — so the real-world choice is                           │
│   actually just C vs. A, DURING a partition:                                                        │
│                                                                                                          │
│   CP system:  refuses to answer (or answers with an error) rather than                                    │
│                 risk returning stale/inconsistent data                                                       │
│                                                                                                                   │
│   AP system:  keeps answering — possibly with data that's slightly                                                 │
│                 out of date, favoring "always respond" over "always                                                   │
│                 correct"                                                                                                 │
└──────────────────────────────────────────────────────────┘
```

```
┌────────────────────────────────────────────────────┐
│   CP examples:  most traditional RDBMS replication setups,             │
│                   etcd, ZooKeeper, Google Spanner                          │
│                                                                                │
│   AP examples:   DNS, Cassandra (in its default config),                        │
│                    most CDNs                                                        │
└────────────────────────────────────────────────────┘
```

**Outside of an actual partition, this trade-off doesn't apply** — CAP is
specifically about behavior *during* a network partition; a healthy,
fully-connected system can (and should) aim for both.

---

## Consistency

Beyond the CAP theorem's binary framing, "consistency" comes in useful
shades — a real system usually chooses a specific point on this spectrum
deliberately, not just "strong" or "eventual":

```
┌──────────────────────────────────────────────────────────┐
│   STRONG consistency:      every read sees the most recent            │
│                               committed write, everywhere, always            │
│                               (expensive: usually needs coordination            │
│                               across nodes on every read or write)                 │
│                                                                                        │
│   EVENTUAL consistency:      reads MIGHT see stale data briefly, but                    │
│                                  every replica converges to the same                       │
│                                  value EVENTUALLY, once writes stop                            │
│                                  arriving                                                        │
│                                                                                                      │
│   READ-YOUR-WRITES:            a WEAKER guarantee than strong, but                                    │
│                                    stronger than pure eventual: a client                                 │
│                                    is guaranteed to see ITS OWN writes                                      │
│                                    immediately, even if other clients                                         │
│                                    might briefly see stale data                                                  │
└──────────────────────────────────────────────────────────┘
```

Module 21's ledger used **strong consistency** deliberately (one Mutex,
one source of truth) — genuinely correct for money, where "eventually
correct" isn't good enough. A social media "like count," by contrast, is a
perfectly reasonable place to accept eventual consistency: briefly showing
"127 likes" when it's really 128 costs nothing.

---

## Replication

Keeping copies of the same data on multiple nodes — for fault tolerance
(one node dying doesn't lose data) and for scaling reads (many nodes can
serve read traffic simultaneously).

```
┌──────────────────────────────────────────────────────────┐
│                     Leader-Follower Replication                     │
│                                                                          │
│              writes                    reads                              │
│                │                          │                                  │
│                ▼                          ▼                                    │
│           ┌─────────┐              ┌───────────┐                                 │
│           │  LEADER   │────────────▶│ FOLLOWER 1  │                                 │
│           └─────────┘   replicate   └───────────┘                                 │
│                │         ▲                                                          │
│                │         │  replicate                                                 │
│                └─────────┼──────────▶┌───────────┐                                      │
│                          │           │ FOLLOWER 2  │                                      │
│                                      └───────────┘                                      │
│                                                                                              │
│   SYNCHRONOUS replication:   the leader waits for follower(s) to                              │
│                                 confirm before acknowledging the write —                          │
│                                 strong consistency, but slower, and a                                │
│                                 slow/dead follower can block writes entirely                          │
│                                                                                                            │
│   ASYNCHRONOUS replication:    the leader acknowledges immediately,                                          │
│                                   replicates in the background — fast,                                          │
│                                   available, but a follower can lag                                                │
│                                   behind (and a reader hitting that                                                   │
│                                   follower sees STALE data — this is                                                     │
│                                   exactly "eventual consistency" from the                                                  │
│                                   section above, made concrete)                                                              │
└──────────────────────────────────────────────────────────┘
```

---

## Partitioning

Splitting one large dataset **across** multiple nodes (as opposed to
replication, which copies the SAME data to multiple nodes) — necessary
once a dataset is too large, or gets too much write traffic, for any
single node to handle alone.

```
┌──────────────────────────────────────────────────────────┐
│   HASH-based partitioning:                                          │
│      partition = hash(key) % numPartitions                              │
│      → spreads keys EVENLY across partitions, but makes RANGE               │
│         queries ("give me all users signed up in March") expensive,            │
│         since matching rows could be on any partition                             │
│                                                                                        │
│   RANGE-based partitioning:                                                              │
│      partition determined by a key range (e.g. usernames A-M on                             │
│      partition 1, N-Z on partition 2)                                                          │
│      → makes range queries FAST (contiguous data lives together), but                             │
│         risks "hot spots" — uneven load if one range is far more                                     │
│         active than others (e.g. alphabetically-early usernames                                         │
│         being disproportionately old, high-activity accounts)                                              │
└──────────────────────────────────────────────────────────┘
```

Most real distributed databases (Cassandra, DynamoDB, CockroachDB) let you
choose or combine both strategies depending on your access patterns —
there's no universally "correct" choice, only the right trade-off for a
given workload's actual read/write patterns.

---
## Consensus

The problem of getting multiple nodes to **agree on a single value**
(who's the leader, what the next entry in a log is) even when some nodes
crash, are slow, or the network drops messages between them —
provably impossible to solve perfectly in an asynchronous network with
even one faulty node (a famous result called FLP impossibility), which is
exactly why real consensus algorithms make careful, deliberate trade-offs
instead of chasing a perfect solution.

```
┌──────────────────────────────────────────────────────────┐
│   Raft and Paxos are the two dominant real-world consensus                │
│   algorithms — Raft was specifically designed to be more                     │
│   UNDERSTANDABLE than Paxos while providing equivalent guarantees,               │
│   which is why most modern systems (etcd, Consul, CockroachDB) use it.               │
│                                                                                          │
│   The core mechanism all of them share: a majority (QUORUM) of nodes                       │
│   must agree before a value is considered committed — this is EXACTLY                          │
│   why these systems are typically deployed with an ODD number of nodes                            │
│   (3, 5, 7): a 5-node cluster tolerates 2 failures while still having a                              │
│   majority (3) able to agree.                                                                          │
└──────────────────────────────────────────────────────────┘
```

You'll almost never implement consensus from scratch — you'll use etcd,
Consul, or a database that implements it internally (CockroachDB, Google
Spanner) — but recognizing "this specific problem needs consensus" is the
valuable skill: anywhere multiple nodes need to agree on exactly one
"true" value despite failures.

---

## Leader Election

A specific, extremely common application of consensus: choosing **one**
node among several to act as the leader/coordinator for some duty (issuing
new IDs, running a scheduled job exactly once, accepting all writes in a
leader-follower setup).

```go
type Lease struct {
	HolderID  string
	ExpiresAt time.Time
}

// A node repeatedly tries to acquire or RENEW the lease — this is the
// same mechanism a real system (etcd, using its own consensus underneath)
// exposes to application code, simplified to its essential shape.
func TryAcquireOrRenew(store *LeaseStore, nodeID string, ttl time.Duration) bool {
	current := store.Get()
	if current == nil || time.Now().After(current.ExpiresAt) || current.HolderID == nodeID {
		store.Set(&Lease{HolderID: nodeID, ExpiresAt: time.Now().Add(ttl)})
		return true
	}
	return false // someone else holds a still-valid lease
}
```

```
┌──────────────────────────────────────────────────────────┐
│   Node A: acquires the lease, becomes leader, RENEWS it periodically      │
│           (well before it expires)                                            │
│                                                                                    │
│   Node A crashes  ──▶  stops renewing                                               │
│                                                                                          │
│   Lease EXPIRES (no renewal arrived in time)                                              │
│                                                                                                │
│   Node B: notices the lease is expired, ACQUIRES it, becomes the new                             │
│           leader                                                                                    │
│                                                                                                          │
│   This is exactly how Kubernetes controllers, distributed cron                                            │
│   schedulers, and database leader-follower setups decide who's in                                            │
│   charge, and detect + recover from a leader failing, automatically.                                            │
└──────────────────────────────────────────────────────────┘
```

**The lease's TTL is a real trade-off**: too short, and normal network
jitter causes unnecessary leadership churn (a node briefly fails to renew
in time and gets replaced even though it's fine); too long, and a genuine
failure takes longer to notice and recover from.

---

## Event Sourcing

Instead of storing an entity's **current state** directly (an `Account`
row with a `balance` column you `UPDATE`), store the full sequence of
**events** that led to that state — and derive the current state by
replaying them. This should feel familiar: it's Module 21's ledger
principle (never mutate history, only append), generalized beyond money to
*any* kind of entity.

```
┌──────────────────────────────────────────────────────────┐
│   Traditional (state-based):                                          │
│      accounts table:  { id: 1, balance: 70 }                               │
│      UPDATE accounts SET balance = 70 WHERE id = 1                            │
│      (the events that PRODUCED 70 are gone — overwritten)                        │
│                                                                                       │
│   Event-sourced:                                                                        │
│      events table: [                                                                       │
│        { type: "AccountOpened",   amount: 0   },                                              │
│        { type: "MoneyDeposited",  amount: 100 },                                                 │
│        { type: "MoneyWithdrawn",  amount: 30  },                                                    │
│      ]                                                                                                  │
│      current balance = REPLAY every event, folding them into state:                                        │
│        0 + 100 - 30 = 70                                                                                     │
│                                                                                                                   │
│   The FULL HISTORY is preserved, permanently — "what was the balance                                                │
│   at 3pm last Tuesday?" is answerable by replaying events UP TO that                                                   │
│   point, something the traditional approach can never answer once the                                                    │
│   row has been overwritten.                                                                                                  │
└──────────────────────────────────────────────────────────┘
```

```go
type Event interface{ isEvent() }
type AccountOpened struct{}
type MoneyDeposited struct{ Amount int64 }
type MoneyWithdrawn struct{ Amount int64 }

func (AccountOpened) isEvent()   {}
func (MoneyDeposited) isEvent()  {}
func (MoneyWithdrawn) isEvent()  {}

// Rebuilding state is a FOLD (Module 03's Reduce) over the event history:
func Replay(events []Event) (balance int64) {
	for _, e := range events {
		switch ev := e.(type) {
		case MoneyDeposited:
			balance += ev.Amount
		case MoneyWithdrawn:
			balance -= ev.Amount
		}
	}
	return balance
}
```

**Snapshotting** is the practical answer to "replaying millions of events
every time is slow": periodically save the folded state at a point in
time, so future replays only need to start from the last snapshot forward,
not from the very beginning.

---

## CQRS

**Command Query Responsibility Segregation** — using a completely
different model (even a different data store) for **writes** (commands:
"deposit $50") than for **reads** (queries: "show me this account's
balance"). This pairs naturally with event sourcing: events are the
perfect *write* model (an append-only log), while a separately-maintained,
denormalized **read model** (a plain table, updated by a background
process consuming those events) is optimized purely for fast queries.

```
┌──────────────────────────────────────────────────────────┐
│                                                                  │
│   WRITE side                                    READ side              │
│                                                                     │
│   Command: "deposit $50"                    Query: "get balance"        │
│        │                                              │                    │
│        ▼                                              ▼                      │
│   append MoneyDeposited event          read from a denormalized,               │
│        │                                 pre-computed "account_balances"          │
│        ▼                                 table — NO replay needed at                 │
│   [event stream]                          query time at all                             │
│        │                                              ▲                                    │
│        └──────── a PROJECTOR consumes ─────────────────┘                                      │
│                   events and updates                                                              │
│                   the read model                                                                     │
│                                                                                                          │
│   The write model optimizes for CORRECTNESS and a complete history.                                        │
│   The read model optimizes for QUERY SPEED — they can even be                                                 │
│   different DATABASE TECHNOLOGIES entirely (events in a durable log                                              │
│   like Kafka, read model in a fast key-value store or search index).                                              │
└──────────────────────────────────────────────────────────┘
```

The trade-off: the read model is **eventually consistent** with the write
model (there's a small lag between an event being appended and the
projector updating the read table) — CQRS is a deliberate, informed
application of the Consistency section's spectrum, not an accident.

---
## Saga Pattern

Module 17's database transactions guarantee atomicity **within one
database**. Once a "transaction" spans multiple services (Module 20) —
each with its own database — there is no single `COMMIT` that can cover
all of them. A **saga** is the standard answer: a sequence of local
transactions, each with a matching **compensating action** that undoes it
if a *later* step fails.

```
┌──────────────────────────────────────────────────────────┐
│           Order Saga: place order → charge payment → reserve stock       │
│                                                                              │
│   Step 1: Order Service creates the order (local transaction)                  │
│   Step 2: Payment Service charges the customer (local transaction)                │
│   Step 3: Inventory Service reserves stock (local transaction)                       │
│                                                                                            │
│   If Step 3 FAILS (out of stock):                                                            │
│      run Step 2's COMPENSATION: refund the payment                                              │
│      run Step 1's COMPENSATION: cancel the order                                                   │
│                                                                                                         │
│   The system never has an ATOMIC cross-service transaction — instead,                                    │
│   it guarantees that EVERY step either fully completes, or is fully                                          │
│   undone by its own compensating action, ending in a consistent state                                           │
│   either way.                                                                                                      │
└──────────────────────────────────────────────────────────┘
```

```go
type Step struct {
	Do      func(ctx context.Context) error
	Undo    func(ctx context.Context) error
}

func RunSaga(ctx context.Context, steps []Step) error {
	completed := []Step{}
	for _, step := range steps {
		if err := step.Do(ctx); err != nil {
			// unwind everything that DID succeed, in REVERSE order —
			// the same LIFO shape as defer (Module 02) and a call stack
			for i := len(completed) - 1; i >= 0; i-- {
				completed[i].Undo(ctx)
			}
			return fmt.Errorf("saga failed at step, rolled back: %w", err)
		}
		completed = append(completed, step)
	}
	return nil
}
```

**Choreography vs. orchestration** (Module 20) applies directly here too:
an **orchestrated** saga (shown above) has one coordinator explicitly
calling each step and compensation in order; a **choreographed** saga
instead has each service listening for the previous step's event and
publishing its own completion/failure event, with no central coordinator
— the same trade-off Module 20 covered, now specifically applied to
multi-step transactions instead of general event reactions.

---

## Outbox Pattern

A specific, extremely common bug this pattern exists to prevent: a service
needs to **both** update its own database **and** publish an event about
that change — but those are two *separate* operations against two
*separate* systems (your database, and a message broker), and a crash
between them leaves you with either a database change nobody heard about,
or a published event describing a change that never actually happened.

```
┌──────────────────────────────────────────────────────────┐
│              The DUAL-WRITE PROBLEM (broken)                        │
│                                                                          │
│   1. UPDATE orders SET status = 'placed' WHERE id = 42   ✅ succeeds       │
│   2. [ CRASH right here ]                                                     │
│   3. publish("order.placed", ...)                          ❌ never runs      │
│                                                                                    │
│   The order IS placed, but Payment Service never hears about it —                    │
│   it's now silently stuck forever.                                                      │
│                                                                                              │
│              The OUTBOX PATTERN (fixed)                                                        │
│                                                                                                    │
│   1. IN ONE DATABASE TRANSACTION:                                                                    │
│         UPDATE orders SET status = 'placed' WHERE id = 42                                               │
│         INSERT INTO outbox (event_type, payload) VALUES ('order.placed', ...)                              │
│      → both succeed together, or NEITHER does (Module 17's transactions)                                      │
│                                                                                                                    │
│   2. A SEPARATE relay process polls the outbox table, publishes each                                                │
│      unsent row to the real message broker, marks it sent                                                              │
│         → if the relay crashes AFTER publishing but BEFORE marking sent,                                                  │
│           it just republishes on restart — a DUPLICATE delivery, which                                                       │
│           is exactly Module 20's at-least-once guarantee, requiring the                                                          │
│           same idempotent-consumer handling Module 21's webhook receiver                                                            │
│           already implements                                                                                                            │
└──────────────────────────────────────────────────────────┘
```

The core insight: the event's *existence* is now guaranteed by the
**same** database transaction as the business change it describes — the
only thing that can still fail is the *delivery* of an already-guaranteed
event, which is a solved problem (retry until acknowledged).

---

## Distributed Locks

A `sync.Mutex` (Module 12) only coordinates goroutines **within one
process**. Once multiple separate processes/services need mutually
exclusive access to something (a shared resource, "only one worker should
process this batch job right now"), the lock itself has to live somewhere
all of them can reach — typically Redis, etcd, or a database row.

```go
// A Redis-backed lock — the classic pattern is a single atomic
// SET-if-not-exists with an expiry, so a crashed holder's lock still
// eventually releases on its own:
func TryLock(rdb *redis.Client, key string, holderID string, ttl time.Duration) (bool, error) {
	return rdb.SetNX(ctx, key, holderID, ttl).Result() // atomic: only succeeds if the key didn't already exist
}

func Unlock(rdb *redis.Client, key string, holderID string) error {
	// only delete if WE still hold it — a Lua script makes this check-and-
	// delete atomic, preventing accidentally releasing someone ELSE's lock
	// (e.g., if our own lock already expired and someone else acquired it)
	script := `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`
	return rdb.Eval(ctx, script, []string{key}, holderID).Err()
}
```

```
┌──────────────────────────────────────────────────────────┐
│   ALWAYS set a TTL/expiry on a distributed lock — a lock held by         │
│   a process that crashed WITHOUT releasing it would otherwise block          │
│   everyone else FOREVER. A TTL bounds the damage: worst case, everyone           │
│   waits out the expiry, then someone new acquires it.                               │
│                                                                                          │
│   This is the EXACT same lease-based mechanism as Leader Election                          │
│   above — a distributed lock IS a lease, just scoped to "exclusive                            │
│   access to one resource" instead of "who's the leader."                                         │
└──────────────────────────────────────────────────────────┘
```

**Distributed locks are inherently a best-effort safety mechanism, not a
perfect guarantee** — clock drift, network delays, and garbage-collection
pauses can all cause a holder to believe it still holds a lock after it's
actually expired and been reacquired elsewhere. For anything where
correctness truly cannot be violated (Module 21's money movements), prefer
designs that don't *require* perfect mutual exclusion at all — idempotency
keys and the ledger's own balance invariant are more robust safety nets
than any lock alone.

---

Onto the project — a set of small, focused reference implementations for
the patterns concrete enough to code directly: Event Sourcing, CQRS
(built on top of it), the Saga Pattern (coordinating a multi-service order
flow with real compensations), the Outbox Pattern (with a real relay
process), a lease-based Distributed Lock, Leader Election (the same lease
mechanism, applied to choosing a leader instead), and a small simulated
Replication demo showing the sync-vs-async trade-off directly.
