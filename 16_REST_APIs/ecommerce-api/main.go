// E-commerce API — built with Chi, covering filtering, sorting, search,
// per-IP rate limiting, TTL-based caching, and structured logging. Chi
// handlers are plain net/http functions (Module 15's Handler model),
// contrasted deliberately against the Todo API project's Gin-owned Context.
//
// Run with: go mod tidy && go run .
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
)

func main() {
	store := NewProductStore()
	store.Create(Product{Name: "Wireless Mouse", Description: "Ergonomic wireless mouse", Category: "electronics", Price: 24.99, InStock: true})
	store.Create(Product{Name: "Mechanical Keyboard", Description: "RGB backlit mechanical keyboard", Category: "electronics", Price: 89.99, InStock: true})
	store.Create(Product{Name: "Standing Desk", Description: "Adjustable height standing desk", Category: "furniture", Price: 349.00, InStock: false})
	store.Create(Product{Name: "Desk Lamp", Description: "LED desk lamp with USB charging", Category: "furniture", Price: 32.50, InStock: true})
	store.Create(Product{Name: "USB-C Hub", Description: "7-in-1 USB-C hub with HDMI", Category: "electronics", Price: 45.00, InStock: true})

	cache := NewQueryCache(5 * time.Second) // short TTL — see cache.go's comment on invalidation
	limiter := NewIPRateLimiter(5, 10)      // 5 req/s sustained, burst of 10 — the guide's numbers
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	r := chi.NewRouter()
	r.Use(loggingMiddleware(logger))
	r.Use(rateLimitMiddleware(limiter))

	r.Get("/products", listProductsHandler(store, cache))
	r.Get("/products/search", searchProductsHandler(store))
	r.Get("/products/{id}", getProductHandler(store))
	r.Post("/products", createProductHandler(store, cache))
	r.Put("/products/{id}", updateProductHandler(store, cache))
	r.Delete("/products/{id}", deleteProductHandler(store, cache))

	fmt.Println("E-commerce API (Chi) listening on http://localhost:8080")
	fmt.Println("See README.md for a full curl walkthrough.")
	http.ListenAndServe(":8080", r)
}
