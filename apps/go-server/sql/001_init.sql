-- apps/go-server/sql/001_init.sql
--
-- Initial database migration for the Wordle Go server.
-- Creates the `users` table, which stores authentication credentials
-- and basic game statistics, along with an index for efficient lookups.
--
-- Applied automatically by the Go server's migration system (db.go).
--
-- Schema notes:
--   • id            – primary key (string, UUID/crypto ID)
--   • username      – unique username, required
--   • password_hash – bcrypt or similar hash of the user's password
--   • created_at    – ISO 8601 / RFC3339 timestamp string
--   • games_played  – counter of total games played
--   • wins          – counter of total games won
--   • streak        – current win streak
--
-- Indexes:
--   • idx_users_username → ensures fast lookups by username.
--
-- WAL mode (Write-Ahead Logging) is enabled for concurrency.

PRAGMA journal_mode = WAL;

CREATE TABLE IF NOT EXISTS users (
  id            TEXT PRIMARY KEY,
  username      TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at    TEXT NOT NULL,
  games_played  INTEGER NOT NULL DEFAULT 0,
  wins          INTEGER NOT NULL DEFAULT 0,
  streak        INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
