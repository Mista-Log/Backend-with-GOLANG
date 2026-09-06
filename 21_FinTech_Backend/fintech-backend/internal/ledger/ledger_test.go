package ledger

import (
	"errors"
	"testing"
)

func newTestLedger() *Ledger {
	return New(NewAuditLog())
}

func TestPost_BalancedTransactionSucceeds(t *testing.T) {
	l := newTestLedger()
	l.OpenAccount("cash", "Cash", Asset, "USD")
	l.OpenAccount("alice-wallet", "Alice's Wallet", Liability, "USD")

	tx, err := l.Post("test", "key-1", "deposit", []Entry{
		{AccountID: "cash", Direction: Debit, Amount: 5000},
		{AccountID: "alice-wallet", Direction: Credit, Amount: 5000},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.Status != Completed {
		t.Errorf("status = %v; want Completed", tx.Status)
	}
}

func TestPost_UnbalancedTransactionRejected(t *testing.T) {
	l := newTestLedger()
	l.OpenAccount("cash", "Cash", Asset, "USD")
	l.OpenAccount("alice-wallet", "Alice's Wallet", Liability, "USD")

	_, err := l.Post("test", "key-1", "broken", []Entry{
		{AccountID: "cash", Direction: Debit, Amount: 5000},
		{AccountID: "alice-wallet", Direction: Credit, Amount: 4000}, // deliberately mismatched
	})
	if err == nil {
		t.Fatal("expected an error for an unbalanced transaction, got nil")
	}
	var unbalanced *ErrUnbalanced
	if !errors.As(err, &unbalanced) {
		t.Fatalf("expected *ErrUnbalanced, got %T: %v", err, err)
	}
	if unbalanced.TotalDebits != 5000 || unbalanced.TotalCredits != 4000 {
		t.Errorf("got debits=%d credits=%d; want 5000/4000", unbalanced.TotalDebits, unbalanced.TotalCredits)
	}

	// The whole point of rejecting atomically: NEITHER entry should have
	// been written. Confirm both accounts still show a zero balance.
	cashBalance, _ := l.Balance("cash")
	if cashBalance != 0 {
		t.Errorf("cash balance after a REJECTED transaction = %d; want 0 (nothing should have been written)", cashBalance)
	}
}

func TestPost_IdempotencyKeyPreventsDoubleProcessing(t *testing.T) {
	l := newTestLedger()
	l.OpenAccount("cash", "Cash", Asset, "USD")
	l.OpenAccount("alice-wallet", "Alice's Wallet", Liability, "USD")

	entries := []Entry{
		{AccountID: "cash", Direction: Debit, Amount: 5000},
		{AccountID: "alice-wallet", Direction: Credit, Amount: 5000},
	}

	tx1, err := l.Post("test", "same-key", "deposit", entries)
	if err != nil {
		t.Fatalf("first Post failed: %v", err)
	}

	// Simulate a client retry after a lost response — SAME idempotency key.
	tx2, err := l.Post("test", "same-key", "deposit", entries)
	if err != nil {
		t.Fatalf("second Post (retry) failed: %v", err)
	}

	if tx1.ID != tx2.ID {
		t.Errorf("retry produced a DIFFERENT transaction (%s vs %s) — idempotency is broken", tx1.ID, tx2.ID)
	}

	// The balance must reflect ONE deposit, not two.
	balance, _ := l.Balance("alice-wallet")
	if balance != 5000 {
		t.Errorf("alice-wallet balance = %d; want 5000 (retried request must NOT double-apply)", balance)
	}
}

func TestBalance_LiabilityIncreasesOnCredit(t *testing.T) {
	l := newTestLedger()
	l.OpenAccount("cash", "Cash", Asset, "USD")
	l.OpenAccount("alice-wallet", "Alice's Wallet", Liability, "USD")

	l.Post("test", "k1", "deposit", []Entry{
		{AccountID: "cash", Direction: Debit, Amount: 10000},
		{AccountID: "alice-wallet", Direction: Credit, Amount: 10000},
	})

	cashBalance, _ := l.Balance("cash")
	walletBalance, _ := l.Balance("alice-wallet")

	// The guide's Double Entry table: Asset increases on DEBIT, Liability
	// increases on CREDIT. Both entries were for the SAME amount, but each
	// account's balance should reflect its OWN type's increasing direction.
	if cashBalance != 10000 {
		t.Errorf("cash (asset) balance = %d; want 10000 (debit should increase an asset)", cashBalance)
	}
	if walletBalance != 10000 {
		t.Errorf("alice-wallet (liability) balance = %d; want 10000 (credit should increase a liability)", walletBalance)
	}
}

func TestPost_UnknownAccountRejected(t *testing.T) {
	l := newTestLedger()
	l.OpenAccount("cash", "Cash", Asset, "USD")

	_, err := l.Post("test", "k1", "bad", []Entry{
		{AccountID: "cash", Direction: Debit, Amount: 100},
		{AccountID: "does-not-exist", Direction: Credit, Amount: 100},
	})
	if err == nil {
		t.Fatal("expected an error for an unknown account, got nil")
	}
}
