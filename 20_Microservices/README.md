# Go for Beginners — Module 20: Microservices

## Contents

1. **[20-microservices.md](./20-microservices.md)** — RPC and Go's own
   `net/rpc`, gRPC and Protocol Buffers (real `.proto` syntax, code
   generation, and why field *numbers* — not names — make the wire format
   evolvable), service discovery (client-side vs. server-side), API
   gateways, event-driven architecture (choreography vs. orchestration),
   and message queues (delivery guarantees, and why idempotent consumers
   are usually the practical answer to "exactly-once behavior" without
   true exactly-once delivery) — then a **Tools** section covering Kafka,
   RabbitMQ, NATS, and Redis Streams with real client code for each.
   Diagrams throughout every section.

2. **[microservices-demo/](./microservices-demo)** — Three independently
   runnable services (Order, Payment, Notification) communicating entirely
   through events over a **real, embedded NATS server** — no Docker, no
   separately-installed broker, just `go run`. Fronted by a working API
   Gateway with self-registering service discovery, plus a genuine
   `.proto`-defined gRPC interface documented alongside the event-driven
   one. The README proves loose coupling by *killing* Payment Service
   mid-demo and showing the other two are completely unaffected, walks
   through service discovery failing before registration and succeeding
   after, and explains exactly where this design's idempotency guarantee
   already lives.

## Suggested Order

```
Microservices guide ──▶ microservices-demo (Order/Payment/Notification)
```

One project spanning three services plus a gateway — microservices
concepts don't really separate into independent exercises the way earlier
modules' topics could; the whole point is several pieces interacting.

## Setup — Four Terminals, Real Infrastructure

```bash
cd microservices-demo
go mod tidy    # fetches nats.go and the embedded nats-server — needs internet access

# Terminal 1
go run ./cmd/gateway
# Terminal 2 (also starts the embedded NATS server)
go run ./cmd/orderservice
# Terminal 3
go run ./cmd/paymentservice
# Terminal 4
go run ./cmd/notificationservice
```

Then, from a fifth terminal:
```bash
curl -X POST http://localhost:8080/api/orders -d '{"item": "Keyboard", "amount": 89.99}'
```
and watch all four terminals — see the project's own README for the full
walkthrough, including what happens when you kill a service mid-flow.

*Note: this module builds on nearly everything before it — Module 06
(interfaces, gRPC's generated server interfaces), Module 07 (generics, the
event bus's `Subscribe[T any]`), Module 11 (serialization — events are
JSON, `.proto` messages are Protobuf, deliberately different formats for
different interfaces to the same service), Module 12 (concurrency — every
subscription callback runs concurrently), Module 15 (HTTP — the gateway's
reverse proxy), Module 17 (the repository pattern this module's messaging
abstraction mirrors), and Module 19 (project structure and ports-and-
adapters, which this project's `internal/` layout and event-vs-REST-vs-gRPC
interfaces apply directly). This is the last module — if a piece feels
unfamiliar, that earlier module is exactly where to look.*
