// cmd/orderservice is Order Service's binary. It also hosts an EMBEDDED
// NATS server (via nats-server's Go API) — in a real deployment, NATS
// would be its own separately-run process/cluster that every service
// connects to as a client; embedding it here purely means you don't need
// to separately install and run a NATS server to follow along. Every
// other service in this demo connects to it as an ordinary NATS client,
// exactly as they would against a real, separately-deployed broker.
//
// Run this SECOND, after cmd/gateway.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"

	"microservicesdemo/internal/events"
	"microservicesdemo/internal/orderservice"
)

const (
	natsPort    = 4222
	httpAddr    = "localhost:8081"
	gatewayAddr = "http://localhost:8080"
)

func startEmbeddedNATS() (*natsserver.Server, error) {
	opts := &natsserver.Options{Port: natsPort}
	srv, err := natsserver.NewServer(opts)
	if err != nil {
		return nil, err
	}
	go srv.Start()
	if !srv.ReadyForConnections(5 * time.Second) {
		return nil, fmt.Errorf("NATS server did not become ready in time")
	}
	return srv, nil
}

// registerWithGateway retries a few times, since the gateway or this
// service might start in either order in practice — a small taste of the
// resilience real service registration needs.
func registerWithGateway() {
	body, _ := json.Marshal(map[string]string{"service": "orders-service", "address": httpAddr})
	for attempt := 1; attempt <= 5; attempt++ {
		resp, err := http.Post(gatewayAddr+"/register", "application/json", bytes.NewReader(body))
		if err == nil && resp.StatusCode == http.StatusNoContent {
			log.Println("[order-service] registered with the API gateway")
			return
		}
		log.Printf("[order-service] gateway registration attempt %d failed, retrying...", attempt)
		time.Sleep(1 * time.Second)
	}
	log.Println("[order-service] WARNING: could not register with the gateway — is cmd/gateway running?")
}

func main() {
	natsSrv, err := startEmbeddedNATS()
	if err != nil {
		log.Fatal("failed to start embedded NATS server:", err)
	}
	defer natsSrv.Shutdown()
	fmt.Printf("Embedded NATS server listening on localhost:%d\n", natsPort)

	nc, err := nats.Connect(fmt.Sprintf("nats://localhost:%d", natsPort))
	if err != nil {
		log.Fatal("failed to connect to NATS:", err)
	}
	defer nc.Close()

	bus := events.NewBus(nc)
	svc := orderservice.New(bus)
	if err := svc.Start(); err != nil {
		log.Fatal("failed to start order service subscriptions:", err)
	}

	go registerWithGateway()

	fmt.Println("Order Service HTTP API listening on http://" + httpAddr)
	log.Fatal(http.ListenAndServe(httpAddr, orderservice.NewHTTPHandler(svc)))
}
