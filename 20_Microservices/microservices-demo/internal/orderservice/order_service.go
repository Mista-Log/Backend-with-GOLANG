// Package orderservice owns the Order domain: creating orders, and
// updating their status in reaction to payment events published by a
// completely separate service it never directly imports or calls.
package orderservice

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"microservicesdemo/internal/events"
)

type Order struct {
	ID     int     `json:"id"`
	Item   string  `json:"item"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"`
}

type Service struct {
	mu     sync.Mutex
	orders map[int]*Order
	nextID int
	bus    *events.Bus
}

func New(bus *events.Bus) *Service {
	return &Service{orders: make(map[int]*Order), nextID: 1, bus: bus}
}

// Start subscribes to the events this service reacts to. Note it NEVER
// subscribes to anything about HOW payment was processed — only the
// OUTCOME (completed/failed), which is all Order Service actually needs
// to know to do its own job.
func (s *Service) Start() error {
	if _, err := events.Subscribe(s.bus, events.SubjectPaymentCompleted, s.handlePaymentCompleted); err != nil {
		return err
	}
	if _, err := events.Subscribe(s.bus, events.SubjectPaymentFailed, s.handlePaymentFailed); err != nil {
		return err
	}
	return nil
}

func (s *Service) PlaceOrder(item string, amount float64) (*Order, error) {
	if item == "" || amount <= 0 {
		return nil, fmt.Errorf("item is required and amount must be positive")
	}

	s.mu.Lock()
	order := &Order{ID: s.nextID, Item: item, Amount: amount, Status: "pending"}
	s.orders[order.ID] = order
	s.nextID++
	s.mu.Unlock()

	log.Printf("[order-service] order #%d placed (%s, $%.2f) — publishing %s",
		order.ID, order.Item, order.Amount, events.SubjectOrderCreated)

	// Publish and MOVE ON — Order Service does not wait for payment to be
	// processed. This is the guide's event-driven diagram, made real: the
	// HTTP handler returns "pending" to the caller immediately.
	if err := events.Publish(s.bus, events.SubjectOrderCreated, events.OrderCreated{
		OrderID: order.ID, Item: order.Item, Amount: order.Amount,
	}); err != nil {
		log.Printf("[order-service] failed to publish OrderCreated: %v", err)
	}

	return order, nil
}

func (s *Service) handlePaymentCompleted(e events.PaymentCompleted) {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.orders[e.OrderID]
	if !ok {
		log.Printf("[order-service] payment completed for unknown order #%d", e.OrderID)
		return
	}
	order.Status = "paid"
	log.Printf("[order-service] order #%d marked PAID (transaction %s)", order.ID, e.TransactionID)
}

func (s *Service) handlePaymentFailed(e events.PaymentFailed) {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.orders[e.OrderID]
	if !ok {
		log.Printf("[order-service] payment failed for unknown order #%d", e.OrderID)
		return
	}
	order.Status = "payment_failed"
	log.Printf("[order-service] order #%d marked PAYMENT_FAILED (%s)", order.ID, e.Reason)
}

func (s *Service) Get(id int) (*Order, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	return o, ok
}

func (s *Service) List() []*Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Order, 0, len(s.orders))
	for _, o := range s.orders {
		out = append(out, o)
	}
	return out
}

// --- HTTP API -----------------------------------------------------

func NewHTTPHandler(svc *Service) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Item   string  `json:"item"`
			Amount float64 `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		order, err := svc.PlaceOrder(req.Item, req.Amount)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, order)
	})

	mux.HandleFunc("GET /orders", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, svc.List())
	})

	mux.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid order id"})
			return
		}
		order, ok := svc.Get(id)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "order not found"})
			return
		}
		writeJSON(w, http.StatusOK, order)
	})

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return mux
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
