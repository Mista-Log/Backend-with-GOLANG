# 16. REST APIs

Module 15 built a real API on the standard library alone — genuinely
sufficient for most projects. This module covers the frameworks you'll see
across real Go job postings and codebases, and the topics that turn a basic
CRUD API into a production-grade one.

---

## Frameworks

All four of these are **routers with extras** — none of them replace
`net/http`'s `Handler` model from Module 15; they build on top of it. None
are in the standard library, so each needs `go get` before use.

### Gin

The most widely-adopted Go web framework, known for speed and a large
middleware ecosystem. Built-in JSON binding + validation is its biggest
everyday convenience over stdlib.

```bash
go get github.com/gin-gonic/gin
```
```go
r := gin.Default() // includes logging + recovery middleware out of the box

r.GET("/tasks/:id", func(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id})
})

r.Run(":8080")
```

### Chi

A minimal, idiomatic router that stays extremely close to stdlib's own
`net/http` types — `chi.Router` **is** an `http.Handler`, and Chi handlers
are ordinary `func(w http.ResponseWriter, r *http.Request)`, unlike Gin's
own `*gin.Context` abstraction. Chi is the closest thing to "stdlib plus
better routing and middleware composition."

```bash
go get github.com/go-chi/chi/v5
```
```go
r := chi.NewRouter()
r.Use(middleware.Logger)

r.Get("/tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	w.Write([]byte(id))
})

http.ListenAndServe(":8080", r)
```

### Echo

Similar in spirit to Gin (its own Context type, built-in binding/validation,
a large middleware set), with a slightly different API style and a strong
focus on performance and clean error handling via a centralized
`HTTPErrorHandler`.

```bash
go get github.com/labstack/echo/v4
```
```go
e := echo.New()

e.GET("/tasks/:id", func(c echo.Context) error {
	id := c.Param("id")
	return c.JSON(http.StatusOK, map[string]string{"id": id})
})

e.Start(":8080")
```

### Fiber

Built on `fasthttp` instead of the standard library's `net/http` —
deliberately Express.js-flavored (a deliberate design choice to feel
familiar to developers coming from Node.js), and generally the fastest of
the four in raw throughput benchmarks. **The trade-off:** because it doesn't
use `net/http`, Fiber's request/response types (`fiber.Ctx`) are **not**
interchangeable with stdlib `http.Handler`-based code or most of the wider
Go HTTP ecosystem (like `httptest` for testing, or middleware written for
`net/http`) — everything needs a Fiber-specific equivalent.

```bash
go get github.com/gofiber/fiber/v2
```
```go
app := fiber.New()

app.Get("/tasks/:id", func(c *fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{"id": id})
})

app.Listen(":8080")
```

### Choosing

```
┌──────────────────────────────────────────────────────────┐
│                                                                  │
│   Want to stay closest to stdlib, most compatible with the         │
│   wider ecosystem (testing tools, existing net/http middleware)?      │
│      → Chi                                                                │
│                                                                                │
│   Want the biggest community, most tutorials/Stack Overflow                     │
│   answers, built-in validation, and don't mind a framework-owned                   │
│   Context type?                                                                       │
│      → Gin                                                                              │
│                                                                                              │
│   Want Gin-like ergonomics with a different API style and strong                              │
│   centralized error handling?                                                                    │
│      → Echo                                                                                          │
│                                                                                                          │
│   Want maximum raw throughput and are fine giving up net/http                                            │
│   compatibility entirely?                                                                                  │
│      → Fiber                                                                                                 │
│                                                                                                                  │
│   Building something small, or want zero new dependencies at all?                                                 │
│      → Module 15's stdlib approach — genuinely sufficient for most                                                   │
│         real APIs, especially since Go 1.22's routing improvements                                                     │
└──────────────────────────────────────────────────────────┘
```

This module's two projects use **Gin** (Todo API) and **Chi** (E-commerce
API) specifically so you see both a framework-owned-Context style and a
stdlib-compatible style in real, contrasting use — Echo and Fiber follow
similar shapes to one or the other once you've used both.

---
## Topics

### CRUD

Create, Read, Update, Delete — the four operations nearly every REST
resource needs, conventionally mapped to HTTP methods (Module 15's routing
table did exactly this for tasks):

```
┌────────────────────────────────────────────────────┐
│   POST    /products         →  CREATE                     │
│   GET     /products          →  READ (list)                  │
│   GET     /products/{id}      →  READ (one)                     │
│   PUT     /products/{id}       →  UPDATE (full replace)            │
│   PATCH   /products/{id}        →  UPDATE (partial)                   │
│   DELETE  /products/{id}         →  DELETE                               │
└────────────────────────────────────────────────────┘
```

**`PUT` vs. `PATCH`**: `PUT` conventionally means "replace this resource
entirely with what I'm sending" (omitted fields should reset to their
zero/default value); `PATCH` means "apply only these specific changes,
leave everything else untouched." Many real APIs are sloppy about this
distinction — being deliberate about it is a genuine polish signal.

---

### Validation

Beyond basic type-correctness (JSON decoding already handles that), real
validation checks **business rules**: required fields, ranges, formats.
Gin and Echo both integrate `go-playground/validator` via struct tags:

```go
type CreateProductRequest struct {
	Name  string  `json:"name" binding:"required,min=2,max=100"`
	Price float64 `json:"price" binding:"required,gt=0"`
	SKU   string  `json:"sku" binding:"required,alphanum"`
}

func createProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// req is now GUARANTEED to satisfy every `binding` rule above
}
```

```
┌────────────────────────────────────────────────────┐
│   binding:"required"     →  must be present / non-zero-value        │
│   binding:"min=2,max=100" →  string length OR numeric range              │
│   binding:"gt=0"           →  must be greater than zero                     │
│   binding:"email"           →  must look like a valid email address            │
│   binding:"alphanum"         →  letters and digits only                          │
└────────────────────────────────────────────────────┘
```

This is the same idea as Module 08's hand-written `validation` package,
just declared via struct tags and enforced automatically by the framework
at bind time, instead of called explicitly line by line.

---

### Pagination

Never return an unbounded list — pagination caps response size and lets
clients page through large collections.

**Offset-based** (simple, but can skip/repeat items if the underlying data
changes between requests):
```
GET /products?page=2&limit=20
```
```go
page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
offset := (page - 1) * limit
results := allProducts[offset : offset+limit]
```

**Cursor-based** (more robust for frequently-changing data — each page
points to "continue after this specific item," not a numeric position):
```
GET /products?cursor=eyJpZCI6NDJ9&limit=20
```

```
┌────────────────────────────────────────────────────────┐
│   Offset:  page 1 = items 1-20, page 2 = items 21-40, ...          │
│             PROBLEM: if item #15 is deleted between requests,           │
│             page 2 now starts from the WRONG position (everything          │
│             shifted by one) — you'd skip or repeat an item                    │
│                                                                                    │
│   Cursor:   "give me 20 items AFTER the one with ID 42"                             │
│              ROBUST to insertions/deletions elsewhere in the list,                     │
│              since it's anchored to a specific item, not a raw count                      │
└────────────────────────────────────────────────────────┘
```

Every paginated response should also tell the client what's available:
```json
{
  "data": [...],
  "page": 2,
  "limit": 20,
  "total": 143,
  "totalPages": 8
}
```

---

### Filtering

Letting clients narrow results via query parameters:
```
GET /products?category=electronics&minPrice=10&maxPrice=100&inStock=true
```
```go
func filterProducts(products []Product, c *gin.Context) []Product {
	var filtered []Product
	for _, p := range products {
		if cat := c.Query("category"); cat != "" && p.Category != cat {
			continue
		}
		if minStr := c.Query("minPrice"); minStr != "" {
			if min, _ := strconv.ParseFloat(minStr, 64); p.Price < min {
				continue
			}
		}
		filtered = append(filtered, p)
	}
	return filtered
}
```
For a real database-backed API, filters translate into `WHERE` clauses
instead of an in-memory loop — the query-parameter *contract* with clients
stays the same either way.

---

### Sorting

```
GET /products?sort=price&order=desc
```
```go
sort.Slice(products, func(i, j int) bool {
	switch c.Query("sort") {
	case "price":
		if c.Query("order") == "desc" {
			return products[i].Price > products[j].Price
		}
		return products[i].Price < products[j].Price
	default: // "name", or unspecified
		return products[i].Name < products[j].Name
	}
})
```
**Validate the `sort` field against an allow-list** of real column/field
names — passing a raw, unchecked client value straight into a database
`ORDER BY` clause is a real SQL-injection-adjacent risk if you're building
the query as a string.

---

### Search

Free-text search across one or more fields:
```
GET /products/search?q=wireless+mouse
```
```go
func searchProducts(products []Product, query string) []Product {
	query = strings.ToLower(query)
	var results []Product
	for _, p := range products {
		if strings.Contains(strings.ToLower(p.Name), query) ||
			strings.Contains(strings.ToLower(p.Description), query) {
			results = append(results, p)
		}
	}
	return results
}
```
This substring approach is fine for small in-memory datasets (exactly what
this module's projects use); real production search at scale typically
delegates to a database's full-text search features or a dedicated search
engine (Elasticsearch, Meilisearch) — the API *contract* (`?q=...`) usually
stays identical regardless of what's answering it underneath.

---

### Rate Limiting

Caps how many requests a client can make in a given window — protects a
service from being overwhelmed (accidentally or deliberately) by one
client. The standard building block is `golang.org/x/time/rate`'s **token
bucket**:

```go
import "golang.org/x/time/rate"

limiter := rate.NewLimiter(5, 10) // 5 requests/second sustained, burst up to 10

func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

```
┌────────────────────────────────────────────────────────┐
│                     Token Bucket                                  │
│                                                                        │
│   ┌─────────────┐   refills at 5 tokens/second                          │
│   │  ● ● ● ● ●  │   (holds up to 10 — the "burst" capacity)                 │
│   │  ● ● ● ● ●  │                                                              │
│   └─────────────┘                                                                │
│                                                                                       │
│   Each request TAKES one token. Bucket empty? → reject (429 Too Many               │
│   Requests). Requests arriving slower than the refill rate never run                  │
│   out of tokens at all — only a genuine BURST beyond capacity gets                       │
│   throttled.                                                                                │
└────────────────────────────────────────────────────────┘
```

**Per-client** rate limiting (the far more common real need) requires one
limiter *per client identity* (API key, IP address), typically stored in a
map — exactly the Mutex-or-sync.Map decision from Module 12, applied here.

---

### Logging

Beyond Module 15's basic request-timing middleware, production logging
typically means **structured** logs (JSON, not free-text) so they're
machine-parseable by log aggregation tools:

```go
import "log/slog" // standard library, Go 1.21+

logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
```
```json
{"time":"2026-08-24T10:00:00Z","level":"INFO","msg":"request","method":"GET","path":"/products","duration_ms":12}
```
`log/slog` (standard library since Go 1.21) covers most structured-logging
needs without any external dependency at all.

---

### Caching

Storing expensive-to-compute or frequently-requested results so subsequent
identical requests skip the work entirely:

```go
type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

var cache sync.Map // Module 12: fine here since it's mostly independent keys

func cachedHandler(key string, ttl time.Duration, compute func() []byte) []byte {
	if v, ok := cache.Load(key); ok {
		entry := v.(cacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.data // CACHE HIT — skip the expensive work entirely
		}
	}
	data := compute() // CACHE MISS — do the real work
	cache.Store(key, cacheEntry{data: data, expiresAt: time.Now().Add(ttl)})
	return data
}
```

```
┌────────────────────────────────────────────────────┐
│   Request 1: GET /products?category=electronics            │
│      cache MISS → compute (slow) → STORE in cache               │
│                                                                      │
│   Request 2 (within TTL): same query                                  │
│      cache HIT → return instantly, no recomputation                      │
│                                                                              │
│   Request 3 (after TTL expires): same query                                    │
│      cache MISS again → recompute → refresh the cache                             │
└────────────────────────────────────────────────────┘
```

**The hardest part of caching is invalidation** — if a product's price
changes, every cache entry that might include it needs to expire or be
explicitly cleared, or clients see stale data. This module's E-commerce API
project uses a short TTL specifically to sidestep needing precise
invalidation logic, a common, pragmatic real-world trade-off.

---

### Swagger / OpenAPI

**OpenAPI** is a language-agnostic specification format (JSON/YAML)
describing a REST API's endpoints, request/response shapes, and
authentication — machine-readable documentation that tools can generate
interactive docs, client SDKs, and test cases from. **Swagger** is the
tooling ecosystem built around that spec (Swagger UI renders it as
browsable, interactive documentation).

The common Go workflow (via `swaggo/swag`) generates the spec from
**comments directly above your handlers**, so documentation lives right
next to the code it describes:

```go
// CreateProduct godoc
// @Summary      Create a new product
// @Description  Adds a product to the catalog
// @Tags         products
// @Accept       json
// @Produce      json
// @Param        product  body      CreateProductRequest  true  "Product to create"
// @Success      201      {object}  Product
// @Failure      400      {object}  ErrorResponse
// @Router       /products [post]
func createProduct(c *gin.Context) { ... }
```

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init                                          # scans comments, generates docs/
```

```
┌────────────────────────────────────────────────────────┐
│   Your handler's @-comments                                     │
│        │                                                            │
│        ▼                                                              │
│   swag init  →  generates an openapi.json/yaml spec                     │
│        │                                                                  │
│        ▼                                                                     │
│   Swagger UI (served at e.g. /swagger/index.html) renders it as              │
│   an interactive page — anyone can browse every endpoint, see                    │
│   expected request/response shapes, and even send test requests                     │
│   directly from the browser, with ZERO separate documentation effort               │
└────────────────────────────────────────────────────────┘
```

---

Onto the projects — Todo API (Gin) covers CRUD, validation, pagination, and
Swagger annotations in a framework-owned-Context style; E-commerce API
(Chi) covers filtering, sorting, search, rate limiting, caching, and
logging in a stdlib-compatible style, across a richer multi-resource domain.
