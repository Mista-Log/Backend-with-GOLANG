package bank

import "testing"

// --- Unit tests -----------------------------------------------------

func TestAccount_Deposit(t *testing.T) {
	acc := NewAccount("acc1", 100, nil)

	if err := acc.Deposit(50); err != nil {
		t.Fatalf("Deposit(50) returned unexpected error: %v", err)
	}
	if got := acc.Balance(); got != 150 {
		t.Errorf("Balance() = %.2f; want 150.00", got)
	}
}

func TestAccount_Deposit_RejectsNonPositive(t *testing.T) {
	acc := NewAccount("acc1", 100, nil)

	if err := acc.Deposit(0); err == nil {
		t.Error("Deposit(0) should have returned an error, got nil")
	}
	if err := acc.Deposit(-10); err == nil {
		t.Error("Deposit(-10) should have returned an error, got nil")
	}
	if got := acc.Balance(); got != 100 {
		t.Errorf("Balance() after rejected deposits = %.2f; want unchanged 100.00", got)
	}
}

// --- Table test -----------------------------------------------------

func TestAccount_Withdraw(t *testing.T) {
	tests := []struct {
		name          string
		startBalance  float64
		withdrawAmount float64
		wantErr       bool
		wantBalance   float64
	}{
		{"simple withdrawal", 100, 30, false, 70},
		{"withdraw entire balance", 100, 100, false, 0},
		{"withdraw more than balance", 100, 150, true, 100}, // rejected — balance unchanged
		{"withdraw zero", 100, 0, true, 100},
		{"withdraw negative", 100, -10, true, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acc := NewAccount("acc1", tt.startBalance, nil)
			err := acc.Withdraw(tt.withdrawAmount)

			if tt.wantErr && err == nil {
				t.Errorf("Withdraw(%.2f) with balance %.2f: expected an error, got nil", tt.withdrawAmount, tt.startBalance)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Withdraw(%.2f) with balance %.2f: unexpected error: %v", tt.withdrawAmount, tt.startBalance, err)
			}
			if got := acc.Balance(); got != tt.wantBalance {
				t.Errorf("Balance() after Withdraw(%.2f) = %.2f; want %.2f", tt.withdrawAmount, got, tt.wantBalance)
			}
		})
	}
}

// --- Mocking (a hand-written fake, per Module 14's guide section 6) --------

// FakeLogger records every call it receives, so tests can assert on WHAT
// was logged without touching a real file or database.
type FakeLogger struct {
	calls []loggedCall
}

type loggedCall struct {
	accountID string
	amount    float64
}

func (f *FakeLogger) Log(accountID string, amount float64) {
	f.calls = append(f.calls, loggedCall{accountID, amount})
}

func TestDeposit_LogsTransaction(t *testing.T) {
	logger := &FakeLogger{}
	acc := NewAccount("acc1", 100, logger)

	if err := acc.Deposit(50); err != nil {
		t.Fatalf("Deposit returned unexpected error: %v", err)
	}

	if len(logger.calls) != 1 {
		t.Fatalf("expected 1 logged call, got %d", len(logger.calls))
	}
	if logger.calls[0].accountID != "acc1" || logger.calls[0].amount != 50 {
		t.Errorf("logged call = %+v; want {acc1 50}", logger.calls[0])
	}
}

func TestWithdraw_LogsNegativeAmount(t *testing.T) {
	logger := &FakeLogger{}
	acc := NewAccount("acc1", 100, logger)

	if err := acc.Withdraw(30); err != nil {
		t.Fatalf("Withdraw returned unexpected error: %v", err)
	}

	if len(logger.calls) != 1 {
		t.Fatalf("expected 1 logged call, got %d", len(logger.calls))
	}
	// Withdrawals log a NEGATIVE amount — a deliberate convention so a
	// transaction history built purely from logged calls could sum to the
	// correct running balance without needing to know each call's "type."
	if logger.calls[0].amount != -30 {
		t.Errorf("logged amount = %.2f; want -30.00", logger.calls[0].amount)
	}
}

func TestDeposit_RejectedAmounts_DoNotLog(t *testing.T) {
	logger := &FakeLogger{}
	acc := NewAccount("acc1", 100, logger)

	acc.Deposit(-10) // rejected — should never reach the logger

	if len(logger.calls) != 0 {
		t.Errorf("expected 0 logged calls for a rejected deposit, got %d", len(logger.calls))
	}
}

// --- Bank / Transfer tests -----------------------------------------------

func TestBank_OpenAccount_RejectsDuplicates(t *testing.T) {
	b := NewBank(nil)
	if _, err := b.OpenAccount("acc1", 100); err != nil {
		t.Fatalf("first OpenAccount failed: %v", err)
	}
	if _, err := b.OpenAccount("acc1", 50); err == nil {
		t.Error("second OpenAccount with the same ID should have failed, got nil error")
	}
}

func TestTransfer_Success(t *testing.T) {
	b := NewBank(nil)
	b.OpenAccount("alice", 100)
	b.OpenAccount("bob", 50)

	if err := b.Transfer("alice", "bob", 40); err != nil {
		t.Fatalf("Transfer returned unexpected error: %v", err)
	}

	alice, _ := b.Account("alice")
	bob, _ := b.Account("bob")
	if alice.Balance() != 60 {
		t.Errorf("alice balance = %.2f; want 60.00", alice.Balance())
	}
	if bob.Balance() != 90 {
		t.Errorf("bob balance = %.2f; want 90.00", bob.Balance())
	}
}

func TestTransfer_InsufficientFunds_NoChange(t *testing.T) {
	b := NewBank(nil)
	b.OpenAccount("alice", 10)
	b.OpenAccount("bob", 50)

	err := b.Transfer("alice", "bob", 100)
	if err == nil {
		t.Fatal("expected an error for insufficient funds, got nil")
	}

	alice, _ := b.Account("alice")
	bob, _ := b.Account("bob")
	if alice.Balance() != 10 {
		t.Errorf("alice balance = %.2f; want unchanged 10.00", alice.Balance())
	}
	if bob.Balance() != 50 {
		t.Errorf("bob balance = %.2f; want unchanged 50.00", bob.Balance())
	}
}

func TestTransfer_UnknownAccount(t *testing.T) {
	b := NewBank(nil)
	b.OpenAccount("alice", 100)

	if err := b.Transfer("alice", "nobody", 10); err == nil {
		t.Error("expected an error transferring to an unknown account, got nil")
	}
	if err := b.Transfer("nobody", "alice", 10); err == nil {
		t.Error("expected an error transferring from an unknown account, got nil")
	}
}

// --- ParseAmount unit + table tests -----------------------------------

func TestParseAmount(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Amount
		wantErr bool
	}{
		{"plain number", "19.99", 19.99, false},
		{"dollar sign", "$19.99", 19.99, false},
		{"whitespace", "  42.50  ", 42.50, false},
		{"zero", "0", 0, false},
		{"negative", "-5.00", -5, false},
		{"empty string", "", 0, true},
		{"just a dollar sign", "$", 0, true},
		{"garbage", "not-a-number", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAmount(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("ParseAmount(%q): expected an error, got nil", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ParseAmount(%q): unexpected error: %v", tt.input, err)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseAmount(%q) = %v; want %v", tt.input, got, tt.want)
			}
		})
	}
}
