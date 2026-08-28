// Package domain holds the CORE business types and rules — the innermost
// ring of the guide's Clean/Hexagonal Architecture diagrams. Notice what
// this file does NOT import: no net/http, no database driver, no
// prometheus, nothing framework-shaped at all. That's the entire point —
// dependencies point INWARD, and this is the center everything else
// depends on, never the reverse.
package domain

import (
	"context"
	"errors"
)

var ErrOrderNotFound = errors.New("order not found")

// Order is an ENTITY (Module 19's DDD section): it has a persistent
// identity (ID) that survives changes to its other fields.
type Order struct {
	ID     int
	Item   string
	Amount float64
	Status string
}

// Cancel is a METHOD ENFORCING A BUSINESS RULE directly on the entity —
// the DDD habit of keeping rules inside the domain type itself, not
// scattered across HTTP handlers or SQL queries.
func (o *Order) Cancel() error {
	if o.Status == "shipped" {
		return errors.New("cannot cancel an order that has already shipped")
	}
	o.Status = "cancelled"
	return nil
}

// OrderRepository is a PORT (Hexagonal Architecture's term) — the domain
// layer DEFINES this interface (it knows what it needs), but never
// implements it. Concrete ADAPTERS (internal/repository) implement it,
// depending inward on this package — never the other way around.
type OrderRepository interface {
	Create(ctx context.Context, o *Order) error
	Get(ctx context.Context, id int) (*Order, error)
	Update(ctx context.Context, o *Order) error
	List(ctx context.Context) ([]*Order, error)
}
