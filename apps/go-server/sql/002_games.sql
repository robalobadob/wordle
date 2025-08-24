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
