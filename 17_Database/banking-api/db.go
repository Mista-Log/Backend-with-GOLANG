package main

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"time"

	_ "modernc.org/sqlite" // pure-Go driver — registers itself via import side effect
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// openDB connects, verifies connectivity, configures the pool, and applies
// every migration — everything a real Go program needs before its first
// real query.
func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	// SQLite allows only ONE writer at a time at the database-file level,
	// no matter how many *sql.DB connections you open — capping MaxOpenConns
	// at 1 here avoids "database is locked" errors under concurrent writes,
	// by having Go's own connection pool queue writers instead of the
	// database rejecting them. (Postgres/MySQL, which handle real concurrent
	// writers themselves, would instead use a much higher number here —
	// see the guide's Connection Pool section.)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return db, nil
}

// runMigrations applies every *.sql file under migrations/, in filename
// order, tracking what's already been applied in its own table — so
// running this on an already-migrated database is always a safe no-op.
func runMigrations(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		filename TEXT PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return err
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	var filenames []string
	for _, e := range entries {
		filenames = append(filenames, e.Name())
	}
	sort.Strings(filenames) // ensures 0001_... runs before 0002_..., etc.

	for _, name := range filenames {
		var alreadyApplied int
		db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE filename = ?", name).Scan(&alreadyApplied)
		if alreadyApplied > 0 {
			continue // already ran this one — skip it
		}

		sqlBytes, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(sqlBytes)); err != nil {
			return fmt.Errorf("applying migration %s: %w", name, err)
		}
		if _, err := db.Exec("INSERT INTO schema_migrations (filename) VALUES (?)", name); err != nil {
			return err
		}
		fmt.Println("applied migration:", name)
	}
	return nil
}
