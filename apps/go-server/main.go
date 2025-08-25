// apps/go-server/main.go
//
// Entry point for the Wordle Go backend server.
// Responsibilities:
//   - Load environment variables (from .env and process).
//   - Configure logging (zerolog).
//   - Initialize word lists (allowed guesses + answers).
//   - Open and migrate SQLite/Postgres database.
//   - Create an in-memory game state store.
//   - Start HTTP server exposing game + auth routes.

package main

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/robalobadob/wordle/apps/go-server/internal/httpserver"
	"github.com/robalobadob/wordle/apps/go-server/internal/store"
	"github.com/robalobadob/wordle/apps/go-server/internal/words"
)

func main() {
	// Load .env file if present (non-fatal if missing).
	_ = godotenv.Load()

	// Configure logging level (LOG_LEVEL=debug|info|warn|error).
	if lvl, err := zerolog.ParseLevel(getEnv("LOG_LEVEL", "info")); err == nil {
		zerolog.SetGlobalLevel(lvl)
	}

	// Initialize dictionaries of allowed/answer words.
	if err := words.Init(); err != nil {
		log.Fatal().Err(err).Msg("failed to load word lists")
	}

	// Open DB connection (defaults to ./data/app.db if DATABASE_URL not set).
	// DB should already have "users" table from earlier migrations.
	db, err := openDB(envStr("DATABASE_URL", "./data/app.db"))
	if err != nil {
		log.Fatal().Err(err).Msg("openDB failed")
	}
	defer db.Close()

	// Apply schema migrations.
	if err := migrate(db); err != nil {
		log.Fatal().Err(err).Msg("migrate failed")
	}

	// Create in-memory store for active game state (per-process only).
	mem := store.NewMemoryStore()

	// Construct HTTP server with memory store + database.
	srv := httpserver.New(mem, db)

	// Server listen address (defaults to :3000).
	addr := ":" + envStr("PORT", "3000")

	// Log startup details including client origin (for CORS).
	log.Info().
		Str("addr", addr).
		Str("client_origin", envStr("CLIENT_ORIGIN", "http://localhost:5173")).
		Msg("go-server listening")

	// Start blocking server loop. Exit fatally if it stops.
	if err := srv.Start(addr); err != nil {
		log.Fatal().Err(err).Msg("server exited")
	}
}

// envStr returns the value of env var k, or def if unset/empty.
func envStr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// getEnv is an alias for envStr (kept for compatibility).
func getEnv(k, def string) string { return envStr(k, def) }
