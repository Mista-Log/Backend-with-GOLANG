package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"inventory/models"
)

// FakeProductRepository satisfies the SAME ProductRepository interface as
// SQLiteProductRepository, but holds everything in memory — no database,
// no setup, instant. This is Module 14's mocking pattern applied to a real
// data layer: business logic written against ProductRepository can run
// against this in tests, and the real one in production, with ZERO code
// changes to the logic itself.
type FakeProductRepository struct {
	mu       sync.Mutex
	products map[int]*models.Product
	nextID   int
}

func NewFakeProductRepository() *FakeProductRepository {
	return &FakeProductRepository{products: make(map[int]*models.Product), nextID: 1}
}

func (r *FakeProductRepository) Create(ctx context.Context, p *models.Product) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.products {
		if existing.SKU == p.SKU {
			return 0, fmt.Errorf("UNIQUE constraint failed: products.sku") // mimics the real DB's error
		}
	}
	p.ID = r.nextID
	r.products[p.ID] = p
	r.nextID++
	return p.ID, nil
}

func (r *FakeProductRepository) GetByID(ctx context.Context, id int) (*models.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.products[id]
	if !ok {
		return nil, fmt.Errorf("no product with id %d", id)
	}
	return p, nil
}

func (r *FakeProductRepository) GetBySKU(ctx context.Context, sku string) (*models.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range r.products {
		if p.SKU == sku {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no product with sku %q", sku)
}

func (r *FakeProductRepository) ListByCategory(ctx context.Context, category string) ([]models.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []models.Product
	for _, p := range r.products {
		if p.Category == category {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (r *FakeProductRepository) AdjustQuantity(ctx context.Context, id int, delta int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.products[id]
	if !ok {
		return fmt.Errorf("no product with id %d", id)
	}
	if p.Quantity+delta < 0 {
		return fmt.Errorf("adjustment would make quantity negative")
	}
	p.Quantity += delta
	return nil
}

func (r *FakeProductRepository) Delete(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.products[id]; !ok {
		return fmt.Errorf("no product with id %d", id)
	}
	delete(r.products, id)
	return nil
}
