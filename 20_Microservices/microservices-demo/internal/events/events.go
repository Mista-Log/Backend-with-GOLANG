// Package events defines the event VOCABULARY shared by every service —
// the contract they all communicate through, without any service needing
// to import another service's package directly. This is what keeps them
// genuinely decoupled: Order Service doesn't know Payment Service exists;
// it only knows the shape of the events it publishes and subscribes to.
package events

import (
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

// Subjects — NATS's term for what Kafka calls "topics." Centralizing them
// here as constants means a typo in a subject name is a compile error
// (an unknown identifier), not a silent "nobody ever receives this" bug.
const (
	SubjectOrderCreated     = "order.created"
	SubjectPaymentCompleted = "payment.completed"
	SubjectPaymentFailed    = "payment.failed"
)

type OrderCreated struct {
	OrderID int     `json:"orderId"`
	Item    string  `json:"item"`
	Amount  float64 `json:"amount"`
}

type PaymentCompleted struct {
	OrderID       int    `json:"orderId"`
	TransactionID string `json:"transactionId"`
}

type PaymentFailed struct {
	OrderID int    `json:"orderId"`
	Reason  string `json:"reason"`
}

// Bus wraps a *nats.Conn with JSON encoding, so every service works with
// typed Go structs instead of raw []byte — the events themselves are
// serialized with encoding/json (Module 11), not protobuf, since the
// event bus and the gRPC API (proto/order.proto) are deliberately
// independent interfaces to the same underlying service.
type Bus struct {
	nc *nats.Conn
}

func NewBus(nc *nats.Conn) *Bus {
	return &Bus{nc: nc}
}

// Publish marshals v to JSON and publishes it on the given subject. Any
// type works here (Module 03's generics-adjacent `any`), since publishing
// doesn't need to know the payload's shape in advance — only subscribers do.
func Publish(bus *Bus, subject string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return bus.nc.Publish(subject, data)
}

// Subscribe is GENERIC (Module 07) over the event type T — each call site
// specifies exactly which event shape it expects, and the compiler checks
// that the handler function matches, with no type assertions or `any`
// anywhere in the calling code.
func Subscribe[T any](bus *Bus, subject string, handler func(T)) (*nats.Subscription, error) {
	return bus.nc.Subscribe(subject, func(msg *nats.Msg) {
		var event T
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("events: failed to decode message on %q: %v", subject, err)
			return
		}
		handler(event)
	})
}
