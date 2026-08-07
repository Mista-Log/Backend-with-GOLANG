// Project: Mini Database
//
// An in-memory key/value store built specifically to put pointers front and
// center: records are stored as *Record so updates mutate in place, reads
// come in both a dangerous "live pointer" flavor and a safe "copy" flavor,
// and transactions are implemented via a deep-copied snapshot for rollback.
package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// Record is an ordinary struct — nothing about it is special until you
// notice how it's stored and passed around below.
type Record struct {
	Key       string
	Value     string
	Version   int
	UpdatedAt time.Time
}

// update uses a POINTER receiver deliberately: it needs to mutate the
// record that's actually stored in the database, not a throwaway copy.
func (r *Record) update(value string) {
	r.Value = value
	r.Version++
	r.UpdatedAt = time.Now()
}

// DB stores *Record, not Record, so every lookup returns something that
// can be mutated in place through the map — the same map-of-pointers
// pattern from Module 04's Inventory System, now explained at the pointer
// level instead of just used.
type DB struct {
	records map[string]*Record
	log     []string
}

func NewDB() *DB {
	return &DB{records: make(map[string]*Record)}
}

// Set creates a record if it's new, or updates it in place (through the
// pointer receiver) if it already exists — either way, ESCAPE ANALYSIS
// decides that `Record{}` below can't live on Set's stack frame, because
// its address is about to be stored in db.records and outlive this call.
// Go moves it to the heap automatically; you can confirm this yourself with
// `go build -gcflags="-m" .`
func (db *DB) Set(key, value string) *Record {
	if r, exists := db.records[key]; exists {
		r.update(value)
		db.log = append(db.log, fmt.Sprintf("updated %q -> %q (v%d)", key, value, r.Version))
		return r
	}
	r := &Record{Key: key, Value: value, Version: 1, UpdatedAt: time.Now()}
	db.records[key] = r
	db.log = append(db.log, fmt.Sprintf("created %q -> %q (v1)", key, value))
	return r
}

// GetLive returns the ACTUAL stored pointer — fast, but dangerous: whoever
// calls this can mutate the database directly, bypassing update()'s
// version-bumping and logging entirely. Included so the danger is visible
// and comparable against GetCopy below, not as a recommended pattern.
func (db *DB) GetLive(key string) (*Record, bool) {
	r, ok := db.records[key]
	return r, ok
}

// GetCopy returns a genuinely independent copy — dereferencing the stored
// pointer and assigning it to a new local Record value copies every field.
// Safe for callers to inspect or even modify without touching the database.
func (db *DB) GetCopy(key string) (Record, bool) {
	r, ok := db.records[key]
	if !ok {
		return Record{}, false
	}
	return *r, ok // *r dereferences, then the assignment copies the struct
}

func (db *DB) Delete(key string) error {
	if _, ok := db.records[key]; !ok {
		return fmt.Errorf("no record with key %q", key)
	}
	delete(db.records, key)
	db.log = append(db.log, fmt.Sprintf("deleted %q", key))
	return nil
}

func (db *DB) SortedKeys() []string {
	keys := make([]string, 0, len(db.records))
	for k := range db.records {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Snapshot returns map[string]Record — VALUES, not pointers — on purpose.
// This is what makes Transaction.Rollback below actually work: if this
// returned map[string]*Record instead, "snapshotting" would just hand back
// the same live pointers already in db.records, and mutating the database
// afterward would silently corrupt the "snapshot" too.
func (db *DB) Snapshot() map[string]Record {
	snap := make(map[string]Record, len(db.records))
	for k, r := range db.records {
		snap[k] = *r // dereference + copy, same idea as GetCopy
	}
	return snap
}

// Transaction holds a snapshot taken at Begin() time, and knows how to
// restore the database to exactly that state.
type Transaction struct {
	db       *DB
	snapshot map[string]Record
}

func (db *DB) Begin() *Transaction {
	return &Transaction{db: db, snapshot: db.Snapshot()}
}

// Rollback restores every key that existed at Begin() time back to its
// snapshotted value, and removes any key that was created DURING the
// transaction (present now, absent from the snapshot).
func (tx *Transaction) Rollback() {
	for key := range tx.db.records {
		if _, existedBefore := tx.snapshot[key]; !existedBefore {
			delete(tx.db.records, key) // created during the tx — undo it
		}
	}
	for key, snapshotValue := range tx.snapshot {
		restored := snapshotValue // local copy, its OWN address, safe to point at
		tx.db.records[key] = &restored
	}
	tx.db.log = append(tx.db.log, "transaction rolled back")
}

func (tx *Transaction) Commit() {
	tx.db.log = append(tx.db.log, "transaction committed")
	// Nothing else to do — every Set() call during the transaction already
	// wrote straight into the live db.records map. "Commit" here just means
	// "stop tracking the snapshot; keep what's there."
}

func readLine(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	db := NewDB()

main:
	for {
		fmt.Println("\n1) Set  2) Get (safe copy)  3) Get (live pointer, risky)  4) Delete  5) List keys  6) Begin transaction  7) Show log  8) Exit")
		choice := readLine(reader, "Choose an option: ")

		switch choice {
		case "1":
			key := readLine(reader, "Key: ")
			value := readLine(reader, "Value: ")
			r := db.Set(key, value)
			fmt.Printf("Stored. version=%d updatedAt=%s\n", r.Version, r.UpdatedAt.Format(time.RFC3339))

		case "2":
			key := readLine(reader, "Key: ")
			r, ok := db.GetCopy(key)
			if !ok {
				fmt.Println("Not found.")
				continue main
			}
			fmt.Printf("%s = %q (v%d) [this is a COPY — modifying it won't touch the DB]\n", r.Key, r.Value, r.Version)

		case "3":
			key := readLine(reader, "Key: ")
			r, ok := db.GetLive(key)
			if !ok {
				fmt.Println("Not found.")
				continue main
			}
			fmt.Printf("%s = %q (v%d) [this is the LIVE pointer]\n", r.Key, r.Value, r.Version)
			fmt.Println("Demonstrating the danger: mutating it directly, bypassing Set()...")
			r.Value = "TAMPERED (version not bumped, no log entry!)"
			fmt.Printf("db now has: %s = %q (v%d, unchanged) — corrupted without a log entry\n", r.Key, r.Value, r.Version)

		case "4":
			key := readLine(reader, "Key to delete: ")
			if err := db.Delete(key); err != nil {
				fmt.Println("Error:", err)
			}

		case "5":
			keys := db.SortedKeys()
			if len(keys) == 0 {
				fmt.Println("Database is empty.")
			}
			for _, k := range keys {
				r := db.records[k]
				fmt.Printf("  %-15s = %-20q v%d\n", k, r.Value, r.Version)
			}

		case "6":
			runTransaction(reader, db)

		case "7":
			if len(db.log) == 0 {
				fmt.Println("No activity yet.")
			}
			for _, entry := range db.log {
				fmt.Println(" -", entry)
			}

		case "8":
			break main

		default:
			fmt.Println("Unknown option.")
		}
	}

	fmt.Println("\nGoodbye!")
}

// runTransaction is its own labeled loop (same pattern as Module 02's
// Banking Menu) so "commit" and "rollback" can cleanly return control to
// the main menu without any extra flags.
func runTransaction(reader *bufio.Reader, db *DB) {
	tx := db.Begin()
	fmt.Println("Transaction started — changes are LIVE immediately, but Rollback")
	fmt.Println("will restore everything to how it was right now.")

tx:
	for {
		fmt.Println("\n1) Set  2) List keys  3) Commit  4) Rollback")
		choice := readLine(reader, "Transaction option: ")

		switch choice {
		case "1":
			key := readLine(reader, "Key: ")
			value := readLine(reader, "Value: ")
			db.Set(key, value)
			fmt.Println("Set (live).")

		case "2":
			for _, k := range db.SortedKeys() {
				r := db.records[k]
				fmt.Printf("  %-15s = %-20q v%d\n", k, r.Value, r.Version)
			}

		case "3":
			tx.Commit()
			fmt.Println("Committed.")
			break tx

		case "4":
			tx.Rollback()
			fmt.Println("Rolled back — database restored to pre-transaction state.")
			break tx

		default:
			fmt.Println("Unknown option.")
		}
	}
}
