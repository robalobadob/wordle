PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;

/* Recreate `games` with user_id NULLABLE and a CHECK to ensure at least one owner exists */
CREATE TABLE games_new (
  id           TEXT PRIMARY KEY,
  user_id      TEXT,                       -- now NULLABLE
  anonymous_id TEXT,                       -- guest owner
  answer       TEXT NOT NULL,
  started_at   TEXT NOT NULL,
  finished_at  TEXT,
  status       TEXT NOT NULL DEFAULT 'playing', -- 'playing' | 'won' | 'lost'
  guesses      INTEGER NOT NULL DEFAULT 0,
  CHECK (user_id IS NOT NULL OR anonymous_id IS NOT NULL),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

/* Copy data from old table; existing rows have user_id, no anonymous_id */
INSERT INTO games_new (id, user_id, anonymous_id, answer, started_at, finished_at, status, guesses)
SELECT id, user_id, NULL, answer, started_at, finished_at, status, guesses
FROM games;

/* Swap tables */
DROP TABLE games;
ALTER TABLE games_new RENAME TO games;

/* Recreate indexes */
CREATE INDEX IF NOT EXISTS idx_games_user_id  ON games(user_id);
CREATE INDEX IF NOT EXISTS idx_games_status   ON games(status);
CREATE INDEX IF NOT EXISTS idx_games_anon     ON games(anonymous_id);

COMMIT;
PRAGMA foreign_keys=ON;
