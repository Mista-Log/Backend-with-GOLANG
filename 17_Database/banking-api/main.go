// Banking API — plain database/sql against SQLite, with every SQL
// statement, transaction boundary, and isolation-level choice fully
// explicit. Demonstrates migrations, constraints, a real ACID transfer
// (with a deliberate rollback-on-failure demo), and concurrent-safe writes.
//
// Run with: go mod tidy && go run .
package main

import (
	"context"
	"fmt"
	"os"
	"sync"
)

func main() {
	os.Remove("bank.db") // start fresh each run, for a repeatable demo

	db, err := openDB("bank.db")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer db.Close()

	ctx := context.Background()

	fmt.Println("\n=== Creating accounts ===")
	aliceID, _ := CreateAccount(db, "alice", 500)
	bobID, _ := CreateAccount(db, "bob", 100)
	fmt.Printf("alice: account #%d, bob: account #%d\n", aliceID, bobID)

	fmt.Println("\n=== A successful transfer ===")
	if err := Transfer(ctx, db, aliceID, bobID, 150); err != nil {
		fmt.Println("Error:", err)
	} else {
		alice, _ := GetAccount(db, aliceID)
		bob, _ := GetAccount(db, bobID)
		fmt.Printf("alice: %.2f, bob: %.2f\n", alice.Balance, bob.Balance)
	}

	fmt.Println("\n=== A transfer that fails (insufficient funds) — rollback demo ===")
	before, _ := GetAccount(db, bobID)
	err = Transfer(ctx, db, bobID, aliceID, 10000)
	after, _ := GetAccount(db, bobID)
	fmt.Println("error:", err)
	fmt.Printf("bob's balance before attempt: %.2f, after failed attempt: %.2f (UNCHANGED — the deduction never committed)\n",
		before.Balance, after.Balance)

	fmt.Println("\n=== The database CHECK constraint, tested directly ===")
	_, err = db.Exec("UPDATE accounts SET balance = -50 WHERE id = ?", aliceID)
	fmt.Println("attempting to force a negative balance directly:", err)

	fmt.Println("\n=== Several concurrent transfers — safe thanks to the transaction + connection pool ===")
	carolID, _ := CreateAccount(db, "carol", 1000)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Transfer(ctx, db, carolID, aliceID, 10) // ten concurrent $10 transfers
		}()
	}
	wg.Wait()

	carol, _ := GetAccount(db, carolID)
	alice, _ := GetAccount(db, aliceID)
	fmt.Printf("carol: %.2f (expected 900.00 — exactly 100 deducted across 10 transfers, no lost updates)\n", carol.Balance)
	fmt.Printf("alice: %.2f\n", alice.Balance)

	fmt.Println("\n=== Full transfer log (the audit table, written inside every transaction) ===")
	transfers, _ := ListTransfers(db)
	for _, t := range transfers {
		fmt.Printf("  #%d: %d -> %d, $%.2f at %s\n", t.ID, t.FromID, t.ToID, t.Amount, t.Created)
	}
}
