# Go for Beginners — Module 17: Databases

## Contents

1. **[17-databases.md](./17-databases.md)** — SQL databases compared
   (Postgres, MySQL, SQLite) and `database/sql` fundamentals (connecting,
   `QueryRow`/`Query`/`Exec`, placeholders, and the SQL-injection rule that
   matters more than anything else in this module); GORM vs. sqlx (a full
   ORM vs. an ergonomic layer over real SQL — a common point of confusion,
   clarified directly); and an **Advanced** section covering transactions
   (with a before/after diagram of a transfer with and without one),
   indexes, constraints, isolation levels, migrations, connection pooling
   (tied directly back to Module 12's semaphore pattern), and the
   repository pattern (Module 06's interfaces + Module 14's mocking,
   applied to the data layer). Diagrams throughout.

2. **[banking-api/](./banking-api)** — Plain `database/sql` + SQLite, with
   every transaction boundary and isolation-level choice fully explicit.
   Real migrations (embedded `.sql` files, applied and tracked at startup),
   a `CHECK` constraint tested directly against a raw `UPDATE`, a genuine
   rollback-on-failure demonstration, and ten concurrent transfers proving
   no lost updates — plus a case study on why SQLite specifically needs
   `MaxOpenConns(1)`, unlike Postgres or MySQL.

3. **[inventory/](./inventory)** — `sqlx` + the repository pattern. The
   exact same business logic (`LowStockReport`, `Restock`) runs against
   *two* different `ProductRepository` implementations — a real
   SQLite-backed one and a pure in-memory fake — back to back, printing
   identical results, with a case study on the real risk that a fake can
   silently drift from the database's actual constraint behavior.

## Suggested Order

```
Databases guide ──▶ Banking API (database/sql) ──▶ Inventory (sqlx + repository pattern)
                       (explicit transactions,          (ergonomic SQL, interface-based
                        isolation, migrations)            data access, swappable backends)
```

Banking API keeps every database interaction fully visible and manual —
the right starting point for understanding what a transaction actually
does. Inventory then shows the ergonomic middle ground (sqlx) and the
architectural pattern (repositories) that makes a real data layer testable
without a database in the loop.

## Setup — Real Dependencies Again

Like Module 16, both projects here depend on external packages and need a
one-time, internet-connected setup step:

```bash
cd banking-api && go mod tidy && go run .
cd inventory && go mod tidy && go run .
```

Both use `modernc.org/sqlite` — a pure-Go SQLite driver requiring no C
compiler or system dependency, so once `go mod tidy` succeeds, everything
else works exactly like every earlier project. Both create a local `.db`
file (deleted and recreated fresh on every run for a repeatable demo).

*Note: this module builds on Modules 00–16 — start there first if you
haven't already, especially Module 06 (interfaces — the repository pattern
depends on them directly), Module 08 (validation — business-rule
enforcement above the database), Module 12 (concurrency — connection pools
are a semaphore in disguise), and Module 14 (mocking — the fake repository
here is that same idea, one layer deeper).*
