// Package httpapi provides the one piece of this project that MUST be
// real HTTP by definition: a webhook is an inbound HTTP callback, so
// something has to be listening. Everything else in this demo is
// exercised via direct function calls for clarity (see cmd/demo's
// comment on this choice) — but a webhook receiver can't be anything
// other than a real server.
package httpapi

import (
	"encoding/json"
	"log"
	"net/http"

	"fintechbackend/internal/bankapi"
	"fintechbackend/internal/paymentgateway"
)

func NewRouter(gateway *paymentgateway.Service) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /webhooks/bank", func(w http.ResponseWriter, r *http.Request) {
		var payload bankapi.WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if err := gateway.HandleWebhook(payload); err != nil {
			log.Printf("webhook processing error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Respond 200 quickly — the guide's Webhook section: slow
		// processing inside a webhook handler is what causes gateways to
		// time out and redeliver, creating MORE duplicates, not fewer.
		w.WriteHeader(http.StatusOK)
	})

	return mux
}
