# Reference Project — production-service

A small, runnable skeleton applying every section of Module 19's guide at
once. This isn't a numbered exercise project like earlier modules — it's a
**reference architecture** you can point to, copy from, and adapt.

```
production-service/
├── go.mod
├── cmd/
│   └── server/
│       └── main.go                 THIN — wiring + startup + shutdown only
└── internal/
    ├── config/
    │   └── config.go                  env vars + defaults
    ├── domain/
    │   └── order.go                     Order entity + OrderRepository PORT
    ├── repository/
    │   └── order_repository.go            in-memory ADAPTER implementing the port
    ├── service/
    │   └── order_service.go                use cases: PlaceOrder, CancelOrder, ...
    └── transport/
        └── http/
            └── router.go                    HTTP handlers + health checks
```

## Setup

```bash
cd production-service
go mod tidy    # fetches prometheus/client_golang — needs internet access
go run ./cmd/server
```

---

## 1. The Dependency Direction, Made Concrete

```
┌──────────────────────────────────────────────────────────────┐
│                                                                      │
│   internal/transport/http    (outermost — imports service AND domain) │
│         │                                                               │
│         │  depends on                                                     │
│         ▼                                                                    │
│   internal/service            (imports domain ONLY)                            │
│         │                                                                         │
│         │  depends on                                                                │
│         ▼                                                                               │
│   internal/domain              (imports NOTHING from this project —                        │
│                                   no net/http, no prometheus, nothing)                         │
│         ▲                                                                                         │
│         │  ALSO depends on domain (implements its interface)                                        │
│         │                                                                                              │
│   internal/repository          (imports domain, to implement OrderRepository)                            │
│                                                                                                               │
│   Every arrow points TOWARD domain. Nothing domain contains ever                                               │
│   imports repository, service, or transport/http — verify this yourself:                                          │
│   `grep -r "productionservice/internal" internal/domain/` returns NOTHING.                                            │
└──────────────────────────────────────────────────────────────┘
```

## 2. A Request, Traced Through Every Layer

```bash
curl -X POST http://localhost:8080/orders -d '{"item": "Keyboard", "amount": 89.99}'
```

```
┌──────────────────────────────────────────────────────────────┐
│   transport/http.NewRouter's "POST /orders" handler                    │
│        │  decodes JSON into createOrderRequest                            │
│        ▼                                                                     │
│   service.OrderService.PlaceOrder(ctx, "Keyboard", 89.99)                        │
│        │  constructs a domain.Order{Item, Amount, Status: "pending"}                │
│        ▼                                                                               │
│   domain.OrderRepository.Create(ctx, order)   ◀── called through the INTERFACE,           │
│        │                                            OrderService has no idea WHICH             │
│        │                                            concrete type is behind it                    │
│        ▼                                                                                              │
│   repository.InMemoryOrderRepository.Create(...)   ◀── the ACTUAL implementation                        │
│        │  assigns order.ID, stores it in a Mutex-protected map                                             │
│        ▼                                                                                                       │
│   back up through OrderService: ordersCreated.Inc() (metrics),                                                    │
│        logger.Info("order placed", ...) (structured log)                                                             │
│        ▼                                                                                                                  │
│   back up through the HTTP handler: writeJSON(201, order)                                                                    │
│        ▼                                                                                                                        │
│   curl receives: {"ID":1,"Item":"Keyboard","Amount":89.99,"Status":"pending"}                                                       │
└──────────────────────────────────────────────────────────────┘
```

## 3. Health Checks — Two Different Questions

```bash
curl -i http://localhost:8080/healthz   # 200 OK — the process is running, period
curl -i http://localhost:8080/readyz     # 200 OK — `ready()` returned nil
```

```
┌──────────────────────────────────────────────────────────┐
│   /healthz  →  ALWAYS 200 unless the process itself is dead/           │
│                  deadlocked. A Kubernetes liveness probe hitting            │
│                  this would only ever RESTART the container if                 │
│                  the Go process itself stopped responding entirely.               │
│                                                                                        │
│   /readyz   →  calls the injected `ready func() error` — in THIS demo,                  │
│                  always nil (nothing external to check), but in a real                     │
│                  service this pings a database/cache/downstream API.                          │
│                  A Kubernetes READINESS probe failing here STOPS routing                          │
│                  traffic to this instance, WITHOUT restarting it —                                   │
│                  exactly right when the process is fine but a dependency                                │
│                  is temporarily down.                                                                      │
└──────────────────────────────────────────────────────────┘
```

## 4. Metrics — Watch a Real Counter Increment

```bash
curl -s http://localhost:9090/metrics | grep orders_created_total
# orders_created_total 0

curl -X POST http://localhost:8080/orders -d '{"item": "Mouse", "amount": 24.99}'
curl -X POST http://localhost:8080/orders -d '{"item": "Monitor", "amount": 199.99}'

curl -s http://localhost:9090/metrics | grep orders_created_total
# orders_created_total 2
```

```
┌──────────────────────────────────────────────────────────┐
│   Metrics live on a SEPARATE server (:9090), not the main app             │
│   port (:8080) — a common real pattern, so metrics scraping                  │
│   (typically from an internal Prometheus server) never competes                 │
│   with real user traffic on the same listener, and can be firewalled              │
│   off from the public internet independently.                                        │
└──────────────────────────────────────────────────────────┘
```

## 5. Graceful Shutdown — Watch It Happen

```bash
go run ./cmd/server
# in another terminal, start a request that takes a moment, THEN Ctrl+C the server quickly:
curl -X POST http://localhost:8080/orders -d '{"item": "Desk", "amount": 349.00}' &
# immediately press Ctrl+C in the server's terminal
```

```
┌──────────────────────────────────────────────────────────┐
│   Ctrl+C  (sends SIGINT)                                             │
│        │                                                                │
│        ▼                                                                  │
│   main.go's <-sigCh unblocks                                                │
│        │                                                                       │
│        ▼                                                                         │
│   logger.Info("shutdown signal received, finishing in-flight requests")            │
│        │                                                                                │
│        ▼                                                                                   │
│   appServer.Shutdown(ctx)   [ctx has a 10-second deadline, from cfg.ShutdownTimeout]          │
│        │                                                                                          │
│        ├─▶ stops accepting NEW connections on :8080 immediately                                      │
│        ├─▶ the in-flight POST /orders request from curl is allowed to FINISH                             │
│        └─▶ once it (and anything else in flight) completes, Shutdown returns                                │
│        │                                                                                                        │
│        ▼                                                                                                           │
│   logger.Info("shutdown complete")   ──▶  process exits cleanly, exit code 0                                          │
│                                                                                                                          │
│   Without this, Ctrl+C would kill the process IMMEDIATELY — that in-flight                                                │
│   curl request could be cut off mid-response, with the client seeing a                                                       │
│   broken connection instead of its actual result.                                                                              │
└──────────────────────────────────────────────────────────┘
```

## 6. Testing This Architecture Without a Server or Database

Because `OrderService` depends only on the `domain.OrderRepository`
**interface**, Module 14's whole testing toolkit applies directly:

```go
type fakeOrderRepo struct {
	orders map[int]*domain.Order
}
// ... implement the 4 interface methods with a plain map, no locking even needed for a single-threaded test

func TestPlaceOrder(t *testing.T) {
	repo := &fakeOrderRepo{orders: map[int]*domain.Order{}}
	svc := service.NewOrderService(repo, slog.Default())

	order, err := svc.PlaceOrder(context.Background(), "Widget", 9.99)
	// assertions against `order` and `err` — no HTTP server, no real
	// repository, no network or disk I/O anywhere in this test at all
}
```

## Try It Yourself
- Swap `InMemoryOrderRepository` for a SQLite-backed one using Module 17's
  patterns — confirm `OrderService` and the HTTP layer need **zero** code
  changes, only `cmd/server/main.go`'s one constructor call
- Add a `readyz` check that actually pings something — even a fake
  `errors.New("simulated outage")` toggled by an env var — and watch
  `/readyz` return 503 while `/healthz` keeps returning 200 throughout
- Add a Prometheus **Histogram** tracking request duration per route (the
  guide's Metrics section), as HTTP middleware wrapping the router — a
  good exercise in Module 15's middleware pattern combined with this
  module's observability material
