package daily

import (
"context"
"database/sql"
)

type Result struct {
UserID string `json:"userId"`
Date string `json:"date"`
WordIndex int `json:"wordIndex"`
Guesses int `json:"guesses"`
ElapsedMs int `json:"elapsedMs"`
}

type Store struct { db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) AlreadyPlayed(ctx context.Context, userID, date string) (bool, error) {
var cnt int
err := s.db.QueryRowContext(ctx,
"SELECT COUNT(1) FROM daily_results WHERE user_id=? AND date=?",
userID, date,
).Scan(&cnt)
return cnt > 0, err
}

func (s *Store) InsertResult(ctx context.Context, r Result) error {
_, err := s.db.ExecContext(ctx,
`INSERT OR IGNORE INTO daily_results(user_id, date, word_index, guesses, elapsed_ms)
VALUES(?,?,?,?,?)`, r.UserID, r.Date, r.WordIndex, r.Guesses, r.ElapsedMs,
)
return err
}

type LBRow struct {
UserID string `json:"userId"`
Guesses int `json:"guesses"`
ElapsedMs int `json:"elapsedMs"`
}

func (s *Store) Leaderboard(ctx context.Context, date string, limit int) ([]LBRow, error) {
rows, err := s.db.QueryContext(ctx,
`SELECT user_id, guesses, elapsed_ms
FROM daily_results
WHERE date=?
ORDER BY elapsed_ms ASC, guesses ASC, created_at ASC
LIMIT ?`, date, limit,
)
if err != nil { return nil, err }
defer rows.Close()
var out []LBRow
for rows.Next() {
var r LBRow
if err := rows.Scan(&r.UserID, &r.Guesses, &r.ElapsedMs); err != nil { return nil, err }
out = append(out, r)
}
return out, rows.Err()
}