package main

import (
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"

	"github.com/robalobadob/wordle/apps/go-server/internal/httpserver"
	"github.com/robalobadob/wordle/apps/go-server/internal/store"
)

func main() {
	_ = godotenv.Load()
	level, err := zerolog.ParseLevel(getEnv("LOG_LEVEL", "info"))
	if err == nil { zerolog.SetGlobalLevel(level) }

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
