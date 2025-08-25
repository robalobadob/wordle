-- apps/go-server/sql/daily_results.sql
--
-- Migration: Create `daily_results` table
-- Stores one record per user per day for the Daily Challenge mode.
--
-- Schema notes:
--   • id          – surrogate primary key (auto-increment integer)
--   • user_id     – owner of the result (registered or guest/anon mapped ID)
--   • date        – day of the challenge in "YYYY-MM-DD" (UTC or chosen tz boundary)
--   • word_index  – index of the chosen answer from the canonical answers list
--   • guesses     – number of guesses taken before finishing (if finished)
--   • elapsed_ms  – time taken (milliseconds) from game start to completion
--   • created_at  – timestamp of when result was recorded (defaults to now)
--
-- Constraints:
--   • UNIQUE(user_id, date) → ensures one result per user per day.
--
-- Indexes:
--   • idx_daily_results_date
--       - Accelerates queries for all results on a given day.
--   • idx_daily_results_date_time
--       - Optimizes leaderboard queries sorted by elapsed_ms for a given date.
--
-- Usage:
--   - Inserted when a user wins their daily game.
--   - Queried by leaderboard API (/daily/leaderboard).
--   - Enforced uniqueness prevents replaying the same day multiple times.

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
