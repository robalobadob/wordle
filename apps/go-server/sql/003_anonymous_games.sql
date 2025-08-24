ALTER TABLE games ADD COLUMN anonymous_id TEXT;

CREATE INDEX IF NOT EXISTS idx_games_anon ON games(anonymous_id);
