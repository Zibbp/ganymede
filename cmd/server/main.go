package main

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	_ "github.com/zibbp/ganymede/internal/kv"
	"github.com/zibbp/ganymede/internal/server"
	"github.com/zibbp/ganymede/internal/utils"
)

func main() {
	ctx := context.Background()

	if os.Getenv("development") == "true" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	log.Info().Str("commit", utils.Commit).Str("build_time", utils.BuildTime).Msg("starting server")
	if err := server.Run(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to run")
	}
}
