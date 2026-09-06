// Package paymentgateway implements Project — Payment Gateway: initiating
// a deposit through the (simulated) external bank, and ONLY posting a
// ledger transaction once the bank's webhook confirms it actually
// settled — never optimistically before confirmation, per the guide's
// Payment Gateway section.
package paymentgateway

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"fintechbackend/internal/bankapi"
	"fintechbackend/internal/ledger"
	"fintechbackend/internal/wallet"
	"fintechbackend/internal/webhook"
)

const bankFundingAccountID = "bank-settlement"

type Service struct {
	ledger      *ledger.Ledger
	wallets     *wallet.Service
	bank        *bankapi.Server
	webhookURL  string
	idempotency *webhook.IdempotencyStore

	mu           sync.Mutex
	pendingByRef map[string]string                 // our reference -> userID, while awaiting the bank's callback
	lastWebhook  map[string]bankapi.WebhookPayload // reference -> the payload we last received, for the demo's duplicate-delivery test
}

func New(l *ledger.Ledger, w *wallet.Service, bank *bankapi.Server, webhookURL string) *Service {
	l.OpenAccount(bankFundingAccountID, "Bank Settlement Account", ledger.Liability, "USD")
	return &Service{
		ledger:       l,
		wallets:      w,
		bank:         bank,
		webhookURL:   webhookURL,
		idempotency:  webhook.NewIdempotencyStore(),
		pendingByRef: make(map[string]string),
		lastWebhook:  make(map[string]bankapi.WebhookPayload),
	}
}

// InitiateDeposit calls out to the (fake) external bank and returns
// IMMEDIATELY — it does NOT wait for settlement, and it does NOT touch
// the ledger at all. The actual credit happens later, in HandleWebhook,
// once the bank confirms.
func (s *Service) InitiateDeposit(userID string, amount int64, currency string) (reference string, err error) {
	if _, err := s.wallets.Get(userID); err != nil {
		return "", err
	}

	reference = randomRef()
	s.mu.Lock()
	s.pendingByRef[reference] = userID
	s.mu.Unlock()

	body, _ := json.Marshal(bankapi.DepositRequest{
		Reference:  reference,
		Amount:     amount,
		Currency:   currency,
		WebhookURL: s.webhookURL,
	})
	resp, err := http.Post(s.bank.URL+"/deposits", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("contacting bank: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("bank rejected deposit request: status %d", resp.StatusCode)
	}
	return reference, nil
}

// HandleWebhook is called when the bank's callback arrives — POTENTIALLY
// more than once for the SAME event (Module 20's at-least-once delivery).
// The idempotency check happens FIRST, before anything else, per the
// guide's Idempotency section.
func (s *Service) HandleWebhook(payload bankapi.WebhookPayload) error {
	s.mu.Lock()
	s.lastWebhook[payload.Reference] = payload // kept so the demo can replay an EXACT duplicate
	s.mu.Unlock()

	if alreadyProcessed := s.idempotency.CheckAndMark(payload.IdempotencyKey); alreadyProcessed {
		return nil // NOT an error — a duplicate delivery is expected and handled by doing nothing
	}

	s.mu.Lock()
	userID, ok := s.pendingByRef[payload.Reference]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("webhook for unknown reference %q", payload.Reference)
	}

	if payload.EventType != "deposit.settled" {
		return nil // a failed-deposit event: nothing to credit
	}

	w, err := s.wallets.Get(userID)
	if err != nil {
		return err
	}

	// ONLY NOW, after bank confirmation AND the idempotency check, does
	// real money move in the ledger.
	_, err = s.ledger.Post("payment-gateway", payload.IdempotencyKey,
		fmt.Sprintf("bank deposit settled for %s", userID), []ledger.Entry{
			{AccountID: w.AccountID, Direction: ledger.Debit, Amount: payload.Amount},
			{AccountID: bankFundingAccountID, Direction: ledger.Credit, Amount: payload.Amount},
		})
	return err
}

// LastWebhookFor returns the exact payload most recently received for a
// reference — used ONLY by cmd/demo, to replay a genuinely identical
// duplicate delivery rather than fabricating one.
func (s *Service) LastWebhookFor(reference string) (bankapi.WebhookPayload, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.lastWebhook[reference]
	return p, ok
}

func randomRef() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "dep_" + hex.EncodeToString(b)
}
