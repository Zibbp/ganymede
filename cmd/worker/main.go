package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/worker"
)

func main() {
	ctx := context.Background()

	if os.Getenv("DEVELOPMENT") == "true" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	log.Info().Str("commit", utils.Commit).Str("tag", utils.Tag).Str("build_time", utils.BuildTime).Msg("starting worker")

	// Set up the worker
	riverWorkerClient, err := worker.SetupWorker(ctx)
	if err != nil {
		log.Panic().Err(err).Msg("Error setting up river worker")
	}

	// start worker in a goroutine
	go func() {
		if err := riverWorkerClient.Start(); err != nil {
			log.Panic().Err(err).Msg("Error running river worker")
		}
	}()

	// Set up channel to listen for OS signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	<-sigs

	// Gracefully stop the worker
	if err := riverWorkerClient.Stop(); err != nil {
		log.Panic().Err(err).Msg("Error stopping river worker")
	}

	log.Info().Msg("worker stopped")
}
