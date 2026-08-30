// cmd/gateway is the API Gateway — the single external entry point.
// Run this FIRST, before the other services, so they have something to
// register with on startup.
package main

import (
	"fmt"
	"log"
	"net/http"

	"microservicesdemo/internal/gateway"
	"microservicesdemo/internal/registry"
)

func main() {
	reg := registry.New()
	handler := gateway.NewHandler(reg)

	fmt.Println("API Gateway listening on http://localhost:8080")
	fmt.Println("Waiting for services to register themselves...")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
