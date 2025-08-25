// apps/go-server/db.go
//
// Database helpers for the Wordle Go server.
// Responsibilities:
//   - Opening SQLite database with safe defaults (WAL, busy timeout, foreign keys).
//   - Applying migrations from ./sql/*.sql (idempotent, recorded in _migrations).
//   - Convenience helpers for the Daily Challenge (insert/check results, leaderboard).
//
// Note: This file assumes SQLite but can be adapted for other backends.

package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

/**
 * openDB opens (and creates if missing) a SQLite database file.
 *
 * - Ensures parent directory exists for relative DSNs (e.g. ./data/app.db).
 * - Configures busy timeout and WAL journaling mode.
 * - Enforces foreign keys.
 *
 * @param dsn Database path or DSN string.
 * @returns *sql.DB ready for queries/migrations.
 */
func openDB(dsn string) (*sql.DB, error) {
	// Ensure directory exists for ./data/app.db, etc.
	dir := filepath.Dir(dsn)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	// Open DB with busy timeout and WAL journaling.
	db, err := sql.Open("sqlite3", dsn+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	// Explicitly enforce foreign keys + WAL.
	if _, err := db.Exec(`PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;`); err != nil {
		return nil, fmt.Errorf("set pragmas: %w", err)
	}
	return db, nil
}

/**
 * migrate applies SQL migrations from ./sql directory.
 *
 * - Uses a _migrations table to track applied files.
 * - Executes each *.sql file in lexical order.
 * - Skips if already applied.
 * - Detects "self-managed" scripts (with BEGIN TRANSACTION or PRAGMA FOREIGN_KEYS=OFF)
 *   and runs them outside of an outer transaction.
 */
func migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS _migrations (name TEXT PRIMARY KEY);`); err != nil {
		return fmt.Errorf("create _migrations: %w", err)
	}

	// Collect ./sql/*.sql
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
	sort.Strings(files)

	for _, f := range files {
		// Skip if already applied
		var done int
		err := db.QueryRow(`SELECT 1 FROM _migrations WHERE name=?`, f).Scan(&done)
		if err == nil {
			log.Info().Str("migration", f).Msg("already applied")
			continue
		}
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("query _migrations: %w", err)
		}

		// Read file contents
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		sqlText := string(sqlBytes)

		// Detect scripts that manage their own tx or FK pragmas.
		upper := strings.ToUpper(sqlText)
		selfManaged := strings.Contains(upper, "BEGIN TRANSACTION") ||
			strings.Contains(upper, "PRAGMA FOREIGN_KEYS=OFF") ||
			strings.Contains(upper, "PRAGMA FOREIGN_KEYS = OFF")

		if selfManaged {
			// Run as-is
			if _, err := db.Exec(sqlText); err != nil {
				return fmt.Errorf("apply %s: %w", f, err)
			}
			if _, err := db.Exec(`INSERT INTO _migrations(name) VALUES (?)`, f); err != nil {
				return fmt.Errorf("record %s: %w", f, err)
			}
			log.Info().Str("migration", f).Msg("applied (self-managed)")
			continue
		}

		// Run inside dedicated transaction
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(sqlText); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply %s: %w", f, err)
		}
		if _, err := tx.Exec(`INSERT INTO _migrations(name) VALUES (?)`, f); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record %s: %w", f, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit %s: %w", f, err)
		}
		log.Info().Str("migration", f).Msg("applied")
	}
	return nil
}

/* ----------------------- Daily Challenge helpers ------------------------ */

/**
 * DailyResult represents a single user's attempt at the daily challenge.
 * Stored in daily_results table (with UNIQUE(user_id, date)).
 */
type DailyResult struct {
	UserID    string    // User identifier
	Date      string    // "YYYY-MM-DD" (UTC or chosen TZ cutoff)
	WordIndex int       // Index of the day's answer word
	Guesses   int       // Number of guesses taken
	ElapsedMs int       // Time from first input to win (milliseconds)
	CreatedAt time.Time // Populated by DB on insert
}

/** Row type returned for leaderboard queries. */
type DailyLBRow struct {
	UserID    string
	Guesses   int
	ElapsedMs int
}

/**
 * DailyAlreadyPlayed returns true if a user has already played
 * the daily challenge for the given date.
 */
func DailyAlreadyPlayed(ctx context.Context, db *sql.DB, userID, date string) (bool, error) {
	var cnt int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM daily_results WHERE user_id=? AND date=?`,
		userID, date,
	).Scan(&cnt); err != nil {
		return false, err
	}
	return cnt > 0, nil
}

/**
 * InsertDailyResult inserts a new daily result row.
 *
 * - Respects UNIQUE(user_id, date).
 * - If a row already exists, the insert is ignored (no error).
 */
func InsertDailyResult(ctx context.Context, db *sql.DB, r DailyResult) error {
	_, err := db.ExecContext(ctx, `
        INSERT OR IGNORE INTO daily_results
            (user_id, date, word_index, guesses, elapsed_ms)
        VALUES (?, ?, ?, ?, ?)`,
		r.UserID, r.Date, r.WordIndex, r.Guesses, r.ElapsedMs,
	)
	return err
}

/**
 * GetDailyLeaderboard fetches the top players for a given date.
 *
 * - Ordered by elapsed time ASC, then guesses ASC, then created_at ASC.
 * - Default limit is 20 if not specified.
 */
func GetDailyLeaderboard(ctx context.Context, db *sql.DB, date string, limit int) ([]DailyLBRow, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := db.QueryContext(ctx, `
        SELECT user_id, guesses, elapsed_ms
        FROM daily_results
        WHERE date=?
        ORDER BY elapsed_ms ASC, guesses ASC, created_at ASC
        LIMIT ?`, date, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DailyLBRow, 0, limit)
	for rows.Next() {
		var r DailyLBRow
		if err := rows.Scan(&r.UserID, &r.Guesses, &r.ElapsedMs); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
