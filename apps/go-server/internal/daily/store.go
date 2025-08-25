// apps/go-server/internal/daily/store.go
//
// Database-backed store for the "Daily Challenge" feature.
// Encapsulates CRUD operations for results and leaderboard queries.
//
// Table expected: daily_results
//   - user_id TEXT
//   - date TEXT ("YYYY-MM-DD")
//   - word_index INT
//   - guesses INT
//   - elapsed_ms INT
//   - created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
//   - UNIQUE(user_id, date)

package daily

import (
	"context"
	"database/sql"
)

/**
 * Result represents a single user's daily challenge attempt.
 * Stored in daily_results table (one row per user per date).
 */
type Result struct {
	UserID    string `json:"userId"`   // User identifier
	Date      string `json:"date"`     // "YYYY-MM-DD"
	WordIndex int    `json:"wordIndex"`// Index of day's answer word
	Guesses   int    `json:"guesses"`  // Number of guesses taken
	ElapsedMs int    `json:"elapsedMs"`// Duration from start to win in ms
}

/**
 * Store wraps a sql.DB and provides methods for daily challenge persistence.
 */
type Store struct{ db *sql.DB }

/** NewStore constructs a daily challenge store bound to the given DB. */
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

/**
 * AlreadyPlayed checks if a user has already played the daily challenge
 * for the given date.
 *
 * @returns true if at least one row exists in daily_results.
 */
func (s *Store) AlreadyPlayed(ctx context.Context, userID, date string) (bool, error) {
	var cnt int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(1) FROM daily_results WHERE user_id=? AND date=?",
		userID, date,
	).Scan(&cnt)
	return cnt > 0, err
}

/**
 * InsertResult inserts a new daily result row.
 *
 * - Uses INSERT OR IGNORE to respect UNIQUE(user_id, date).
 * - If the user already has a row for the given date, this is a no-op.
 */
func (s *Store) InsertResult(ctx context.Context, r Result) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO daily_results(user_id, date, word_index, guesses, elapsed_ms)
		 VALUES(?,?,?,?,?)`,
		r.UserID, r.Date, r.WordIndex, r.Guesses, r.ElapsedMs,
	)
	return err
}

/**
 * LBRow represents a leaderboard entry for a given day.
 */
type LBRow struct {
	UserID    string `json:"userId"`
	Guesses   int    `json:"guesses"`
	ElapsedMs int    `json:"elapsedMs"`
}

/**
 * Leaderboard returns the top players for a given date.
 *
 * - Sorted by elapsed_ms ASC, then guesses ASC, then created_at ASC.
 * - Limit is enforced by the query.
 */
func (s *Store) Leaderboard(ctx context.Context, date string, limit int) ([]LBRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT user_id, guesses, elapsed_ms
		   FROM daily_results
		  WHERE date=?
		  ORDER BY elapsed_ms ASC, guesses ASC, created_at ASC
		  LIMIT ?`, date, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LBRow
	for rows.Next() {
		var r LBRow
		if err := rows.Scan(&r.UserID, &r.Guesses, &r.ElapsedMs); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
