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
	// Load .env if present
	_ = godotenv.Load()

	// Logging level (LOG_LEVEL=debug|info|warn|error)
	if lvl, err := zerolog.ParseLevel(getEnv("LOG_LEVEL", "info")); err == nil {
		zerolog.SetGlobalLevel(lvl)
	}

	// Game dictionaries
	if err := words.Init(); err != nil {
		log.Fatal().Err(err).Msg("failed to load word lists")
	}

	// DB (expects users table per earlier migration)
	db, err := openDB(envStr("DATABASE_URL", "./data/app.db"))
	if err != nil {
		log.Fatal().Err(err).Msg("openDB failed")
	}
	defer db.Close()
	if err := migrate(db); err != nil {
		log.Fatal().Err(err).Msg("migrate failed")
	}

	// Game state store
	mem := store.NewMemoryStore()

	// Single server hosting game + auth routes
	srv := httpserver.New(mem, db)

	addr := ":" + envStr("PORT", "3000")
	log.Info().
		Str("addr", addr).
		Str("client_origin", envStr("CLIENT_ORIGIN", "http://localhost:5173")).
		Msg("go-server listening")
	if err := srv.Start(addr); err != nil {
		log.Fatal().Err(err).Msg("server exited")
	}
}

func envStr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
func getEnv(k, def string) string { return envStr(k, def) }
