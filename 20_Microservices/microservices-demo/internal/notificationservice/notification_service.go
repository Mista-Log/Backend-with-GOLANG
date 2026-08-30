// Package notificationservice demonstrates the FAN-OUT benefit of
// event-driven architecture directly: it subscribes to the exact same
// order.created event Payment Service does, completely independently.
// Order Service was never modified to "also notify" — it just publishes
// one event, and however many services care can each react on their own.
package notificationservice

import (
	"fmt"
	"log"

	"microservicesdemo/internal/events"
)

type Service struct {
	bus *events.Bus
}

func New(bus *events.Bus) *Service {
	return &Service{bus: bus}
}

func (s *Service) Start() error {
	if _, err := events.Subscribe(s.bus, events.SubjectOrderCreated, s.notifyOrderCreated); err != nil {
		return err
	}
	if _, err := events.Subscribe(s.bus, events.SubjectPaymentCompleted, s.notifyPaymentCompleted); err != nil {
		return err
	}
	if _, err := events.Subscribe(s.bus, events.SubjectPaymentFailed, s.notifyPaymentFailed); err != nil {
		return err
	}
	return nil
}

// send stands in for actually calling an email/SMS provider (Module 06's
// Notification Service project covered that side in depth) — here it just
// prints, so the event flow itself stays the focus.
func (s *Service) send(message string) {
	fmt.Printf("  📧 [notification-service] %s\n", message)
}

func (s *Service) notifyOrderCreated(e events.OrderCreated) {
	log.Printf("[notification-service] received order #%d — sending confirmation", e.OrderID)
	s.send(fmt.Sprintf("Order #%d confirmed: %s ($%.2f). We'll email you when payment clears.", e.OrderID, e.Item, e.Amount))
}

func (s *Service) notifyPaymentCompleted(e events.PaymentCompleted) {
	log.Printf("[notification-service] received payment success for order #%d", e.OrderID)
	s.send(fmt.Sprintf("Payment confirmed for order #%d (transaction %s). Your order is on its way!", e.OrderID, e.TransactionID))
}

func (s *Service) notifyPaymentFailed(e events.PaymentFailed) {
	log.Printf("[notification-service] received payment failure for order #%d", e.OrderID)
	s.send(fmt.Sprintf("Payment failed for order #%d (%s). Please update your payment method.", e.OrderID, e.Reason))
}
