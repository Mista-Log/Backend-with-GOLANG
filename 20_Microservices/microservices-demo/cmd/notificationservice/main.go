// cmd/notificationservice is Notification Service's binary — like Payment
// Service, it has no HTTP server and no direct connection to any other
// service. It subscribes to THREE event types independently.
//
// Run this FOURTH (order relative to payment-service doesn't matter — both
// subscribe to order.created independently and neither knows the other exists).
package main

import (
	"fmt"
	"log"

	"github.com/nats-io/nats.go"

	"microservicesdemo/internal/events"
	"microservicesdemo/internal/notificationservice"
)

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal("failed to connect to NATS (is cmd/orderservice running?):", err)
	}
	defer nc.Close()

	bus := events.NewBus(nc)
	svc := notificationservice.New(bus)
	if err := svc.Start(); err != nil {
		log.Fatal("failed to start notification service subscriptions:", err)
	}

	fmt.Println("Notification Service running — subscribed to order.created, payment.completed, payment.failed")
	fmt.Println("Press Ctrl+C to stop.")
	select {}
}
