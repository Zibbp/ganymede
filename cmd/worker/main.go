package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/worker"
)

const workerShutdownTimeout = 300*time.Second + 5*time.Second

func main() {
	ctx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	if os.Getenv("DEVELOPMENT") == "true" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	log.Info().Str("commit", utils.Commit).Str("tag", utils.Tag).Str("build_time", utils.BuildTime).Msg("starting worker")

	// Set up the worker
	riverWorkerClient, err := worker.SetupWorker(ctx)
	if err != nil {
		log.Panic().Err(err).Msg("Error setting up river worker")
	}

	defer func() {
		if err := riverWorkerClient.Close(); err != nil {
			log.Error().Err(err).Msg("error closing worker resources")
		}
	}()

	if err := riverWorkerClient.Start(); err != nil {
		log.Panic().Err(err).Msg("Error running river worker")
	}

	<-ctx.Done()

	// Gracefully stop the worker
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), workerShutdownTimeout)
	defer cancelShutdown()
	if err := riverWorkerClient.Stop(shutdownCtx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Warn().Err(err).Msg("worker shutdown timed out; exiting")
		} else {
			log.Panic().Err(err).Msg("Error stopping river worker")
		}
	}

	log.Info().Msg("worker stopped")
}
