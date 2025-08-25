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

    // Collect and sort ./sql/*.sql
    root := "sql"
    var files []string
    if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
        if err != nil { return err }
        if d.IsDir() { return nil }
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

        sqlBytes, err := os.ReadFile(f)
        if err != nil {
            return fmt.Errorf("read %s: %w", f, err)
        }
        sqlText := string(sqlBytes)

        // Detect scripts that manage their own transaction / FK pragma toggles.
        upper := strings.ToUpper(sqlText)
        selfManaged := strings.Contains(upper, "BEGIN TRANSACTION") ||
            strings.Contains(upper, "PRAGMA FOREIGN_KEYS=OFF") ||
            strings.Contains(upper, "PRAGMA FOREIGN_KEYS = OFF")

        if selfManaged {
            // Execute as-is, not inside an outer tx
            if _, err := db.Exec(sqlText); err != nil {
                return fmt.Errorf("apply %s: %w", f, err)
            }
            if _, err := db.Exec(`INSERT INTO _migrations(name) VALUES (?)`, f); err != nil {
                return fmt.Errorf("record %s: %w", f, err)
            }
            log.Info().Str("migration", f).Msg("applied (self-managed)")
            continue
        }

        // Normal migration: run inside its own transaction
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

type DailyResult struct {
    UserID    string    // required
    Date      string    // "YYYY-MM-DD" (UTC or your chosen tz boundary)
    WordIndex int       // index into answers list
    Guesses   int       // number of guesses taken
    ElapsedMs int       // time from first input to win
    CreatedAt time.Time // filled by DB on insert; optional on read
}

type DailyLBRow struct {
    UserID    string
    Guesses   int
    ElapsedMs int
}

// DailyAlreadyPlayed returns true if user has a row for the given date.
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

// InsertDailyResult inserts a result; respects UNIQUE(user_id, date).
// If a duplicate exists, this is a no-op (returns nil).
func InsertDailyResult(ctx context.Context, db *sql.DB, r DailyResult) error {
    _, err := db.ExecContext(ctx, `
        INSERT OR IGNORE INTO daily_results
            (user_id, date, word_index, guesses, elapsed_ms)
        VALUES (?, ?, ?, ?, ?)`,
        r.UserID, r.Date, r.WordIndex, r.Guesses, r.ElapsedMs,
    )
    return err
}

// GetDailyLeaderboard returns top rows ordered by elapsed time, then guesses, then created_at.
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
