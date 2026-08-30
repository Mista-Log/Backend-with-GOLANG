// Package paymentservice knows nothing about HTTP, and nothing about
// Order Service's internal types or database — it only knows the
// events.OrderCreated shape it subscribes to, and the two event shapes
// it's allowed to publish back. This is the whole point of an event-driven
// boundary: swap Order Service's entire implementation, or replace it with
// a different language entirely, and Payment Service needs zero changes,
// as long as the event contract stays the same.
package paymentservice

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"microservicesdemo/internal/events"
)

type Service struct {
	bus *events.Bus
}

func New(bus *events.Bus) *Service {
	return &Service{bus: bus}
}

func (s *Service) Start() error {
	_, err := events.Subscribe(s.bus, events.SubjectOrderCreated, s.handleOrderCreated)
	return err
}

// handleOrderCreated simulates calling out to a real payment processor
// (Module 06's Payment Gateway project, conceptually) — a short delay,
// then a randomized outcome, purely so both the success and failure event
// paths are visible when you run this demo.
func (s *Service) handleOrderCreated(e events.OrderCreated) {
	log.Printf("[payment-service] received order #%d ($%.2f) — processing payment...", e.OrderID, e.Amount)
	time.Sleep(300 * time.Millisecond) // simulated processing latency

	if rand.Intn(100) < 80 { // 80% success rate, so failures are visible but not dominant
		txID := fmt.Sprintf("TX-%05d", rand.Intn(100000))
		log.Printf("[payment-service] order #%d payment SUCCEEDED (%s) — publishing %s",
			e.OrderID, txID, events.SubjectPaymentCompleted)
		events.Publish(s.bus, events.SubjectPaymentCompleted, events.PaymentCompleted{
			OrderID: e.OrderID, TransactionID: txID,
		})
		return
	}

	reason := "card declined"
	log.Printf("[payment-service] order #%d payment FAILED (%s) — publishing %s",
		e.OrderID, reason, events.SubjectPaymentFailed)
	events.Publish(s.bus, events.SubjectPaymentFailed, events.PaymentFailed{
		OrderID: e.OrderID, Reason: reason,
	})
}
