# Project 1 — Banking API (database/sql + SQLite)

Every SQL statement, transaction boundary, and isolation-level choice in
this project is fully explicit — no ORM generating queries for you, so
exactly what runs against the database is exactly what's written in the code.

## Setup

```bash
cd banking-api
go mod tidy    # fetches the pure-Go SQLite driver — needs internet access
go run .
```

This creates `bank.db` (a plain SQLite file) in the current directory —
delete it any time to start completely fresh (`main.go` does this
automatically on every run, for a repeatable demo).

## What Happens When You Run It

```
┌──────────────────────────────────────────────────────────┐
│   1. openDB("bank.db")                                              │
│         → connects, configures the pool (MaxOpenConns=1 — see            │
│            the case study below), runs migrations/0001_*.sql               │
│                                                                                 │
│   2. Create alice ($500) and bob ($100)                                          │
│                                                                                       │
│   3. Transfer $150 from alice to bob → SUCCESS                                          │
│         alice: $350, bob: $250                                                             │
│                                                                                                │
│   4. Attempt to transfer $10,000 from bob → FAILS (insufficient funds)                          │
│         bob's balance is checked BEFORE and AFTER the failed attempt —                              │
│         proves the rollback actually left it untouched                                                 │
│                                                                                                             │
│   5. Attempt to force a negative balance with a raw UPDATE, bypassing                                        │
│      the application logic entirely → REJECTED by the CHECK constraint                                        │
│                                                                                                                    │
│   6. Ten CONCURRENT $10 transfers from a new account (carol, $1000)                                                  │
│         → all ten succeed, none lost, none corrupted                                                                    │
│                                                                                                                              │
│   7. Print the full transfers audit log                                                                                        │
└──────────────────────────────────────────────────────────┘
```

## Transaction Anatomy

```
┌──────────────────────────────────────────────────────────┐
│   Transfer(ctx, db, aliceID, bobID, 150)                                │
│                                                                              │
│   tx, _ := db.BeginTx(ctx, Serializable)                                       │
│   defer tx.Rollback()   ◀── ALWAYS registered immediately — becomes a            │
│                              no-op if Commit() succeeds below                       │
│        │                                                                              │
│        ▼                                                                                │
│   SELECT balance FROM accounts WHERE id = aliceID                                          │
│        │  → 500                                                                              │
│        ▼                                                                                        │
│   500 >= 150?  YES → continue          (if NO: return an error here —                              │
│        │                                  deferred Rollback() undoes NOTHING,                          │
│        │                                  since nothing was written yet)                                  │
│        ▼                                                                                                     │
│   UPDATE accounts SET balance = balance - 150 WHERE id = aliceID                                                │
│        ▼                                                                                                           │
│   UPDATE accounts SET balance = balance + 150 WHERE id = bobID                                                        │
│        ▼                                                                                                                 │
│   INSERT INTO transfers (from_id, to_id, amount) VALUES (aliceID, bobID, 150)                                            │
│        ▼                                                                                                                     │
│   tx.Commit()   ◀── ALL THREE writes become permanent, together, atomically                                                    │
└──────────────────────────────────────────────────────────┘
```

If **any** step after `BeginTx` returns an error, the function returns
immediately — and the deferred `tx.Rollback()` undoes every write the
transaction had made up to that point, leaving the database exactly as it
was before `Transfer` was ever called.

## Case Study: Why `MaxOpenConns(1)` for SQLite

```go
db.SetMaxOpenConns(1)
db.SetMaxIdleConns(1)
```

This looks like it would badly hurt concurrency — and for Postgres or
MySQL, capping the pool at 1 connection genuinely would be a serious
mistake. But SQLite is fundamentally different: the entire database is
**one file**, and SQLite itself only allows **one writer at a time** at the
file level, no matter how many separate connections your program opens.
Without this cap, ten concurrent `Transfer` calls (like the demo's step 6)
would each grab their own connection, all try to write to the same file
simultaneously, and several would fail outright with a
`"database is locked"` error — not a graceful queue, an actual error each
caller would have to handle.

```
┌──────────────────────────────────────────────────────────┐
│   WITHOUT the cap:  10 goroutines ──▶ 10 separate connections               │
│                        ──▶ several hit SQLite's single-writer limit           │
│                        ──▶ "database is locked" errors, transfers FAIL          │
│                                                                                     │
│   WITH MaxOpenConns(1):  10 goroutines ──▶ all share ONE connection               │
│                             ──▶ Go's connection pool ITSELF queues them,             │
│                                  running each transaction to completion                 │
│                                  before starting the next                                  │
│                             ──▶ every transfer eventually succeeds, safely,                  │
│                                  just serialized rather than truly parallel                      │
└──────────────────────────────────────────────────────────┘
```

This is exactly Module 12's semaphore pattern (a buffered channel limiting
concurrency to what a resource can actually handle) — except here it's
`sql.DB`'s own pool doing the limiting, tuned to match what the *specific
database engine* underneath can actually support safely. A Postgres-backed
version of this same project would instead set `MaxOpenConns` much higher
(often 25+), since Postgres is built to handle many genuine concurrent
writers itself.

## Try It Yourself
- Change `Transfer`'s isolation level from `sql.LevelSerializable` to
  `sql.LevelReadCommitted` and rerun the concurrent-transfers demo — with
  SQLite's single-writer behavior already serializing everything via the
  connection pool, do you expect (and do you actually observe) any
  difference here? Think through why the connection pool cap might make
  this particular isolation choice less consequential than it would be
  against a database that truly allows concurrent writers
- Add a `migrations/0002_add_account_type.up.sql` that adds a
  `type TEXT NOT NULL DEFAULT 'checking'` column, and confirm `runMigrations`
  picks it up and applies it on the next run without re-running `0001`
- Add a `GetAccountHistory(db, accountID) ([]TransferRecord, error)` that
  filters `ListTransfers`'s query to `WHERE from_id = ? OR to_id = ?` —
  this is exactly the query the `idx_transfers_from`/`idx_transfers_to`
  indexes exist to speed up; confirm with
  `EXPLAIN QUERY PLAN SELECT ... WHERE from_id = ?` that SQLite actually
  uses the index
