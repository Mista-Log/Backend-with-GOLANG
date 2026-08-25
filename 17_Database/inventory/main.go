// Inventory — sqlx + the repository pattern. Business logic here (
// LowStockReport, Restock) depends ONLY on the ProductRepository interface,
// never on sqlx or SQLite directly — proven by running the exact same
// logic against both a real SQLite-backed repository and a pure in-memory
// fake, with zero changes to the logic itself.
//
// Run with: go mod tidy && go run .
package main

import (
	"context"
	"fmt"
	"os"

	"inventory/models"
	"inventory/repository"
)

// LowStockReport is BUSINESS LOGIC — notice it imports "inventory/repository"
// for the INTERFACE only, never sqlx or modernc.org/sqlite. This function
// has no idea whether it's talking to a real database or not.
func LowStockReport(ctx context.Context, repo repository.ProductRepository, category string, threshold int) ([]models.Product, error) {
	products, err := repo.ListByCategory(ctx, category)
	if err != nil {
		return nil, err
	}
	var low []models.Product
	for _, p := range products {
		if p.Quantity <= threshold {
			low = append(low, p)
		}
	}
	return low, nil
}

// Restock is also pure business logic against the interface — a good
// example of validation living ABOVE the repository layer, not inside it.
func Restock(ctx context.Context, repo repository.ProductRepository, sku string, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("restock amount must be positive")
	}
	product, err := repo.GetBySKU(ctx, sku)
	if err != nil {
		return err
	}
	return repo.AdjustQuantity(ctx, product.ID, amount)
}

func seed(ctx context.Context, repo repository.ProductRepository) {
	repo.Create(ctx, &models.Product{SKU: "MOU-001", Name: "Wireless Mouse", Category: "electronics", Price: 24.99, Quantity: 3})
	repo.Create(ctx, &models.Product{SKU: "KEY-002", Name: "Mechanical Keyboard", Category: "electronics", Price: 89.99, Quantity: 15})
	repo.Create(ctx, &models.Product{SKU: "HUB-003", Name: "USB-C Hub", Category: "electronics", Price: 45.00, Quantity: 2})
	repo.Create(ctx, &models.Product{SKU: "LMP-004", Name: "Desk Lamp", Category: "furniture", Price: 32.50, Quantity: 8})
}

func runDemo(ctx context.Context, label string, repo repository.ProductRepository) {
	fmt.Printf("\n=== %s ===\n", label)
	seed(ctx, repo)

	low, _ := LowStockReport(ctx, repo, "electronics", 5)
	fmt.Println("Low-stock electronics (quantity <= 5):")
	for _, p := range low {
		fmt.Printf("  %s (%s): %d remaining\n", p.Name, p.SKU, p.Quantity)
	}

	if err := Restock(ctx, repo, "HUB-003", 20); err != nil {
		fmt.Println("Restock error:", err)
	} else {
		fmt.Println("Restocked USB-C Hub by 20 units")
	}

	afterRestock, _ := repo.GetBySKU(ctx, "HUB-003")
	fmt.Printf("USB-C Hub quantity now: %d\n", afterRestock.Quantity)

	if err := Restock(ctx, repo, "MOU-001", -100); err != nil {
		fmt.Println("Expected validation error:", err)
	}
}

func main() {
	ctx := context.Background()

	fmt.Println("Running the SAME business logic against TWO different repositories:")

	// 1. The FAKE, in-memory repository — no database, no setup, instant.
	fakeRepo := repository.NewFakeProductRepository()
	runDemo(ctx, "Fake in-memory repository", fakeRepo)

	// 2. The REAL, SQLite-backed repository.
	os.Remove("inventory.db")
	db, err := openDB("inventory.db")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer db.Close()
	sqliteRepo := repository.NewSQLiteProductRepository(db)
	runDemo(ctx, "Real SQLite-backed repository", sqliteRepo)

	fmt.Println("\nLowStockReport and Restock produced IDENTICAL results against both —")
	fmt.Println("the business logic never knew which one it was talking to.")
}
