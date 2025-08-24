package main

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

// openDB opens (and creates) the SQLite DB and applies sensible PRAGMAs.
func openDB(dsn string) (*sql.DB, error) {
	// Ensure the directory exists for relative paths like ./data/app.db
	dir := filepath.Dir(dsn)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	// Busy timeout + WAL; foreign_keys is enforced via PRAGMA below for clarity
	db, err := sql.Open("sqlite3", dsn+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	// Enforce foreign keys and WAL (again)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;`); err != nil {
		return nil, fmt.Errorf("set pragmas: %w", err)
	}
	return db, nil
}

// migrate applies all SQL files under ./sql in lexical order once.
func migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS _migrations (name TEXT PRIMARY KEY);`); err != nil {
		return fmt.Errorf("create _migrations: %w", err)
	}

	// Collect all .sql files under ./sql (same CWD as the binary working dir)
	root := "sql"
	var files []string
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".sql") {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("walk sql dir: %w", err)
	}

	if len(files) == 0 {
		log.Warn().Msg("no .sql files found under ./sql; skipping migrations")
		return nil
	}
	sort.Strings(files)

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, f := range files {
		var already int
		err := tx.QueryRow(`SELECT 1 FROM _migrations WHERE name = ?`, f).Scan(&already)
		if err == nil {
			log.Info().Str("migration", f).Msg("already applied")
			continue
		}
		if err != sql.ErrNoRows && err != nil {
			return fmt.Errorf("query migrations: %w", err)
		}

		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}

		if _, err := tx.Exec(string(sqlBytes)); err != nil {
			return fmt.Errorf("apply %s: %w", f, err)
		}
		if _, err := tx.Exec(`INSERT INTO _migrations(name) VALUES (?)`, f); err != nil {
			return fmt.Errorf("record %s: %w", f, err)
		}

		log.Info().Str("migration", f).Msg("applied")
	}

	return tx.Commit()
}
