-- apps/go-server/sql/002_games.sql
--
-- Migration #2: Create the `games` table.
-- Stores per-game session data for each user.
--
-- Schema notes:
--   • id          – primary key (string, UUID/crypto ID)
--   • user_id     – foreign key → users.id (each game belongs to one user)
--   • answer      – the correct solution word (may be hidden/blank in some contexts)
--   • started_at  – ISO 8601 / RFC3339 timestamp when game began
--   • finished_at – timestamp when game ended (NULL if still in progress)
--   • status      – enum string: 'playing' | 'won' | 'lost'
--   • guesses     – number of guesses made so far
--
-- Constraints:
--   • FOREIGN KEY (user_id) → users(id), cascades on delete (user deletion removes their games).
--
-- Indexes:
--   • idx_games_user_id → accelerates lookups of a user’s games
--   • idx_games_status  → accelerates filtering by game state (active, won, lost)

CREATE TABLE IF NOT EXISTS games (
  id          TEXT PRIMARY KEY,
  user_id     TEXT NOT NULL,
  answer      TEXT NOT NULL,
  started_at  TEXT NOT NULL,
  finished_at TEXT,
  status      TEXT NOT NULL DEFAULT 'playing', -- 'playing' | 'won' | 'lost'
  guesses     INTEGER NOT NULL DEFAULT 0,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_games_user_id ON games(user_id);
CREATE INDEX IF NOT EXISTS idx_games_status ON games(status);
