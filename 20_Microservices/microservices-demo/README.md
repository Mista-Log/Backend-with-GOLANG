# Project — Order / Payment / Notification Services

Three independently-runnable microservices communicating entirely through
events over a real NATS server, fronted by a working API Gateway with
self-registering service discovery — plus a genuine `.proto`-defined gRPC
interface as the synchronous alternative.

```
microservices-demo/
├── go.mod
├── proto/
│   └── order.proto              real Protobuf/gRPC definition (see section 6)
├── cmd/
│   ├── gateway/main.go             API Gateway — run 1st
│   ├── orderservice/main.go          Order Service + embedded NATS — run 2nd
│   ├── paymentservice/main.go          Payment Service — run 3rd
│   └── notificationservice/main.go      Notification Service — run 4th
└── internal/
    ├── events/                    the shared event VOCABULARY (the only
    │                                 thing these services depend on together)
    ├── orderservice/
    ├── paymentservice/
    ├── notificationservice/
    ├── registry/                  Service Discovery
    └── gateway/                     API Gateway
```

## Setup

```bash
cd microservices-demo
go mod tidy    # fetches nats.go and the embedded nats-server — needs internet access
```

## Running It — Four Terminals

```bash
# Terminal 1
go run ./cmd/gateway

# Terminal 2 (starts the embedded NATS server too)
go run ./cmd/orderservice

# Terminal 3
go run ./cmd/paymentservice

# Terminal 4
go run ./cmd/notificationservice
```

```
┌──────────────────────────────────────────────────────────────┐
│   Terminal 1: Gateway starts, listens on :8080, waits for registrations  │
│                                                                              │
│   Terminal 2: orderservice starts an EMBEDDED NATS server on :4222,          │
│                connects to it as a client, starts its own HTTP API on           │
│                :8081, and registers "orders-service" -> "localhost:8081"           │
│                with the gateway (retrying a few times if the gateway                  │
│                isn't up yet)                                                             │
│                                                                                              │
│   Terminal 3: paymentservice connects to NATS at localhost:4222 as an                         │
│                ordinary CLIENT (it has no idea orderservice is the one                           │
│                hosting the broker — in a real deployment NATS would be its                          │
│                OWN separate process/cluster) and subscribes to order.created                            │
│                                                                                                              │
│   Terminal 4: notificationservice connects the same way, subscribing to                                        │
│                order.created, payment.completed, AND payment.failed                                                │
└──────────────────────────────────────────────────────────────┘
```

## Placing an Order — Through the Gateway

```bash
curl -X POST http://localhost:8080/api/orders \
  -d '{"item": "Mechanical Keyboard", "amount": 89.99}'
```

Watch **all four terminals** as this runs:

```
┌──────────────────────────────────────────────────────────────┐
│   Terminal 1 (gateway):                                                 │
│     [gateway] proxying POST /api/orders -> localhost:8081/orders           │
│                                                                                │
│   Terminal 2 (order-service):                                                    │
│     [order-service] order #1 placed (Mechanical Keyboard, $89.99) —                 │
│       publishing order.created                                                          │
│                                                                                              │
│   Terminal 3 (payment-service), ~300ms later:                                                  │
│     [payment-service] received order #1 ($89.99) — processing payment...                          │
│     [payment-service] order #1 payment SUCCEEDED (TX-48213) —                                        │
│       publishing payment.completed                                                                      │
│     (roughly 1 in 5 runs instead: payment FAILED — publishing payment.failed)                              │
│                                                                                                                 │
│   Terminal 4 (notification-service), reacting to BOTH events independently:                                      │
│     [notification-service] received order #1 — sending confirmation                                                  │
│       📧 Order #1 confirmed: Mechanical Keyboard ($89.99). We'll email                                                  │
│          you when payment clears.                                                                                          │
│     [notification-service] received payment success for order #1                                                              │
│       📧 Payment confirmed for order #1 (transaction TX-48213). Your                                                             │
│          order is on its way!                                                                                                       │
│                                                                                                                                          │
│   Terminal 2 (order-service), reacting to the SAME payment event:                                                                          │
│     [order-service] order #1 marked PAID (transaction TX-48213)                                                                                │
└──────────────────────────────────────────────────────────────┘
```

Then confirm the final state landed correctly, again through the gateway:
```bash
curl http://localhost:8080/api/orders/1
# {"id":1,"item":"Mechanical Keyboard","amount":89.99,"status":"paid"}
```

## 1. The Full Event Flow, as One Diagram

```
┌──────────────────────────────────────────────────────────────┐
│                                                                      │
│   curl ──▶ Gateway ──▶ Order Service                                   │
│                              │                                            │
│                              │  publish "order.created"                     │
│                              ▼                                                │
│                    ┌─────────────────────┐                                     │
│                    │   NATS (embedded)      │                                     │
│                    └─────────────────────┘                                     │
│                         │              │                                          │
│              subscribed │              │ subscribed                                 │
│                         ▼              ▼                                             │
│                Payment Service   Notification Service                                   │
│                         │              │                                                   │
│           publish "payment.*"          │ (reacts immediately — order confirmation)             │
│                         │                                                                        │
│                         ▼                                                                           │
│              ┌─────────────────────┐                                                                   │
│              │   NATS (embedded)      │                                                                   │
│              └─────────────────────┘                                                                   │
│                   │            │                                                                          │
│        subscribed │            │ subscribed                                                                  │
│                   ▼            ▼                                                                             │
│         Order Service   Notification Service                                                                    │
│         (marks paid/       (payment confirmation/                                                                   │
│          failed)             failure notice)                                                                          │
│                                                                                                                          │
│   NEITHER Payment Service NOR Notification Service ever imports                                                           │
│   Order Service's package, calls its HTTP API, or knows its address —                                                        │
│   the ONLY shared dependency across all three is internal/events.                                                              │
└──────────────────────────────────────────────────────────────┘
```

## 2. Why This Proves Loose Coupling (Not Just Asserts It)

Kill Payment Service (Ctrl+C in Terminal 3) and place another order:
```bash
curl -X POST http://localhost:8080/api/orders -d '{"item": "Webcam", "amount": 45.00}'
curl http://localhost:8080/api/orders/2
# {"id":2,"item":"Webcam","amount":45,"status":"pending"}   ◀── stuck pending — no payment
#                                                                  processor is running to pick it up,
#                                                                  but Order Service and Notification
#                                                                  Service are COMPLETELY unaffected —
#                                                                  the order was created and the
#                                                                  confirmation email still "sent" fine
```
Restart Payment Service — it picks up NATS subscriptions fresh but has no
memory of the order that arrived while it was down (core NATS is
at-most-once, per the guide's Message Queue section — a real system
needing to handle this would reach for JetStream's persistence, or have
Order Service periodically re-publish unresolved pending orders).

## 3. Service Discovery, Demonstrated

```bash
# Try the gateway BEFORE order-service has registered (restart order-service
# and immediately, within a second or two, try this):
curl -i http://localhost:8080/api/orders
# 502 Bad Gateway: service unavailable: no instances of "orders-service" registered
```
```
┌──────────────────────────────────────────────────────────┐
│   registry.Registry starts EMPTY                                    │
│        │                                                                │
│        ▼                                                                  │
│   order-service's registerWithGateway() POSTs {"service":"orders-service",   │
│     "address":"localhost:8081"} — RETRIES up to 5 times, 1s apart,               │
│     specifically to survive the gateway or order-service starting FIRST             │
│        │                                                                                │
│        ▼                                                                                   │
│   registry.Register("orders-service", "localhost:8081")                                       │
│        │                                                                                          │
│        ▼                                                                                             │
│   subsequent /api/orders/* requests succeed — reg.Discover("orders-service")                            │
│     now returns "localhost:8081"                                                                           │
└──────────────────────────────────────────────────────────┘
```

## 4. What a Second Order Service Instance Would Look Like

The registry already supports multiple instances per service name — try
running a **second** `orderservice` on a different port:
```bash
PORT=8082 go run ./cmd/orderservice   # (see "Try It Yourself" for the PORT env var change needed)
```
Both would register under `"orders-service"`, and `registry.Discover`'s
random selection would load-balance across them — the exact client-side
discovery pattern from the guide, now with two real, independently running
processes behind one gateway address.

## 5. Idempotency — Why Order IDs Matter for Safety

Notice `handlePaymentCompleted`/`handlePaymentFailed` look up the order by
ID and simply *set* its status, rather than, say, incrementing a counter.
If NATS ever redelivered a `payment.completed` message twice (a real
possibility under at-least-once delivery, per the guide), processing it
twice would set the same status twice — completely harmless. This is the
guide's idempotency principle, already built into this design rather than
bolted on afterward.

## 6. The gRPC / Protobuf Side

`proto/order.proto` defines a real, correct gRPC interface to Order
Service's same underlying data — a synchronous alternative to (not a
replacement for) its REST API and event interface. It's not wired into the
running demo because generating Go code from it needs the `protoc`
compiler plus two plugins installed locally:

```bash
# One-time local setup (NOT covered by go mod tidy):
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
# protoc itself: install via your OS package manager (apt/brew/etc.)

protoc --go_out=. --go-grpc_out=. proto/order.proto
# generates proto/orderpb/order.pb.go and order_grpc.pb.go
```

Once generated, a server implementation would look like this (illustrative
— this code is NOT included as a compilable file in this project, since it
depends on the generated package that doesn't exist until you run `protoc`
yourself):

```go
type grpcOrderServer struct {
	orderpb.UnimplementedOrderServiceServer
	svc *orderservice.Service
}

func (s *grpcOrderServer) GetOrder(ctx context.Context, req *orderpb.GetOrderRequest) (*orderpb.Order, error) {
	order, ok := s.svc.Get(int(req.Id))
	if !ok {
		return nil, status.Error(codes.NotFound, "order not found")
	}
	return &orderpb.Order{Id: int32(order.ID), Item: order.Item, Amount: order.Amount, Status: order.Status}, nil
}
```

Notice this would call the **exact same** `orderservice.Service` that both
the REST handler and the event subscriptions already use — gRPC here is
just a third *interface* to the same core logic, not a separate
implementation of it. This is Clean Architecture (Module 19) directly: the
`orderservice` package is the inner ring; REST, events, and gRPC are three
different outer-ring adapters around the identical core.

## Try It Yourself
- Make `orderservice`'s HTTP port configurable via a `PORT` environment
  variable (Module 19's config patterns) so you can actually run a second
  instance for section 4's load-balancing demo
- Add a fourth event, `order.shipped`, published by a brand new (very
  small) Shipping Service — and confirm Notification Service can subscribe
  to it too with just a few added lines, without Order Service or Payment
  Service needing ANY changes at all
- Generate the gRPC code for real (section 6) and add a `StreamOrderUpdates`
  implementation that subscribes internally to `payment.completed`/
  `payment.failed` for the requested order ID and streams an updated
  `Order` to the gRPC client each time — bridging the event-driven world
  back into a synchronous streaming RPC
