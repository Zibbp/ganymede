package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/zibbp/ganymede/internal/admin"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/auth"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/queue"
	transportHttp "github.com/zibbp/ganymede/internal/transport/http"
	"github.com/zibbp/ganymede/internal/twitch"
	"github.com/zibbp/ganymede/internal/vod"
)

func Run() error {

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	store, err := database.NewDatabase()
	if err != nil {
		log.Error().Err(err).Msg("failed to create database connection")
		return err
	}

	authService := auth.NewService(store)
	channelService := channel.NewService(store)
	vodService := vod.NewService(store)
	queueService := queue.NewService(store, vodService, channelService)

	twitchService := twitch.NewService(store)
	archiveService := archive.NewService(store, twitchService, channelService, vodService, queueService)
	adminService := admin.NewService(store)

	httpHandler := transportHttp.NewHandler(authService, channelService, vodService, queueService, twitchService, archiveService, adminService)

	if err := httpHandler.Serve(); err != nil {
		return err
	}

	return nil
}

func main() {
	if err := Run(); err != nil {
		log.Fatal().Err(err).Msg("failed to run")
	}
}
