package main

import (
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/robalobadob/wordle/apps/go-server/internal/httpserver"
	"github.com/robalobadob/wordle/apps/go-server/internal/store"
	"github.com/robalobadob/wordle/apps/go-server/internal/words"
	"os"
)

func main() {
	_ = godotenv.Load()
	if lvl, err := zerolog.ParseLevel(getEnv("LOG_LEVEL", "info")); err == nil {
		zerolog.SetGlobalLevel(lvl)
	}

	if err := words.Init(); err != nil {
		log.Fatal().Err(err).Msg("failed to load word lists")
	}

	mem := store.NewMemoryStore()
	srv := httpserver.New(mem)
	port := getEnv("PORT", "5175")
	log.Info().Str("port", port).Msg("starting go-server")
	if err := srv.Start(":" + port); err != nil {
		log.Fatal().Err(err).Msg("server exited")
	}
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}
