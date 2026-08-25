package main

import (
	"context"
	"database/sql"
	"fmt"
)

type Account struct {
	ID      int
	Owner   string
	Balance float64
}

func CreateAccount(db *sql.DB, owner string, initialBalance float64) (int, error) {
	result, err := db.Exec("INSERT INTO accounts (owner, balance) VALUES (?, ?)", owner, initialBalance)
	if err != nil {
		return 0, fmt.Errorf("creating account: %w", err)
	}
	id, err := result.LastInsertId()
	return int(id), err
}

func GetAccount(db *sql.DB, id int) (*Account, error) {
	var a Account
	err := db.QueryRow("SELECT id, owner, balance FROM accounts WHERE id = ?", id).
		Scan(&a.ID, &a.Owner, &a.Balance)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no account with id %d", id)
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// Transfer moves money between two accounts inside ONE transaction: deduct,
// credit, and log — all three commit together, or none of them do. Uses
// LevelSerializable specifically because a transfer is exactly the kind of
// operation where two concurrent transfers touching the SAME account could
// otherwise interleave incorrectly (the guide's Isolation Levels section).
func Transfer(ctx context.Context, db *sql.DB, fromID, toID int, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("transfer amount must be positive")
	}

	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback() // no-op if Commit() below succeeds first

	var fromBalance float64
	if err := tx.QueryRow("SELECT balance FROM accounts WHERE id = ?", fromID).Scan(&fromBalance); err != nil {
		return fmt.Errorf("looking up source account: %w", err)
	}
	if fromBalance < amount {
		return fmt.Errorf("insufficient funds: account %d has %.2f, needs %.2f", fromID, fromBalance, amount)
	}

	// The CHECK(balance >= 0) constraint from the migration is a SECOND
	// line of defense here — even if the application-level check above had
	// a bug, the database itself would still reject a negative balance.
	if _, err := tx.Exec("UPDATE accounts SET balance = balance - ? WHERE id = ?", amount, fromID); err != nil {
		return fmt.Errorf("deducting from source account: %w", err)
	}
	if _, err := tx.Exec("UPDATE accounts SET balance = balance + ? WHERE id = ?", amount, toID); err != nil {
		return fmt.Errorf("crediting destination account: %w", err)
	}
	if _, err := tx.Exec("INSERT INTO transfers (from_id, to_id, amount) VALUES (?, ?, ?)", fromID, toID, amount); err != nil {
		return fmt.Errorf("logging transfer: %w", err)
	}

	return tx.Commit()
}

type TransferRecord struct {
	ID       int
	FromID   int
	ToID     int
	Amount   float64
	Created  string
}

func ListTransfers(db *sql.DB) ([]TransferRecord, error) {
	rows, err := db.Query("SELECT id, from_id, to_id, amount, created_at FROM transfers ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []TransferRecord
	for rows.Next() {
		var r TransferRecord
		if err := rows.Scan(&r.ID, &r.FromID, &r.ToID, &r.Amount, &r.Created); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}
