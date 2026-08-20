package bank

import (
	"encoding/json"
	"net/http"
)

type depositRequest struct {
	Account string  `json:"account"`
	Amount  float64 `json:"amount"`
}

type balanceResponse struct {
	Account string  `json:"account"`
	Balance float64 `json:"balance"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// NewHTTPHandler wraps a *Bank as an http.Handler — used by
// handler_test.go's integration test, and by the top-level main.go for
// manual exploration.
func NewHTTPHandler(b *Bank) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /deposit", func(w http.ResponseWriter, r *http.Request) {
		var req depositRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		acc, err := b.Account(req.Account)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		if err := acc.Deposit(req.Amount); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, balanceResponse{Account: acc.ID, Balance: acc.Balance()})
	})

	mux.HandleFunc("POST /withdraw", func(w http.ResponseWriter, r *http.Request) {
		var req depositRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		acc, err := b.Account(req.Account)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		if err := acc.Withdraw(req.Amount); err != nil {
			writeError(w, http.StatusUnprocessableEntity, err)
			return
		}
		writeJSON(w, http.StatusOK, balanceResponse{Account: acc.ID, Balance: acc.Balance()})
	})

	mux.HandleFunc("GET /balance", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("account")
		acc, err := b.Account(id)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, balanceResponse{Account: acc.ID, Balance: acc.Balance()})
	})

	return mux
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, errorResponse{Error: err.Error()})
}
