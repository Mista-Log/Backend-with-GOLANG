// Package repository holds ADAPTERS implementing the domain layer's ports.
// This one is in-memory — swap it for a real database-backed
// implementation (Module 17's patterns) with zero changes to the service
// or transport layers, since both depend only on domain.OrderRepository.
package repository

import (
	"context"
	"sync"

	"productionservice/internal/domain"
)

type InMemoryOrderRepository struct {
	mu     sync.Mutex
	orders map[int]*domain.Order
	nextID int
}

func NewInMemoryOrderRepository() *InMemoryOrderRepository {
	return &InMemoryOrderRepository{orders: make(map[int]*domain.Order), nextID: 1}
}

func (r *InMemoryOrderRepository) Create(ctx context.Context, o *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	o.ID = r.nextID
	r.orders[o.ID] = o
	r.nextID++
	return nil
}

func (r *InMemoryOrderRepository) Get(ctx context.Context, id int) (*domain.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	o, ok := r.orders[id]
	if !ok {
		return nil, domain.ErrOrderNotFound
	}
	return o, nil
}

func (r *InMemoryOrderRepository) Update(ctx context.Context, o *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.orders[o.ID]; !ok {
		return domain.ErrOrderNotFound
	}
	r.orders[o.ID] = o
	return nil
}

func (r *InMemoryOrderRepository) List(ctx context.Context) ([]*domain.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*domain.Order, 0, len(r.orders))
	for _, o := range r.orders {
		out = append(out, o)
	}
	return out, nil
}
