package main

import (
	"embed"
	"fmt"
	"sort"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func openDB(path string) (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}
	db.SetMaxOpenConns(1) // same SQLite single-writer reasoning as the Banking API project

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	return db, nil
}

func runMigrations(db *sqlx.DB) error {
	db.MustExec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		filename TEXT PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	var filenames []string
	for _, e := range entries {
		filenames = append(filenames, e.Name())
	}
	sort.Strings(filenames)

	for _, name := range filenames {
		var count int
		db.Get(&count, "SELECT COUNT(*) FROM schema_migrations WHERE filename = ?", name)
		if count > 0 {
			continue
		}
		sqlBytes, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(sqlBytes)); err != nil {
			return fmt.Errorf("applying migration %s: %w", name, err)
		}
		db.MustExec("INSERT INTO schema_migrations (filename) VALUES (?)", name)
		fmt.Println("applied migration:", name)
	}
	return nil
}
