// Package saga implements an ORCHESTRATED saga (Module 20's terminology —
// one coordinator explicitly running each step and compensation) for a
// classic order flow: place order -> charge payment -> reserve stock.
// Each step is a LOCAL transaction against its own simulated service;
// there is no cross-service atomic commit, only the saga's guarantee that
// every step is either fully completed or fully compensated.
package saga

import (
	"context"
	"fmt"
)

// Step pairs a forward action with its compensation — the guide's Step
// type, made concrete for this specific order flow.
type Step struct {
	Name string
	Do   func(ctx context.Context) error
	Undo func(ctx context.Context) error
}

// Run executes every step in order. On the FIRST failure, it unwinds
// every PREVIOUSLY COMPLETED step's compensation, in REVERSE order —
// exactly defer's LIFO shape (Module 02), applied to compensating
// transactions instead of cleanup calls.
func Run(ctx context.Context, steps []Step) (log []string, err error) {
	var completed []Step
	for _, step := range steps {
		if doErr := step.Do(ctx); doErr != nil {
			log = append(log, fmt.Sprintf("FAILED: %s (%v)", step.Name, doErr))
			for i := len(completed) - 1; i >= 0; i-- {
				c := completed[i]
				if undoErr := c.Undo(ctx); undoErr != nil {
					log = append(log, fmt.Sprintf("COMPENSATION FAILED: %s (%v)", c.Name, undoErr))
					continue
				}
				log = append(log, fmt.Sprintf("compensated: %s", c.Name))
			}
			return log, fmt.Errorf("saga failed at %q, rolled back: %w", step.Name, doErr)
		}
		log = append(log, fmt.Sprintf("completed: %s", step.Name))
		completed = append(completed, step)
	}
	return log, nil
}

// --- A small simulated domain: Order, Payment, and Inventory services ---
// Each holds its OWN state, exactly like separate microservices (Module
// 20) would each own their own database — nothing here is shared state
// accessed directly; every interaction goes through a method call
// standing in for what would be a real network request.

type OrderService struct {
	orders map[string]string // orderID -> status
}

func NewOrderService() *OrderService { return &OrderService{orders: make(map[string]string)} }

func (s *OrderService) PlaceOrder(orderID string) error {
	s.orders[orderID] = "placed"
	return nil
}

func (s *OrderService) CancelOrder(orderID string) error {
	s.orders[orderID] = "cancelled"
	return nil
}

func (s *OrderService) Status(orderID string) string { return s.orders[orderID] }

type PaymentService struct {
	charges map[string]int64 // orderID -> amount charged
}

func NewPaymentService() *PaymentService { return &PaymentService{charges: make(map[string]int64)} }

func (s *PaymentService) Charge(orderID string, amount int64) error {
	s.charges[orderID] = amount
	return nil
}

func (s *PaymentService) Refund(orderID string) error {
	delete(s.charges, orderID)
	return nil
}

func (s *PaymentService) WasCharged(orderID string) bool {
	_, ok := s.charges[orderID]
	return ok
}

type InventoryService struct {
	stock     map[string]int
	reserved  map[string]int // orderID -> quantity reserved
}

func NewInventoryService(initialStock map[string]int) *InventoryService {
	return &InventoryService{stock: initialStock, reserved: make(map[string]int)}
}

func (s *InventoryService) Reserve(orderID, sku string, qty int) error {
	if s.stock[sku] < qty {
		return fmt.Errorf("insufficient stock for %s: have %d, need %d", sku, s.stock[sku], qty)
	}
	s.stock[sku] -= qty
	s.reserved[orderID] = qty
	return nil
}

func (s *InventoryService) Release(orderID, sku string) error {
	qty, ok := s.reserved[orderID]
	if !ok {
		return nil // nothing to release — this step never actually completed
	}
	s.stock[sku] += qty
	delete(s.reserved, orderID)
	return nil
}

// BuildOrderSaga wires the three services above into the guide's
// place-order/charge-payment/reserve-stock saga, ready to hand to Run.
func BuildOrderSaga(orders *OrderService, payments *PaymentService, inventory *InventoryService, orderID, sku string, qty int, amount int64) []Step {
	return []Step{
		{
			Name: "place order",
			Do:   func(ctx context.Context) error { return orders.PlaceOrder(orderID) },
			Undo: func(ctx context.Context) error { return orders.CancelOrder(orderID) },
		},
		{
			Name: "charge payment",
			Do:   func(ctx context.Context) error { return payments.Charge(orderID, amount) },
			Undo: func(ctx context.Context) error { return payments.Refund(orderID) },
		},
		{
			Name: "reserve stock",
			Do:   func(ctx context.Context) error { return inventory.Reserve(orderID, sku, qty) },
			Undo: func(ctx context.Context) error { return inventory.Release(orderID, sku) },
		},
	}
}
