-- Daily Challenge results (one per user/day)
CREATE TABLE IF NOT EXISTS daily_results (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id TEXT NOT NULL,
  date TEXT NOT NULL,              -- YYYY-MM-DD (UTC or chosen tz)
  word_index INTEGER NOT NULL,     -- index into answers list
  guesses INTEGER NOT NULL,
  elapsed_ms INTEGER NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(user_id, date)
);

CREATE INDEX IF NOT EXISTS idx_daily_results_date        ON daily_results(date);
CREATE INDEX IF NOT EXISTS idx_daily_results_date_time   ON daily_results(date, elapsed_ms);
