# Project 2 — E-commerce API (Chi)

A richer, multi-feature product API built with **Chi**, covering filtering,
sorting, search, per-IP rate limiting, and TTL-based caching.

## Setup

```bash
cd ecommerce-api
go mod tidy    # fetches Chi and golang.org/x/time — needs internet access
go run .
```

---

## Architecture Overview

```
┌──────────────────────────────────────────────────────────────┐
│                                                                      │
│   Request                                                              │
│      │                                                                    │
│      ▼                                                                      │
│  ┌──────────────────────────────────────────────┐                             │
│  │  loggingMiddleware   (structured JSON, log/slog)   │                             │
│  │  ┌──────────────────────────────────────────┐  │                             │
│  │  │  rateLimitMiddleware  (per-IP token bucket)  │  │                             │
│  │  │  ┌────────────────────────────────────┐  │  │                             │
│  │  │  │  chi router (plain net/http funcs)     │  │  │                             │
│  │  │  │       │                                   │  │  │                             │
│  │  │  │       ▼                                     │  │  │                             │
│  │  │  │  listProductsHandler                          │  │  │                             │
│  │  │  │       │                                          │  │  │                             │
│  │  │  │       ▼                                            │  │  │                             │
│  │  │  │  cache.Get(queryString) ──▶ HIT? return cached        │  │  │                             │
│  │  │  │       │  MISS                                            │  │  │                             │
│  │  │  │       ▼                                                    │  │  │                             │
│  │  │  │  filterProducts → sortProducts → paginate                     │  │  │                             │
│  │  │  │       │                                                          │  │  │                             │
│  │  │  │       ▼                                                            │  │  │                             │
│  │  │  │  cache.Set(queryString, result)                                       │  │  │                             │
│  │  │  └────────────────────────────────────┘  │  │                             │
│  │  └──────────────────────────────────────────┘  │                             │
│  └──────────────────────────────────────────────┘                             │
└──────────────────────────────────────────────────────────────┘
```

---

## Filtering + Sorting + Pagination, Combined

```bash
curl "http://localhost:8080/products?category=electronics&minPrice=30&sort=price&order=desc&page=1&limit=10"
```

```
┌──────────────────────────────────────────────────────────┐
│   All 5 products                                                  │
│        │                                                              │
│        ▼  filterProducts: category=electronics, minPrice=30              │
│   ┌─────────────────────────────────┐                                       │
│   │ Mechanical Keyboard   $89.99         │  ◀── electronics, ≥$30                  │
│   │ USB-C Hub             $45.00           │  ◀── electronics, ≥$30                  │
│   └─────────────────────────────────┘     (Wireless Mouse excluded: $24.99 < 30) │
│        │                                    (Standing Desk, Desk Lamp: not      │
│        ▼  sortProducts: price, desc            electronics)                       │
│   ┌─────────────────────────────────┐                                                │
│   │ Mechanical Keyboard   $89.99         │  ◀── highest first                          │
│   │ USB-C Hub             $45.00           │                                                │
│   └─────────────────────────────────┘                                                          │
│        │                                                                                          │
│        ▼  paginate: page=1, limit=10                                                                 │
│   {"data": [...2 items...], "page":1, "limit":10, "total":2, "totalPages":1}                            │
└──────────────────────────────────────────────────────────┘
```

## Search (separate endpoint, separate strategy)

```bash
curl "http://localhost:8080/products/search?q=usb"
```
```
┌──────────────────────────────────────────────────────────┐
│   store.Search("usb")                                            │
│        │  lowercases query AND every product's Name/Description       │
│        ▼                                                                 │
│   "USB-C Hub" name contains "usb"?           → YES                          │
│   "Desk Lamp" description contains "usb"?      → YES ("USB charging")          │
│   (all others)                                    → NO                            │
│        │                                                                             │
│        ▼                                                                                │
│   {"query":"usb", "results":[USB-C Hub, Desk Lamp], "count":2}                             │
└──────────────────────────────────────────────────────────┘
```

## Caching — Watch It Happen, via the `X-Cache` Header

```bash
curl -i "http://localhost:8080/products?category=electronics" | grep X-Cache
# X-Cache: MISS       ◀── first request: computed fresh

curl -i "http://localhost:8080/products?category=electronics" | grep X-Cache
# X-Cache: HIT        ◀── second, identical request within 5 seconds: served from cache

sleep 6
curl -i "http://localhost:8080/products?category=electronics" | grep X-Cache
# X-Cache: MISS       ◀── TTL expired — recomputed
```

```
┌──────────────────────────────────────────────────────────┐
│   Request A: ?category=electronics                                  │
│        cache key: "category=electronics"                               │
│        MISS → compute → cache.Set("category=electronics", result)          │
│                                                                                 │
│   Request B: ?category=electronics&page=2                                        │
│        cache key: "category=electronics&page=2"   ◀── a DIFFERENT key!             │
│        MISS → computed SEPARATELY, cached separately                                  │
│                                                                                             │
│   Every distinct query string is its OWN cache entry — this is simple                        │
│   and correct, but means highly varied query patterns (many different                           │
│   filter/sort/page combinations) get relatively few cache HITS compared                             │
│   to a smaller, more repetitive set of common queries.                                                 │
└──────────────────────────────────────────────────────────┘
```

Now watch cache invalidation on a write:
```bash
curl -X POST http://localhost:8080/products \
  -d '{"name":"Webcam","description":"1080p webcam","category":"electronics","price":39.99,"inStock":true}'

curl -i "http://localhost:8080/products?category=electronics" | grep X-Cache
# X-Cache: MISS   ◀── the POST called cache.Invalidate() — the whole cache was cleared,
#                      so this now-stale-if-cached listing recomputes and includes the webcam
```

## Rate Limiting — Trigger It Deliberately

```bash
for i in $(seq 1 20); do
  curl -s -o /dev/null -w "%{http_code} " http://localhost:8080/products
done
echo
# 200 200 200 200 200 200 200 200 200 200 429 429 429 429 429 429 429 429 429 429
#                                        ▲
#                                        burst of 10 consumed, then throttled
```

```
┌──────────────────────────────────────────────────────────┐
│                  Token bucket for ONE client IP                       │
│                                                                            │
│   starts full:  ● ● ● ● ● ● ● ● ● ●   (burst = 10)                          │
│                                                                                │
│   requests 1-10:  each takes ONE token   →  200 OK, 200 OK, ... (bucket empties) │
│   request 11+:     bucket EMPTY           →  429 Too Many Requests                 │
│                                                                                        │
│   meanwhile, the bucket REFILLS at 5 tokens/second — wait ~1 second and          │
│   a few requests succeed again, refilling only as fast as the sustained            │
│   rate allows                                                                         │
└──────────────────────────────────────────────────────────┘
```

Try it from what looks like a different client (rate limits are per-IP, so
this won't actually bypass anything on `localhost`, but it's worth
confirming in the code): `getLimiter` in `middleware.go` creates a
*separate* bucket per unique `r.RemoteAddr`, so one noisy client never
throttles anyone else.

## Structured Logging Output

```bash
go run .
# in another terminal: curl http://localhost:8080/products
```
```json
{"time":"2026-08-24T10:00:00Z","level":"INFO","msg":"request","method":"GET","path":"/products","query":"","duration_ms":0}
```
Every field is a real JSON key — a log aggregation tool (or `jq`) can filter
and query these directly, unlike a free-text log line.

---

## Case Study: Why Search Isn't Cached, But Listing Is

```
┌──────────────────────────────────────────────────────────┐
│   /products?category=electronics&sort=price   ◀── LOW cardinality:         │
│   /products?category=furniture&sort=name          a handful of REAL           │
│   /products?inStock=true                            users will tend to           │
│                                                        repeat similar/identical      │
│                                                        combinations often — HIGH        │
│                                                        cache hit rate                     │
│                                                                                              │
│   /products/search?q=wireless mouse for gaming    ◀── HIGH cardinality:          │
│   /products/search?q=usb c hub 2026                    nearly every search           │
│   /products/search?q=desk lamp with clip                query is UNIQUE — a              │
│                                                            cache would mostly fill           │
│                                                            with one-time-use entries            │
│                                                            that are never hit again,               │
│                                                            wasting memory for close to               │
│                                                            zero benefit                                 │
└──────────────────────────────────────────────────────────┘
```

This is a genuinely important caching instinct beyond "cache everything
expensive": a cache's value depends on the **hit rate** you'll realistically
get, which depends heavily on how *repetitive* real traffic to that specific
endpoint actually is — not just on how expensive the underlying computation
is.

## Try It Yourself
- Add a `?category=` **allow-list** check (like `sortProducts`'s
  `allowedSortFields`) that rejects an unrecognized category with a 400
  instead of silently returning zero results — which behavior is actually
  more helpful to an API consumer?
- Change the rate limiter to read a per-client **API key** header instead of
  IP address — more robust for clients behind a shared NAT/proxy, where many
  real users could share one IP
- Add a `GET /products/categories` endpoint returning the distinct list of
  categories currently in the store — and decide whether it belongs in the
  cache too, using this README's cardinality reasoning to justify your answer
