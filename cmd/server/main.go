package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/spf13/viper"
	"github.com/zibbp/ganymede/internal/admin"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/auth"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/playback"
	"github.com/zibbp/ganymede/internal/queue"
	"github.com/zibbp/ganymede/internal/scheduler"
	transportHttp "github.com/zibbp/ganymede/internal/transport/http"
	"github.com/zibbp/ganymede/internal/twitch"
	"github.com/zibbp/ganymede/internal/user"
	"github.com/zibbp/ganymede/internal/vod"
)

func Run() error {

	config.NewConfig()

	configDebug := viper.GetBool("debug")
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	if configDebug {
		log.Info().Msg("debug mode enabled")
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	store, err := database.NewDatabase()
	if err != nil {
		log.Error().Err(err).Msg("failed to create database connection")
		return err
	}

	authService := auth.NewService(store)
	channelService := channel.NewService(store)
	vodService := vod.NewService(store)
	queueService := queue.NewService(store, vodService, channelService)
	twitchService := twitch.NewService()
	archiveService := archive.NewService(store, twitchService, channelService, vodService, queueService)
	adminService := admin.NewService(store)
	userService := user.NewService(store)
	configService := config.NewService(store)
	liveService := live.NewService(store, twitchService, archiveService)
	schedulerService := scheduler.NewService(liveService, archiveService)
	playbackService := playback.NewService(store)

	httpHandler := transportHttp.NewHandler(authService, channelService, vodService, queueService, twitchService, archiveService, adminService, userService, configService, liveService, schedulerService, playbackService)

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
