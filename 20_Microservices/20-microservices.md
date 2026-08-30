# 20. Microservices

The final module. Everything before this built one program, however
internally complex. Microservices is about **several independent
programs**, each owning its own piece of a system, talking to each other
over a network — with all the new failure modes, coordination challenges,
and tooling that implies.

---

## RPC

**Remote Procedure Call** — calling a function that actually runs on a
*different machine/process*, made to look and feel like an ordinary local
function call. Go's standard library includes a basic RPC implementation:

```go
// Server side:
type Args struct{ A, B int }

type Arith int

func (t *Arith) Multiply(args *Args, reply *int) error {
	*reply = args.A * args.B
	return nil
}

rpc.Register(new(Arith))
listener, _ := net.Listen("tcp", ":1234")
rpc.Accept(listener)

// Client side:
client, _ := rpc.Dial("tcp", "localhost:1234")
var reply int
client.Call("Arith.Multiply", &Args{A: 3, B: 4}, &reply)
fmt.Println(reply) // 12
```

```
┌──────────────────────────────────────────────────────────┐
│   Client code LOOKS like a local call:                             │
│      client.Call("Arith.Multiply", args, &reply)                       │
│                                                                            │
│   ...but under the hood:                                                    │
│      args are SERIALIZED  ──▶  sent over the network  ──▶  DESERIALIZED       │
│      on the server, the REAL method runs, the result is serialized               │
│      back, sent over the network, deserialized into `reply`                          │
│                                                                                            │
│   Every network call can now FAIL in ways a local call never could:                          │
│   the network could be down, the server could be slow or overloaded,                            │
│   the connection could drop mid-call — Module 13's context deadlines                               │
│   become mandatory here, not optional.                                                                │
└──────────────────────────────────────────────────────────┘
```

`net/rpc` is genuinely rare in modern production Go — it predates good
generic tooling and doesn't work across languages. **gRPC** (next section)
is what replaced it almost everywhere real RPC is still used today.

---

## gRPC

Google's RPC framework: define a service's methods and message shapes
**once**, in a language-agnostic `.proto` file, and generate strongly-typed
client/server code for Go, Python, Java, and more — from the exact same
definition.

```protobuf
// order.proto
syntax = "proto3";
package order;
option go_package = "myservice/proto/order";

service OrderService {
	rpc GetOrder (GetOrderRequest) returns (Order);
	rpc StreamOrderUpdates (GetOrderRequest) returns (stream Order); // SERVER STREAMING
}

message GetOrderRequest {
	int32 id = 1;
}

message Order {
	int32 id = 1;
	string item = 2;
	double amount = 3;
	string status = 4;
}
```

```bash
# Generates order.pb.go (message types) and order_grpc.pb.go (client/server code):
protoc --go_out=. --go-grpc_out=. order.proto
```

```go
// Server implementation — you write THIS; the interface it satisfies is generated:
type server struct {
	pb.UnimplementedOrderServiceServer
}

func (s *server) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.Order, error) {
	return &pb.Order{Id: req.Id, Item: "Keyboard", Amount: 89.99, Status: "shipped"}, nil
}

grpcServer := grpc.NewServer()
pb.RegisterOrderServiceServer(grpcServer, &server{})
lis, _ := net.Listen("tcp", ":50051")
grpcServer.Serve(lis)

// Client:
conn, _ := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
client := pb.NewOrderServiceClient(conn)
order, err := client.GetOrder(ctx, &pb.GetOrderRequest{Id: 42})
```

```
┌──────────────────────────────────────────────────────────┐
│               gRPC vs. a JSON REST API (Module 15/16)               │
│                                                                          │
│   REST/JSON:   human-readable, universally supported, easy to             │
│                  debug with curl — but no compile-time type safety           │
│                  across languages, and JSON is verbose on the wire              │
│                                                                                     │
│   gRPC/Protobuf: strongly-typed generated CLIENT code (no hand-                       │
│                    written HTTP calls at all), binary wire format is                    │
│                    much smaller/faster to (de)serialize, built-in                          │
│                    STREAMING (server, client, or bidirectional) — but                         │
│                    NOT human-readable on the wire, and needs the                                 │
│                    protoc toolchain as a build step                                                 │
│                                                                                                          │
│   Common real pattern: gRPC for internal SERVICE-TO-SERVICE calls                                          │
│   (where you control both ends and want speed + type safety), REST/                                          │
│   JSON for PUBLIC-FACING APIs (where broad client compatibility and                                              │
│   human-debuggability matter more)                                                                                  │
└──────────────────────────────────────────────────────────┘
```

**Note on this module's projects:** generating `.pb.go` files requires the
`protoc` compiler plus Go plugins installed locally — a heavier setup step
than anything earlier in this course (even more than the `go mod tidy`
steps in Modules 16-19). This module's runnable project includes a real,
correct `.proto` file and documents the exact `protoc` steps to generate
and wire it in, while the project's *running* demo communicates over a
real NATS server (embedded in-process, so nothing extra needs installing)
so the whole thing works out of the box — see the project's own README,
section 6, for the full gRPC path once you're ready to set up `protoc`
locally.

---

## Protocol Buffers

The `.proto` file itself, and the compact binary format it describes.
Every field gets a **number**, not just a name — this is what lets
Protobuf messages evolve over time without breaking old clients or servers:

```protobuf
message Order {
	int32 id = 1;      // field NUMBERS, not just names — this is the wire format's actual key
	string item = 2;
	double amount = 3;
	// adding a new field here, e.g. `string notes = 5;`, is SAFE —
	// old code simply ignores field 5 if it doesn't know about it yet
}
```

```
┌──────────────────────────────────────────────────────────┐
│   JSON:      {"id": 42, "item": "Keyboard", "amount": 89.99}          │
│                (human-readable, field NAMES sent every time)               │
│                                                                                 │
│   Protobuf:   [tag 1: varint 42] [tag 2: string "Keyboard"] [tag 3: ...]          │
│                 (binary, field NUMBERS instead of names — much smaller,             │
│                  but requires the .proto schema to decode meaningfully)                │
└──────────────────────────────────────────────────────────┘
```

**Never reuse or renumber a field number once a message has shipped** — an
old client reading field `2` as `item` would silently misinterpret data if
you later reused `2` for something else entirely. Removing a field should
mark its number `reserved`, never hand it to a new field.

---
## Service Discovery

In a system with many service instances, scaling up and down dynamically,
how does one service find *where* another currently lives? Hardcoding IP
addresses breaks the moment anything restarts or scales.

```
┌──────────────────────────────────────────────────────────┐
│   CLIENT-SIDE discovery:                                                │
│     service asks a REGISTRY ("where are healthy instances of                │
│     payment-service right now?") and picks one itself, often                  │
│     load-balancing across the results client-side                               │
│                                                                                      │
│   SERVER-SIDE discovery:                                                              │
│     service just calls a stable name/address; a LOAD BALANCER or                        │
│     PROXY in front does the lookup and routing transparently                               │
│                                                                                                 │
│   Real-world implementations:                                                                     │
│     - Kubernetes: a Service's stable DNS name resolves to whichever                                  │
│        healthy Pods currently back it — server-side discovery, built in                                │
│     - Consul / etcd: dedicated service registries services register                                       │
│        with on startup and deregister from on shutdown                                                       │
└──────────────────────────────────────────────────────────┘
```

```go
// A minimal client-side discovery client against a registry HTTP API:
resp, _ := http.Get("http://registry:8500/v1/health/service/payment-service?passing=true")
var instances []ServiceInstance
json.NewDecoder(resp.Body).Decode(&instances)
target := instances[rand.Intn(len(instances))] // pick one, e.g. round-robin in a real client
```

This module's projects sidestep real service discovery (a single, fixed
address per service is enough for a local demo) — but the **API Gateway**
below is exactly where server-side discovery/routing would plug in for a
system running at real scale.

---

## API Gateway

A single, public-facing entry point that routes requests to whichever
internal service actually handles them — clients only ever need to know
about the gateway's address, never the internal topology behind it.

```go
import "net/http/httputil"

orderProxy := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: "order-service:8081"})
paymentProxy := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: "payment-service:8082"})

mux := http.NewServeMux()
mux.Handle("/api/orders/", orderProxy)
mux.Handle("/api/payments/", paymentProxy)
```

```
┌──────────────────────────────────────────────────────────┐
│                                                                  │
│                          ┌──────────────┐                          │
│   Client ───────────────▶│  API GATEWAY   │                          │
│                          └──────┬───────┘                          │
│                    ┌────────────┼────────────┐                        │
│                    ▼            ▼            ▼                          │
│              order-service  payment-service  notification-service          │
│                                                                                 │
│   The client NEVER talks to order-service/payment-service/notification-      │
│   service directly — only the gateway. This is where you'd put CROSS-           │
│   CUTTING concerns that would otherwise need to be duplicated in EVERY             │
│   service: authentication (Module 18), rate limiting (Module 16),                    │
│   request logging, TLS termination, and request routing/load balancing.                 │
└──────────────────────────────────────────────────────────┘
```

This is Module 15/16's middleware chaining, applied at the scale of an
entire system instead of one service — the gateway's own middleware stack
(auth, rate limiting, logging) runs once, in front of everything, instead
of being re-implemented inside every individual service.

---

## Event Driven

Instead of Service A directly *calling* Service B (request/response, tight
coupling — A needs to know B exists and be able to reach it right now),
Service A publishes an **event** ("OrderCreated") without knowing or caring
who's listening; any number of other services can react independently.

```
┌──────────────────────────────────────────────────────────┐
│         Request/Response (tight coupling)                            │
│                                                                          │
│   Order Service ──calls──▶ Payment Service ──calls──▶ Notification Service│
│                                                                                │
│   If Notification Service is DOWN, does the whole chain fail? Does              │
│   Order Service need to know about Notification Service AT ALL?                    │
│   This coupling only gets worse as more services need to react to                     │
│   "an order was created."                                                                │
│                                                                                                │
│         Event-Driven (loose coupling)                                                           │
│                                                                                                     │
│   Order Service ──publishes──▶ "order.created" event ──▶ [a message broker]                          │
│                                          │                                                            │
│                        ┌─────────────────┼─────────────────┐                                            │
│                        ▼                 ▼                 ▼                                              │
│                 Payment Service   Notification Service   (any FUTURE                                         │
│                                                              service, added                                    │
│                                                              later, with ZERO                                    │
│                                                              changes to Order                                      │
│                                                              Service at all)                                          │
└──────────────────────────────────────────────────────────┘
```

**Choreography vs. orchestration** — two different ways to coordinate a
multi-step process across services:
```
┌──────────────────────────────────────────────────────────┐
│   CHOREOGRAPHY: each service reacts to events independently,           │
│     with NO central coordinator — Order Service doesn't know            │
│     Payment Service exists; it just publishes "order.created" and          │
│     moves on. This module's project uses this style.                          │
│                                                                                    │
│   ORCHESTRATION: a central coordinator explicitly calls each step           │
│     in sequence ("call Payment Service, THEN call Notification                   │
│     Service"), tracking overall progress itself. More visibility into               │
│     the whole flow in one place, but reintroduces some of the coupling                 │
│     choreography avoids.                                                                  │
└──────────────────────────────────────────────────────────┘
```

---

## Message Queue

The infrastructure piece that makes event-driven architecture actually
work reliably — a durable, ordered (or at least deliverable) channel
between publishers and consumers, decoupled in **time** as well as
identity (a consumer that's temporarily down doesn't lose messages
published while it was offline, unlike a direct HTTP call).

```
┌──────────────────────────────────────────────────────────┐
│                Delivery Guarantees — Know Which You Have                │
│                                                                              │
│   AT-MOST-ONCE:   a message might be LOST, but is never duplicated             │
│                     (fire-and-forget — fine for non-critical events,               │
│                      like a metrics ping)                                             │
│                                                                                            │
│   AT-LEAST-ONCE:   a message is NEVER lost, but might be delivered                           │
│                      MORE THAN ONCE (the consumer crashed after                                 │
│                      processing but before acknowledging — the broker                              │
│                      redelivers, not knowing it was already handled)                                  │
│                      → REQUIRES an IDEMPOTENT consumer (processing the                                   │
│                        same message twice must be safe) — this module's                                     │
│                        Payment Service is built this way deliberately                                            │
│                                                                                                                        │
│   EXACTLY-ONCE:     the hardest, most expensive guarantee — genuinely                                                    │
│                       exactly-once processing, typically requiring                                                            │
│                       transactional coordination between the broker and                                                          │
│                       consumer's own storage                                                                                          │
└──────────────────────────────────────────────────────────┘
```

**Idempotency** — the property that processing the same message twice has
the same effect as processing it once — is usually the practical answer to
"how do I get exactly-once BEHAVIOR out of an at-least-once GUARANTEE,"
without the cost of true exactly-once delivery:

```go
func (s *PaymentService) handleOrderCreated(event Event) {
	if s.alreadyProcessed(event.ID) { // check FIRST — Module 07's Cache.SetIfAbsent idea
		return // safe to receive this event again; already handled, do nothing
	}
	// ... process the payment ...
	s.markProcessed(event.ID)
}
```

**Queue vs. Topic (pub/sub)** — the other fundamental distinction:
```
┌────────────────────────────────────────────────────┐
│   QUEUE:   each message goes to EXACTLY ONE consumer among a               │
│              group (work distribution — e.g. a pool of workers                │
│              each pulling the next available job)                                │
│                                                                                       │
│   TOPIC (pub/sub):  each message goes to EVERY subscriber                              │
│                        independently (fan-out — e.g. "order.created"                        │
│                        reaching BOTH Payment Service AND an Analytics                          │
│                        service, each getting their own full copy)                                 │
└────────────────────────────────────────────────────┘
```

---

Onto the Tools section — the real message brokers this module's concepts
map to, with genuine Go client code for each — followed by the
Order/Payment/Notification project, built on a real, embedded NATS server
(via `nats-server`'s Go API) so the whole demo runs with nothing beyond
`go run` while still exercising the exact same client library (`nats.go`)
and pub/sub semantics a production deployment against a separately-run
NATS cluster would use.
## Tools

Every example below assumes a real broker instance is running (typically
via Docker locally: `docker run -p 9092:9092 apache/kafka`, etc.) — outside
the scope of what this course's sandboxed examples can run directly, but
the client code itself is real and directly usable once a broker is
reachable.

### Kafka

A distributed, durable, high-throughput log — messages are retained (not
deleted on consumption) for a configurable period, and consumers track
their own read position (**offset**), so multiple independent consumer
groups can each read the full history at their own pace.

```go
import "github.com/segmentio/kafka-go"

writer := &kafka.Writer{Addr: kafka.TCP("localhost:9092"), Topic: "order.created"}
writer.WriteMessages(context.Background(), kafka.Message{
	Key:   []byte("order-42"),
	Value: []byte(`{"orderID": 42, "amount": 89.99}`),
})

reader := kafka.NewReader(kafka.ReaderConfig{
	Brokers: []string{"localhost:9092"},
	Topic:   "order.created",
	GroupID: "payment-service", // consumer GROUP — enables the "each message
	                              // to exactly one consumer WITHIN this group" semantics
})
for {
	msg, _ := reader.ReadMessage(context.Background())
	fmt.Println(string(msg.Value))
}
```

Kafka's durability and replay-ability make it the common choice for event
sourcing, analytics pipelines, and any system where "replay everything
that happened since last Tuesday" is a real requirement.

### RabbitMQ

A traditional **message broker** implementing AMQP — built around
exchanges (routing rules) and queues (where messages actually wait),
excellent at complex routing patterns and traditional work-queue
distribution.

```go
import amqp "github.com/rabbitmq/amqp091-go"

conn, _ := amqp.Dial("amqp://guest:guest@localhost:5672/")
ch, _ := conn.Channel()
ch.QueueDeclare("payments", true, false, false, false, nil)

ch.Publish("", "payments", false, false, amqp.Publishing{
	ContentType: "application/json",
	Body:        []byte(`{"orderID": 42}`),
})

msgs, _ := ch.Consume("payments", "", true, false, false, false, nil)
for msg := range msgs {
	fmt.Println(string(msg.Body))
}
```

RabbitMQ is a common choice when you need flexible routing (send this
message type to these three queues, that type to just one) rather than
Kafka's simpler, replay-focused log model.

### NATS

A lightweight, extremely fast, "just messaging" broker — minimal
durability by default (though **JetStream**, its persistence layer, adds
Kafka-like durable streams when needed), popular specifically for its
simplicity and very low latency.

```go
import "github.com/nats-io/nats.go"

nc, _ := nats.Connect(nats.DefaultURL)

nc.Publish("order.created", []byte(`{"orderID": 42}`))

nc.Subscribe("order.created", func(msg *nats.Msg) {
	fmt.Println(string(msg.Data))
})
```

### Redis Streams

If a system already uses Redis (for caching, Module 7-adjacent needs), its
built-in **Streams** data type provides Kafka-like append-only log
semantics without adding a whole separate piece of infrastructure —
consumer groups, message IDs, and acknowledgment included.

```go
import "github.com/redis/go-redis/v9"

rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

rdb.XAdd(ctx, &redis.XAddArgs{
	Stream: "order.created",
	Values: map[string]any{"orderID": 42, "amount": 89.99},
})

entries, _ := rdb.XRead(ctx, &redis.XReadArgs{
	Streams: []string{"order.created", "$"},
	Block:   0,
}).Result()
```

### Choosing

```
┌──────────────────────────────────────────────────────────┐
│   Need durable replay, huge throughput, event sourcing?             │
│      → Kafka                                                              │
│                                                                                │
│   Need flexible routing, traditional task queues, mature                        │
│   ecosystem tooling?                                                              │
│      → RabbitMQ                                                                     │
│                                                                                          │
│   Need simplicity and very low latency, don't need heavy durability?                      │
│      → NATS (add JetStream if you DO need durability later)                                  │
│                                                                                                   │
│   Already running Redis, want messaging without new infrastructure?                                │
│      → Redis Streams                                                                                  │
│                                                                                                            │
│   Building a local demo/course project with zero external infra to           │
│   install, but still want a REAL broker and client library?                     │
│      → an embedded broker (this module's project embeds a real NATS               │
│         server in-process via nats-server's Go API)                                  │
└──────────────────────────────────────────────────────────┘
```

---

## Project — Order / Payment / Notification Services

Rather than requiring Docker and a separately-run broker just to follow
this course's final exercise, the accompanying **`microservices-demo/`**
project embeds a **real NATS server** directly inside Order Service's own
process (via `nats-server`'s Go API) — every other service connects to it
as an ordinary `nats.go` client, exactly as they would against a real,
separately-deployed NATS cluster. Every concept above (pub/sub,
choreography, idempotent consumption, service discovery, an API gateway in
front of independently-runnable services, and a real `.proto` gRPC
interface alongside the event-driven one) is exercised for real, against
real tooling — nothing is faked or hand-rolled. See its own README for the
full architecture and a complete multi-terminal walkthrough.

**Swapping the embedded NATS server for a separately-deployed one** (or for
Kafka, RabbitMQ, or Redis Streams instead) means changing only how each
service's `main.go` connects — `internal/events`'s `Publish`/`Subscribe`
functions and every service's business logic stay completely unchanged,
since they only ever call those two functions, never anything transport-
specific directly. This is Module 17's repository pattern and Module 19's
ports-and-adapters idea, one more time, now applied to messaging
infrastructure instead of a database.
