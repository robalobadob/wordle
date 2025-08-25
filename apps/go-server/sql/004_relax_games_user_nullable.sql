-- apps/go-server/sql/004_relax_games_user_nullable.sql
--
-- Migration #4: Relax `games.user_id` constraint to support guest ownership.
--
-- Context:
--   • In migration #002, every game required a `user_id`.
--   • Migration #003 introduced `anonymous_id` but `user_id` was still NOT NULL,
--     so the schema could not represent true guest-only games.
--   • This migration makes `user_id` nullable and adds a CHECK constraint to ensure
--     that at least one owner field (`user_id` OR `anonymous_id`) is present.
--
-- Implementation notes:
--   • SQLite doesn’t support dropping constraints in-place.
--     → We rebuild the table:
--        1. Temporarily disable foreign_keys.
--        2. Create a new `games_new` table with desired schema.
--        3. Copy existing data from old `games` (user-owned).
--        4. Drop old table and rename new one.
--        5. Recreate indexes.
--        6. Re-enable foreign_keys.
--
-- Schema differences:
--   • user_id is now nullable.
--   • anonymous_id remains nullable.
--   • CHECK constraint enforces (user_id IS NOT NULL OR anonymous_id IS NOT NULL).
--
-- Result:
--   • Games can be owned by either:
--       - a registered user (user_id filled, anonymous_id NULL), OR
--       - a guest session (anonymous_id filled, user_id NULL).
--   • Server logic (see `claimAnonGames` in httpserver/server.go)
--     can migrate anonymous games to a registered user after signup.

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
