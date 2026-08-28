// Package http holds the OUTERMOST ring — framework-facing adapters
// translating HTTP requests into calls on the service layer, and service
// results back into HTTP responses. It depends INWARD on service and
// domain; nothing in those inner layers ever imports this package.
package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"productionservice/internal/domain"
	"productionservice/internal/service"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type createOrderRequest struct {
	Item   string  `json:"item"`
	Amount float64 `json:"amount"`
}

func NewRouter(orders *service.OrderService, ready func() error) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var req createOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		order, err := orders.PlaceOrder(r.Context(), req.Item, req.Amount)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to place order")
			return
		}
		writeJSON(w, http.StatusCreated, order)
	})

	mux.HandleFunc("GET /orders", func(w http.ResponseWriter, r *http.Request) {
		list, _ := orders.ListOrders(r.Context())
		writeJSON(w, http.StatusOK, list)
	})

	mux.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid order id")
			return
		}
		order, err := orders.GetOrder(r.Context(), id)
		if errors.Is(err, domain.ErrOrderNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to fetch order")
			return
		}
		writeJSON(w, http.StatusOK, order)
	})

	mux.HandleFunc("POST /orders/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid order id")
			return
		}
		if err := orders.CancelOrder(r.Context(), id); err != nil {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
	})

	// LIVENESS: "is the process itself alive?" — no dependency checks at all.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// READINESS: "is it ready for real traffic?" — checks the `ready` func
	// the caller supplied, which in a real service would ping a database,
	// a cache, etc. Kept as an injected function here so this package
	// still doesn't need to import any concrete dependency.
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := ready(); err != nil {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	return mux
}
