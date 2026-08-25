# Project 2 — Inventory (sqlx + Repository Pattern)

```
inventory/
├── go.mod
├── db.go                       connection + migrations
├── main.go                       business logic (LowStockReport, Restock)
├── models/
│   └── product.go                 Product struct with `db` tags
├── repository/
│   ├── product_repository.go        ProductRepository interface + SQLite impl
│   └── fake_repository.go             in-memory fake, same interface
└── migrations/
    └── 0001_create_products.up.sql
```

## Setup

```bash
cd inventory
go mod tidy    # fetches sqlx and the SQLite driver — needs internet access
go run .
```

---

## The Repository Pattern, Made Concrete

This project's whole point is proving one claim: **business logic written
against an interface doesn't care what's on the other side of it.**
`main.go` runs the *exact same* `LowStockReport` and `Restock` functions
against two completely different repositories, back to back:

```
┌──────────────────────────────────────────────────────────┐
│                                                                    │
│   LowStockReport(ctx, repo, "electronics", 5)                        │
│   Restock(ctx, repo, "HUB-003", 20)                                     │
│                                                                             │
│        │                    │                                                │
│        │  repo = FakeProductRepository   repo = SQLiteProductRepository        │
│        ▼                    ▼                                                    │
│   ┌─────────────┐    ┌──────────────────────┐                                      │
│   │ in-memory      │    │ real SQL against            │                                      │
│   │ map[int]*Product│    │ a real products.db file     │                                      │
│   │ (Module 12's     │    │ via sqlx + modernc.org/sqlite │                                    │
│   │  Mutex pattern)    │    │                                │                                    │
│   └─────────────┘    └──────────────────────┘                                      │
│                                                                                          │
│   Both print IDENTICAL results. main.go's business logic functions NEVER               │
│   import sqlx, database/sql, or any driver — only "inventory/repository"                  │
│   for the INTERFACE.                                                                          │
└──────────────────────────────────────────────────────────┘
```

This is exactly why Module 14's whole test suite ran with zero database
setup — a real test suite for this project's `LowStockReport`/`Restock`
would use `FakeProductRepository` throughout, running in milliseconds, with
`SQLiteProductRepository` reserved for a smaller set of true integration
tests (Module 14's distinction) confirming the real SQL actually works.

## sqlx in Action: Less Boilerplate Than Raw `database/sql`

```go
// Raw database/sql (Banking API's style) — manual, positional Scan:
var p Product
db.QueryRow("SELECT id, sku, name, category, price, quantity FROM products WHERE id = ?", id).
	Scan(&p.ID, &p.SKU, &p.Name, &p.Category, &p.Price, &p.Quantity)

// sqlx — scans directly into the struct via `db` tags:
var p models.Product
db.GetContext(ctx, &p, "SELECT * FROM products WHERE id = ?", id)
```

```
┌──────────────────────────────────────────────────────────┐
│   db.GetContext(&p, query, args...)    →  ONE row, into a struct        │
│   db.SelectContext(&products, query, args...)  →  MANY rows, into a []struct │
│   db.NamedExecContext(query, p)          →  INSERT/UPDATE, binding FROM         │
│                                                a struct's fields by NAME            │
│                                                (:sku, :name, ... match `db` tags)      │
│                                                                                            │
│   You still wrote every one of these queries yourself — sqlx never                          │
│   generates SQL for you (that's GORM's job) — it only removes the                              │
│   repetitive Scan(&a, &b, &c, ...) argument list.                                                 │
└──────────────────────────────────────────────────────────┘
```

## Constraint Enforcement, Both Layers

```bash
# Try creating a duplicate SKU against the REAL repository:
```
```
┌──────────────────────────────────────────────────────────┐
│   repo.Create(ctx, &Product{SKU: "MOU-001", ...})   (SKU already exists)     │
│        │                                                                        │
│        ▼                                                                          │
│   SQLite's UNIQUE constraint on products.sku REJECTS the insert                      │
│   at the DATABASE level — sqlx surfaces this as a normal Go error                       │
│        │                                                                                   │
│        ▼                                                                                     │
│   "UNIQUE constraint failed: products.sku"                                                     │
│                                                                                                     │
│   FakeProductRepository.Create MANUALLY checks for this same case                                    │
│   (looping over existing products) and returns an error with a                                          │
│   MATCHING message — so callers get consistent behavior regardless                                         │
│   of which repository is actually running underneath.                                                        │
└──────────────────────────────────────────────────────────┘
```

This is worth sitting with: **the fake repository has to deliberately
replicate the real database's constraint behavior** for the interface
contract to actually hold. A fake that "forgets" to enforce uniqueness
would let tests pass against behavior the real, production database would
never actually allow — a genuine risk of the repository pattern worth
knowing about, not just a convenience to take for granted.

## Try It Yourself
- Add a `Reserve(ctx, sku string, quantity int) error` to the interface
  that decrements stock atomically (SQLite: inside a transaction like
  Banking API's `Transfer`; fake: inside the existing Mutex) — then write a
  small test-style function calling it against both repositories and
  confirming they behave identically for an over-reservation attempt
- Extend `ListByCategory` to accept a sort field and order (Module 16's
  Sorting section), and add the equivalent logic to
  `FakeProductRepository.ListByCategory` — a good exercise in keeping two
  implementations of one interface in sync as it grows
- Deliberately break `FakeProductRepository`'s uniqueness check (comment it
  out) and see what silently starts working differently between the two
  repositories — a hands-on look at the "fakes can drift from real behavior"
  risk called out above
