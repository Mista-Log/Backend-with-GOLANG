// Package bankapi simulates an external bank/card network — the "outside
// world" a real Payment Gateway talks to. It runs as a genuine, separate
// HTTP server (via httptest, the same technique Module 18 used for a fake
// OAuth provider), and processes deposits ASYNCHRONOUSLY, calling back to
// a webhook URL once "settled" — exactly matching the guide's Payment
// Gateway diagram, for real.
package bankapi

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"
)

type DepositRequest struct {
	Reference   string `json:"reference"`   // our system's own ID for this request
	Amount      int64  `json:"amount"`      // minor units
	Currency    string `json:"currency"`
	WebhookURL  string `json:"webhookUrl"`  // where the bank calls back with the result
}

type WebhookPayload struct {
	EventType      string `json:"eventType"` // "deposit.settled" or "deposit.failed"
	Reference      string `json:"reference"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	IdempotencyKey string `json:"idempotencyKey"`
}

// Server is the fake bank itself.
type Server struct {
	*httptest.Server
}

func NewServer() *Server {
	mux := http.NewServeMux()
	s := &Server{}

	mux.HandleFunc("POST /deposits", func(w http.ResponseWriter, r *http.Request) {
		var req DepositRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		// Respond IMMEDIATELY — a real bank rail does NOT settle a
		// transfer synchronously within one HTTP request. The actual
		// result arrives later, via the webhook below.
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"status": "processing"})

		// Simulate real-world settlement latency, then call back — this
		// goroutine outlives the original HTTP request entirely, exactly
		// like a real bank's own backend processing would.
		go func(req DepositRequest) {
			time.Sleep(150 * time.Millisecond)

			payload := WebhookPayload{
				EventType:      "deposit.settled",
				Reference:      req.Reference,
				Amount:         req.Amount,
				Currency:       req.Currency,
				IdempotencyKey: randomID(), // a REAL bank assigns ITS OWN event ID, not ours
			}
			body, _ := json.Marshal(payload)
			http.Post(req.WebhookURL, "application/json", bytes.NewReader(body))
		}(req)
	})

	s.Server = httptest.NewServer(mux)
	return s
}

// SimulateDuplicateWebhook resends the exact SAME webhook payload again —
// standing in for the very real, very common scenario of a webhook being
// delivered more than once (Module 20's at-least-once delivery guarantee).
// Used directly by cmd/demo to prove the receiver's idempotency handling.
func SimulateDuplicateWebhook(webhookURL string, payload WebhookPayload) error {
	body, _ := json.Marshal(payload)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook redelivery got status %d", resp.StatusCode)
	}
	return nil
}

func randomID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "evt_" + hex.EncodeToString(b)
}
