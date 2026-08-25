# 17. Databases

Every project so far has stored data in memory — gone the moment the
program exits. This module connects Go to a real, persistent database:
`database/sql`'s standard interface, the two most common ways to work with
it above the raw driver level, and the production concerns (transactions,
indexes, migrations, connection pooling) that separate a toy data layer
from a real one.

---

## SQL: Postgres, MySQL, SQLite

Go's `database/sql` package defines a **driver-agnostic interface** —
the same `sql.DB`, `Query`, `Exec`, and `Scan` calls work against any
database with a compatible driver. You `import _ "driver/package"` purely
for its side effect of registering itself, then interact only through
`database/sql`'s own types.

```
┌──────────────────────────────────────────────────────────┐
│                                                                  │
│   Your Go code                                                     │
│        │  uses ONLY database/sql's types: sql.DB, sql.Rows, ...      │
│        ▼                                                                │
│   database/sql   (driver-agnostic interface)                              │
│        │                                                                    │
│        ▼                                                                      │
│   the REGISTERED driver translates calls into the specific                      │
│   wire protocol for whichever database you're actually talking to                  │
│        │                                                                              │
│        ├──▶ Postgres  (github.com/jackc/pgx or lib/pq)                                  │
│        ├──▶ MySQL      (github.com/go-sql-driver/mysql)                                   │
│        └──▶ SQLite      (modernc.org/sqlite — pure Go, no C compiler needed)                 │
│                                                                                                   │
│   Switching databases, in the IDEAL case, means changing the import                                │
│   and the connection string — your query code often stays identical,                                 │
│   though real SQL dialect differences (see below) can still bite.                                        │
└──────────────────────────────────────────────────────────┘
```

**Postgres** — the most fully-featured open-source option: strong
standards compliance, rich data types (JSON, arrays, full-text search),
and the default choice for most new Go backend projects today.

**MySQL** — extremely widely deployed, especially in older/existing
infrastructure and much of the PHP/web-hosting world; slightly different
SQL dialect quirks (e.g., `AUTO_INCREMENT` vs. Postgres's `SERIAL`).

**SQLite** — an embedded, file-based (or even in-memory) database with
**no separate server process at all** — the entire database is one file
your program reads and writes directly. Ideal for this module's projects
(zero setup, runs anywhere Go runs), local development, tests, CLI tools,
and genuinely fine for many small-to-medium production workloads too.

```go
import (
	"database/sql"
	_ "modernc.org/sqlite" // pure-Go SQLite driver — registers itself, unused directly
)

db, err := sql.Open("sqlite", "app.db") // "app.db" — just a file path
```

**`sql.Open` does NOT connect immediately** — it only validates arguments
and prepares a connection pool for later use. Always follow it with
`db.Ping()` to actually verify connectivity:
```go
if err := db.Ping(); err != nil {
	log.Fatal("cannot reach database:", err)
}
```

### Querying

```go
// QueryRow — expect exactly ONE row back
var name string
err := db.QueryRow("SELECT name FROM users WHERE id = ?", 42).Scan(&name)
if err == sql.ErrNoRows {
	// no matching row — NOT a real error, a normal "not found" outcome
}

// Query — expect MULTIPLE rows, iterate with Next()
rows, err := db.Query("SELECT id, name FROM users WHERE active = ?", true)
if err != nil {
	log.Fatal(err)
}
defer rows.Close() // ALWAYS — leaks the connection back to the pool otherwise

for rows.Next() {
	var id int
	var name string
	if err := rows.Scan(&id, &name); err != nil {
		log.Fatal(err)
	}
	fmt.Println(id, name)
}
if err := rows.Err(); err != nil { // check for errors that happened DURING iteration
	log.Fatal(err)
}
```

```
┌──────────────────────────────────────────────────────────┐
│   QueryRow(...)   →  expect 0 or 1 rows, .Scan() directly            │
│   Query(...)       →  expect 0+ rows, loop with rows.Next()             │
│                          + rows.Scan() per row, defer rows.Close()          │
│   Exec(...)         →  no rows returned (INSERT/UPDATE/DELETE);               │
│                          returns a Result with LastInsertId()/RowsAffected()      │
└──────────────────────────────────────────────────────────┘
```

**Always use placeholders, never string-concatenate values into SQL** —
this is the single most important rule in this entire module:
```go
// ✅ SAFE — the driver handles escaping; user input can NEVER alter the query's structure
db.Query("SELECT * FROM users WHERE name = ?", userInput)

// ❌ SQL INJECTION — a user input like `' OR '1'='1` breaks out of the
//     intended query entirely, potentially exposing or destroying data
db.Query("SELECT * FROM users WHERE name = '" + userInput + "'")
```
**Placeholder syntax differs by driver**: SQLite and MySQL use `?`;
Postgres uses numbered `$1`, `$2`, ... This is one of the real dialect
differences that means switching databases isn't always purely a config
change.

---
## ORM: GORM and sqlx

Both sit above raw `database/sql`, but at very different levels of
abstraction — a common beginner misconception is that they're
interchangeable alternatives; they're really solving different problems.

### GORM — a full ORM

GORM maps Go structs to tables, generates SQL for you, and handles
migrations, associations, and hooks — you write far less raw SQL.

```go
import (
	"gorm.io/gorm"
	"gorm.io/driver/sqlite"
)

type User struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string
	Email string `gorm:"uniqueIndex"`
}

db, _ := gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
db.AutoMigrate(&User{}) // creates/updates the table to match the struct

db.Create(&User{Name: "Ada", Email: "ada@example.com"})

var user User
db.First(&user, "email = ?", "ada@example.com")

db.Model(&user).Update("Name", "Ada Lovelace")
db.Delete(&user)
```

```
┌────────────────────────────────────────────────────┐
│   GORM:  you write Go structs + method calls              │
│            GORM GENERATES the SQL underneath                  │
│            (db.Create → INSERT ...; db.First → SELECT ...)       │
│                                                                       │
│   Trade-off: less SQL to write, but SQL generated for you can          │
│   be harder to predict/tune for complex queries, and "magic"             │
│   behavior (like AutoMigrate) needs to be understood, not just used         │
└────────────────────────────────────────────────────┘
```

### sqlx — NOT an ORM, an ergonomic layer over `database/sql`

sqlx keeps you writing **real SQL** yourself, but eliminates the tedious
manual `Scan(&a, &b, &c, ...)` boilerplate by scanning directly into
structs:

```go
import "github.com/jmoiron/sqlx"

type User struct {
	ID    int    `db:"id"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

db, _ := sqlx.Open("sqlite", "app.db")

var user User
db.Get(&user, "SELECT * FROM users WHERE id = ?", 42) // ONE row, scanned into the struct

var users []User
db.Select(&users, "SELECT * FROM users WHERE active = ?", true) // MANY rows, scanned into a slice

// Named parameters, bound from a struct's fields directly:
db.NamedExec("INSERT INTO users (name, email) VALUES (:name, :email)", user)
```

```
┌────────────────────────────────────────────────────┐
│   sqlx:  YOU write the SQL, exactly as you intend it              │
│            sqlx just eliminates repetitive Scan() boilerplate         │
│            (Get/Select map columns to struct fields via `db` tags)       │
│                                                                                │
│   Trade-off: more SQL to write than GORM, but the query that RUNS               │
│   is always exactly the query you WROTE — no generated-SQL surprises              │
└────────────────────────────────────────────────────┘
```

### Choosing

```
┌──────────────────────────────────────────────────────────┐
│   Want to move fast, don't mind an ORM generating your SQL,          │
│   want auto-migrations and association handling built in?               │
│      → GORM                                                                 │
│                                                                                  │
│   Want to write and fully control your own SQL, but skip manual                    │
│   Scan() boilerplate?                                                                  │
│      → sqlx                                                                              │
│                                                                                                │
│   Want zero extra dependencies, maximum control, don't mind writing                              │
│   Scan(&a, &b, &c) yourself?                                                                        │
│      → plain database/sql                                                                             │
└──────────────────────────────────────────────────────────┘
```

This module's two projects use **plain `database/sql`** (Banking API — so
every SQL statement, transaction boundary, and isolation-level choice is
fully explicit and visible) and **sqlx** (Inventory — showing the
ergonomic middle ground in practice).

---
## Advanced

### Transactions

A transaction groups multiple statements into **one atomic unit** — either
every statement succeeds and is committed, or (on any failure) all of them
are rolled back, leaving the database exactly as it was before.

```go
tx, err := db.Begin()
if err != nil {
	return err
}
defer tx.Rollback() // SAFE to call even after a successful Commit — becomes a no-op

if _, err := tx.Exec("UPDATE accounts SET balance = balance - ? WHERE id = ?", 100, fromID); err != nil {
	return err // deferred Rollback() undoes anything this transaction already did
}
if _, err := tx.Exec("UPDATE accounts SET balance = balance + ? WHERE id = ?", 100, toID); err != nil {
	return err
}

return tx.Commit() // ONLY NOW are both updates permanently applied, together
```

```
┌──────────────────────────────────────────────────────────┐
│              A money transfer WITHOUT a transaction                     │
│                                                                              │
│   UPDATE accounts SET balance = balance - 100 WHERE id = 1;   ✅ succeeds       │
│   [ ...program crashes, or the network drops, RIGHT HERE... ]                    │
│   UPDATE accounts SET balance = balance + 100 WHERE id = 2;   ❌ never runs        │
│                                                                                        │
│   RESULT: $100 has VANISHED — deducted from account 1, never credited                   │
│   to account 2. This is exactly the failure mode transactions exist to                     │
│   prevent.                                                                                    │
│                                                                                                    │
│              The SAME transfer, WRAPPED in a transaction                                            │
│                                                                                                          │
│   tx.Begin()                                                                                               │
│   tx.Exec(deduct from 1)     ✅                                                                               │
│   [ ...crash here... ]                                                                                          │
│   tx.Commit() NEVER CALLED  →  the deduction is AUTOMATICALLY ROLLED BACK                                          │
│                                  when the connection closes without a commit —                                        │
│                                  account 1's balance is UNCHANGED, as if                                                 │
│                                  nothing happened at all                                                                    │
└──────────────────────────────────────────────────────────┘
```

**`defer tx.Rollback()` immediately after a successful `Begin()`** is close
to a hard rule — mirroring Module 12's `defer mu.Unlock()` habit exactly.
Calling `Rollback()` on an already-committed transaction is documented as a
safe no-op, so this pattern correctly handles every exit path (success,
any error, even a panic) with one line.

---

### Indexes

An index is a separate, sorted data structure the database maintains
alongside a table, so it can find matching rows **without scanning every
row** — the difference between an O(n) full table scan and an O(log n)
lookup, exactly like the difference between a slice and a map (Module 04)
applied to disk-resident data.

```sql
CREATE INDEX idx_accounts_owner ON accounts(owner);
```

```
┌──────────────────────────────────────────────────────────┐
│   WITHOUT an index on `owner`:                                        │
│     SELECT * FROM accounts WHERE owner = 'ada'                            │
│     → scans EVERY row in the table, checking each one                        │
│                                                                                   │
│   WITH an index on `owner`:                                                        │
│     SELECT * FROM accounts WHERE owner = 'ada'                                        │
│     → looks up 'ada' directly in the index's sorted structure,                           │
│       jumps straight to matching rows                                                       │
│                                                                                                  │
│   Check what a query ACTUALLY does with:  EXPLAIN QUERY PLAN SELECT ...                            │
└──────────────────────────────────────────────────────────┘
```

**Indexes aren't free** — every `INSERT`/`UPDATE`/`DELETE` on an indexed
column also has to update the index, so indexing every column "just in
case" slows down writes for a benefit that only helps reads. Index columns
you actually filter/sort/join on frequently, not everything.

---

### Constraints

Constraints are rules the database **itself** enforces, rejecting any
write that would violate them — a second line of defense beneath your
application-level validation (Module 08).

```sql
CREATE TABLE accounts (
	id      INTEGER PRIMARY KEY,
	owner   TEXT NOT NULL,
	email   TEXT UNIQUE,
	balance REAL NOT NULL CHECK (balance >= 0),
	FOREIGN KEY (owner_id) REFERENCES owners(id)
);
```

```
┌────────────────────────────────────────────────────┐
│   PRIMARY KEY   →  uniquely identifies each row, auto-indexed         │
│   NOT NULL       →  this column can never be left empty                   │
│   UNIQUE          →  no two rows may share this value                       │
│   CHECK(...)       →  a custom boolean rule every row must satisfy             │
│   FOREIGN KEY       →  this value must exist as a row in another table           │
│                          (prevents an order referencing a customer that            │
│                           doesn't exist, for instance)                                │
└────────────────────────────────────────────────────┘
```

**Why enforce rules in the database when Go already validates them:**
application code can have bugs, get bypassed by a direct database
connection, or simply be one of several different services writing to the
same database — a `CHECK (balance >= 0)` constraint makes "balance goes
negative" **structurally impossible**, regardless of which code path
(or which future bug) attempted it.

---

### Isolation Levels

Isolation levels control what one transaction can **see** of another
transaction's *uncommitted* changes, while both run concurrently — directly
analogous to Module 12's concurrency-safety concerns, at the database
level.

```
┌──────────────────────────────────────────────────────────┐
│   READ UNCOMMITTED  →  can see OTHER transactions' uncommitted            │
│                          changes at all ("dirty reads") — rarely used,        │
│                          weakest guarantee                                       │
│                                                                                       │
│   READ COMMITTED     →  only ever sees changes that have been COMMITTED         │
│                          by other transactions — Postgres's default                 │
│                                                                                          │
│   REPEATABLE READ      →  the SAME query, run twice in one transaction,               │
│                             always returns the SAME rows, even if another                 │
│                             transaction committed changes in between —                       │
│                             MySQL's default                                                     │
│                                                                                                     │
│   SERIALIZABLE           →  strongest: transactions behave AS IF they ran               │
│                               one at a time, in some order, with zero                       │
│                               interference — safest, but most restrictive                       │
│                               (can force one transaction to fail/retry if                          │
│                               it would have conflicted with another)                                  │
└──────────────────────────────────────────────────────────┘
```

```go
tx, err := db.BeginTx(ctx, &sql.TxOptions{
	Isolation: sql.LevelSerializable,
})
```

**Higher isolation = stronger correctness guarantees, but more contention**
(transactions blocking or failing/retrying against each other) — exactly
the same fundamental trade-off as choosing a `Mutex` vs. finer-grained
locking in Module 12, just at the scale of a whole database instead of one
process's memory.

---

### Migration

A migration is a **version-controlled, incremental change to your database
schema** — `CREATE TABLE`, `ALTER TABLE ADD COLUMN`, etc. — applied in
order, tracked so every environment (your laptop, a teammate's, production)
ends up with an identical schema history.

```
migrations/
├── 0001_create_accounts.up.sql      CREATE TABLE accounts (...);
├── 0001_create_accounts.down.sql     DROP TABLE accounts;
├── 0002_add_email_index.up.sql        CREATE INDEX idx_accounts_email ON accounts(email);
└── 0002_add_email_index.down.sql       DROP INDEX idx_accounts_email;
```

```
┌──────────────────────────────────────────────────────────┐
│   Fresh database  ──▶  apply 0001 up  ──▶  apply 0002 up  ──▶  current schema  │
│                                                                                    │
│   Need to undo the last change?  ──▶  apply 0002 DOWN  ──▶  back to after 0001       │
│                                                                                          │
│   A "migrations applied" table in the database itself tracks which                        │
│   numbered migrations have already run, so re-running the migration                          │
│   tool is always safe — it only applies what's NEW.                                             │
└──────────────────────────────────────────────────────────┘
```

The most common tool is `golang-migrate/migrate`, usable as a CLI or
embedded directly in your Go program; this module's projects use a simpler,
hand-rolled version (a numbered `migrations/` folder applied at startup) to
keep the mechanism fully visible without an extra dependency.

---

### Connection Pool

`sql.DB` is **already a connection pool**, not a single connection — Go
manages a set of underlying connections for you, reusing them across
queries instead of opening a new TCP connection per request (which would be
disastrous for performance under real load).

```go
db.SetMaxOpenConns(25)                  // hard cap — protects the DATABASE from being overwhelmed
db.SetMaxIdleConns(25)                    // how many can sit idle, ready for reuse, at once
db.SetConnMaxLifetime(5 * time.Minute)      // recycle connections periodically (helps with
                                              // load balancers, DB restarts, stale connections)
```

```
┌──────────────────────────────────────────────────────────┐
│   Request A ──▶ borrows a connection from the pool ──▶ query ──▶ returns it     │
│   Request B ──▶ borrows a (possibly DIFFERENT, possibly the SAME)                  │
│                   connection ──▶ query ──▶ returns it                                │
│                                                                                          │
│   If ALL connections are currently in use and MaxOpenConns is reached,                    │
│   a new request WAITS for one to free up, rather than opening yet                            │
│   another raw connection — this is EXACTLY Module 12's semaphore                                │
│   pattern (a buffered channel limiting concurrency), just implemented                              │
│   for you inside sql.DB itself.                                                                        │
└──────────────────────────────────────────────────────────┘
```

Since every HTTP request in Module 15/16 runs on its own goroutine, and
they likely all share one `*sql.DB`, this pool is exactly what prevents a
traffic spike from opening hundreds of simultaneous raw database
connections — a very real way an otherwise-correct API can accidentally
take down its own database under load.

---

### Repository Pattern

A **repository** is an interface abstracting data access — "get a user by
ID," "save an account" — so the rest of your application depends on that
interface, never on `database/sql`, GORM, or sqlx directly. This is Module
06's interfaces and Module 14's mocking, applied specifically to the data
layer.

```go
type AccountRepository interface {
	GetByID(ctx context.Context, id int) (*Account, error)
	Save(ctx context.Context, acc *Account) error
}

// The REAL implementation, backed by an actual database:
type sqliteAccountRepository struct {
	db *sql.DB
}

func (r *sqliteAccountRepository) GetByID(ctx context.Context, id int) (*Account, error) {
	// real SQL query here
}

// A FAKE implementation, used only in tests (Module 14's mocking, again):
type fakeAccountRepository struct {
	accounts map[int]*Account
}

func (r *fakeAccountRepository) GetByID(ctx context.Context, id int) (*Account, error) {
	// in-memory map lookup, no database at all
}
```

```
┌──────────────────────────────────────────────────────────┐
│   Business logic (e.g. TransferMoney)                              │
│        │                                                              │
│        │  depends ONLY on the AccountRepository INTERFACE                │
│        ▼                                                                    │
│   AccountRepository                                                            │
│        │                                                                          │
│        ├──▶ sqliteAccountRepository   (production: real SQL, real DB)                │
│        └──▶ fakeAccountRepository      (tests: in-memory, instant, no setup)             │
│                                                                                               │
│   TransferMoney's code is IDENTICAL either way — it never imports                              │
│   database/sql, sqlx, or any driver at all, exactly like Module 06's                              │
│   TransactionLogger example applied at the scale of a whole data layer                               │
└──────────────────────────────────────────────────────────┘
```

This is *the* pattern that makes real applications testable without a
database in the test loop at all — Module 14's whole test suite ran in
milliseconds with zero database setup specifically because of this
separation; the repository pattern is how that same benefit extends to
code that ultimately does need to persist real data.

---

Onto the projects — Banking API uses plain `database/sql` with explicit
transactions and isolation levels for money transfers (the classic example
transactions exist for); Inventory uses `sqlx` with the repository pattern,
migrations, and schema-level constraints for a small product catalog.
