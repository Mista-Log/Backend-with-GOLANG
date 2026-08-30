// cmd/paymentservice is Payment Service's binary. Notice it has NO HTTP
// server at all — it never receives direct calls from anyone. Its entire
// interface to the rest of the system is the event bus: subscribe to
// order.created, publish payment.completed or payment.failed.
//
// Run this THIRD, after cmd/gateway and cmd/orderservice (which hosts the
// embedded NATS server this connects to).
package main

import (
	"fmt"
	"log"

	"github.com/nats-io/nats.go"

	"microservicesdemo/internal/events"
	"microservicesdemo/internal/paymentservice"
)

func main() {
	nc, err := nats.Connect(nats.DefaultURL) // nats.DefaultURL == "nats://localhost:4222"
	if err != nil {
		log.Fatal("failed to connect to NATS (is cmd/orderservice running?):", err)
	}
	defer nc.Close()

	bus := events.NewBus(nc)
	svc := paymentservice.New(bus)
	if err := svc.Start(); err != nil {
		log.Fatal("failed to start payment service subscriptions:", err)
	}

	fmt.Println("Payment Service running — subscribed to order.created")
	fmt.Println("Press Ctrl+C to stop.")
	select {} // block forever — all real work happens in the subscription callback
}
