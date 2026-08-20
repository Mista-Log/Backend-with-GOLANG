package bank

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestBankAPI_DepositAndBalance is an INTEGRATION test — it exercises the
// real HTTP handler, real JSON encoding/decoding, and the real Bank
// together, via a genuine (if local) HTTP round trip, rather than calling
// Bank's methods directly like bank_test.go's unit tests do.
func TestBankAPI_DepositAndBalance(t *testing.T) {
	b := NewBank(nil)
	if _, err := b.OpenAccount("acc1", 100); err != nil {
		t.Fatalf("setup: OpenAccount failed: %v", err)
	}

	server := httptest.NewServer(NewHTTPHandler(b))
	defer server.Close()

	depositResp, err := http.Post(server.URL+"/deposit", "application/json",
		strings.NewReader(`{"account": "acc1", "amount": 50}`))
	if err != nil {
		t.Fatalf("POST /deposit failed: %v", err)
	}
	defer depositResp.Body.Close()

	if depositResp.StatusCode != http.StatusOK {
		t.Fatalf("POST /deposit status = %d; want 200", depositResp.StatusCode)
	}

	var depositBody balanceResponse
	if err := json.NewDecoder(depositResp.Body).Decode(&depositBody); err != nil {
		t.Fatalf("decoding /deposit response: %v", err)
	}
	if depositBody.Balance != 150 {
		t.Errorf("balance after deposit = %.2f; want 150.00", depositBody.Balance)
	}

	balanceResp, err := http.Get(server.URL + "/balance?account=acc1")
	if err != nil {
		t.Fatalf("GET /balance failed: %v", err)
	}
	defer balanceResp.Body.Close()

	var balanceBody balanceResponse
	if err := json.NewDecoder(balanceResp.Body).Decode(&balanceBody); err != nil {
		t.Fatalf("decoding /balance response: %v", err)
	}
	if balanceBody.Balance != 150 {
		t.Errorf("GET /balance = %.2f; want 150.00 (deposit should have persisted)", balanceBody.Balance)
	}
}

func TestBankAPI_WithdrawInsufficientFunds(t *testing.T) {
	b := NewBank(nil)
	b.OpenAccount("acc1", 10)

	server := httptest.NewServer(NewHTTPHandler(b))
	defer server.Close()

	resp, err := http.Post(server.URL+"/withdraw", "application/json",
		strings.NewReader(`{"account": "acc1", "amount": 100}`))
	if err != nil {
		t.Fatalf("POST /withdraw failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status = %d; want 422 (insufficient funds)", resp.StatusCode)
	}

	var body errorResponse
	json.NewDecoder(resp.Body).Decode(&body)
	if body.Error == "" {
		t.Error("expected a non-empty error message in the response body")
	}
}

func TestBankAPI_UnknownAccount(t *testing.T) {
	b := NewBank(nil)
	server := httptest.NewServer(NewHTTPHandler(b))
	defer server.Close()

	resp, err := http.Get(server.URL + "/balance?account=does-not-exist")
	if err != nil {
		t.Fatalf("GET /balance failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d; want 404", resp.StatusCode)
	}
}
