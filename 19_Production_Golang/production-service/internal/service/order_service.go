// Package service holds USE CASES — application-specific orchestration
// sitting one ring out from domain in the Clean Architecture diagram. It
// depends on domain.OrderRepository (the interface, injected in via the
// constructor — Dependency Injection, the guide's section), never on any
// concrete adapter.
package service

import (
	"context"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"

	"productionservice/internal/domain"
)

var ordersCreated = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "orders_created_total",
	Help: "Total number of orders created.",
})

func init() {
	prometheus.MustRegister(ordersCreated)
}

type OrderService struct {
	repo   domain.OrderRepository // the PORT — this field's type is an interface, never a concrete type
	logger *slog.Logger
}

// NewOrderService is CONSTRUCTOR-STYLE DEPENDENCY INJECTION — cmd/server's
// main.go decides which concrete OrderRepository to hand in; OrderService
// itself never constructs one.
func NewOrderService(repo domain.OrderRepository, logger *slog.Logger) *OrderService {
	return &OrderService{repo: repo, logger: logger}
}

func (s *OrderService) PlaceOrder(ctx context.Context, item string, amount float64) (*domain.Order, error) {
	order := &domain.Order{Item: item, Amount: amount, Status: "pending"}
	if err := s.repo.Create(ctx, order); err != nil {
		s.logger.Error("failed to create order", "error", err)
		return nil, err
	}
	ordersCreated.Inc() // Module 19's Metrics section — a real Counter, incremented on a real event
	s.logger.Info("order placed", "orderID", order.ID, "item", order.Item, "amount", order.Amount)
	return order, nil
}

func (s *OrderService) GetOrder(ctx context.Context, id int) (*domain.Order, error) {
	return s.repo.Get(ctx, id)
}

func (s *OrderService) ListOrders(ctx context.Context) ([]*domain.Order, error) {
	return s.repo.List(ctx)
}

func (s *OrderService) CancelOrder(ctx context.Context, id int) error {
	order, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := order.Cancel(); err != nil { // the business RULE lives on the entity itself (domain/order.go)
		s.logger.Warn("order cancellation rejected", "orderID", id, "reason", err)
		return err
	}
	if err := s.repo.Update(ctx, order); err != nil {
		return err
	}
	s.logger.Info("order cancelled", "orderID", id)
	return nil
}
