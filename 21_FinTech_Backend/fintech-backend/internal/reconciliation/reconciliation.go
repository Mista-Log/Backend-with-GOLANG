// Package reconciliation implements Project — Ledger Engine's companion
// safeguard and the guide's Reconciliation section: continuously verifying
// that the ledger's internal story is actually consistent, and that it
// matches the outside world's own records where an external system is
// involved.
package reconciliation

import (
	"fmt"

	"fintechbackend/internal/ledger"
)

type Discrepancy struct {
	Description string
	Expected    int64
	Actual      int64
}

type Report struct {
	Balanced     bool
	Discrepancies []Discrepancy
}

// CheckLedgerIntegrity is the INTERNAL check from the guide's diagram:
// system-wide, does the sum of every debit entry ever posted equal the
// sum of every credit entry ever posted? If this ever fails, something is
// structurally broken in the ledger implementation itself — Post's own
// balance check (ledger.go) should make this mathematically impossible,
// so this function is the independent, external proof that guarantee
// actually held in practice, not just in theory.
func CheckLedgerIntegrity(l *ledger.Ledger, accountIDs []string) Report {
	var totalDebits, totalCredits int64
	for _, id := range accountIDs {
		for _, e := range l.EntriesFor(id) {
			switch e.Direction {
			case ledger.Debit:
				totalDebits += e.Amount
			case ledger.Credit:
				totalCredits += e.Amount
			}
		}
	}

	if totalDebits != totalCredits {
		return Report{
			Balanced: false,
			Discrepancies: []Discrepancy{{
				Description: "system-wide debits do not equal system-wide credits",
				Expected:    totalDebits,
				Actual:      totalCredits,
			}},
		}
	}
	return Report{Balanced: true}
}

// ReconcileAgainstExternalStatement is the EXTERNAL check: compare the
// ledger's own view of "how much has arrived from the bank" against a
// statement the bank itself reports — standing in for a real end-of-day
// bank statement file/API a production system would fetch and compare.
func ReconcileAgainstExternalStatement(l *ledger.Ledger, bankAccountID string, externalStatementTotal int64) Report {
	ledgerTotal, err := l.Balance(bankAccountID)
	if err != nil {
		return Report{Balanced: false, Discrepancies: []Discrepancy{{Description: err.Error()}}}
	}

	// The bank-facing account is a Liability that increases as deposits
	// arrive — its balance should match the bank's own reported total
	// exactly. Any gap here is precisely the kind of thing this module's
	// guide called the "last line of defense."
	if ledgerTotal != externalStatementTotal {
		return Report{
			Balanced: false,
			Discrepancies: []Discrepancy{{
				Description: fmt.Sprintf("ledger vs. bank statement mismatch for %s", bankAccountID),
				Expected:    externalStatementTotal,
				Actual:      ledgerTotal,
			}},
		}
	}
	return Report{Balanced: true}
}
