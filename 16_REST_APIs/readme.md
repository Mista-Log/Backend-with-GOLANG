# Go for Beginners — Module 16: REST APIs

## Contents

1. **[16-rest-apis.md](./16-rest-apis.md)** — The four major frameworks
   (Gin, Chi, Echo, Fiber) compared directly, with a decision diagram for
   choosing between them and stdlib — then every REST topic: CRUD (and the
   real PUT-vs-PATCH distinction), validation via struct tags, offset vs.
   cursor pagination, filtering, sorting (with a security note on allow-
   listing sortable fields), search, rate limiting (token bucket, per-
   client), structured logging (`log/slog`), caching (with an honest
   "invalidation is the hard part" warning), and Swagger/OpenAPI generation
   from code comments. Diagrams throughout every section.

2. **[todo-api/](./todo-api)** — Built with **Gin**. Full CRUD, validation
   via `binding` struct tags, offset-based pagination with a proper response
   envelope, a real PUT-vs-PATCH demonstration (including a `*bool` trick
   for distinguishing "not sent" from "explicitly false"), and Swagger
   annotations on every handler.

3. **[ecommerce-api/](./ecommerce-api)** — Built with **Chi** (plain
   `net/http`-compatible handlers, deliberately contrasted against Gin's
   own Context style). Combined filtering + sorting + pagination, a separate
   search endpoint, per-IP token-bucket rate limiting, TTL-based caching
   with an `X-Cache` header you can watch directly, and structured JSON
   logging.

Both project READMEs are diagram-heavy — full request-flow diagrams, before/
after query walkthroughs, and case studies on real design trade-offs (why
`PatchTodoRequest.Done` is a pointer, why search isn't cached but listing
is, what a token bucket actually looks like mid-drain).

## Suggested Order

```
REST APIs guide ──▶ Todo API (Gin) ──▶ E-commerce API (Chi)
                      (CRUD, validation,      (filtering, sorting, search,
                       pagination, Swagger)     rate limiting, caching, logging)
```

The two projects deliberately use different frameworks with different
philosophies (Gin's owned Context vs. Chi's stdlib-compatible handlers) so
you experience both styles directly, not just read about the difference.

## Setup — These Projects Need Real Dependencies

Unlike every previous module, both projects here depend on external
packages (Gin, Chi, `golang.org/x/time`) and need a one-time, internet-
connected setup step before they'll run:

```bash
cd todo-api && go mod tidy && go run .
cd ecommerce-api && go mod tidy && go run .
```

`go mod tidy` fetches the dependencies and writes `go.sum` (Module 09) —
after that, both projects behave exactly like every earlier one.

*Note: this module builds on Modules 00–15 — start there first if you
haven't already, especially Module 15 (the `net/http` fundamentals every
framework here builds on), Module 08 (validation concepts, now automated
via struct tags), and Module 12 (the Mutex/RWMutex/token-bucket concurrency
safety both projects' shared state relies on).*
