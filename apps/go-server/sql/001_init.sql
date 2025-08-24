PRAGMA journal_mode = WAL;

CREATE TABLE IF NOT EXISTS users (
  id          TEXT PRIMARY KEY,
  username    TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at  TEXT NOT NULL,
  games_played INTEGER NOT NULL DEFAULT 0,
  wins         INTEGER NOT NULL DEFAULT 0,
  streak       INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
