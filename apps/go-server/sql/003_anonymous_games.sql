-- apps/go-server/sql/003_anonymous_games.sql
--
-- Migration #3: Extend `games` table to support anonymous players.
--
-- Context:
--   In earlier migrations, every game was tied to an authenticated `user_id`.
--   To allow guests (non-registered users) to play and later claim their history,
--   we add an `anonymous_id` column. This is typically a random cookie-based ID
--   managed by the server (`ensureAnonID` in httpserver/server.go).
--
-- Schema changes:
--   • anonymous_id – optional text field (nullable)
--       - Populated for guest games
--       - Cleared if/when a guest later signs up/logs in and their games are claimed
--
-- Indexes:
--   • idx_games_anon → accelerates lookups by anonymous_id when resolving guest sessions.
--
-- Notes:
--   • The table now supports two ownership modes:
--       - Registered: user_id is set, anonymous_id is NULL
--       - Guest: anonymous_id is set, user_id may be NULL
--   • Claiming logic migrates games from anonymous_id → user_id.

ALTER TABLE games ADD COLUMN anonymous_id TEXT;

CREATE INDEX IF NOT EXISTS idx_games_anon ON games(anonymous_id);
