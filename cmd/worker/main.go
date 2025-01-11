package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/blocked"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/config"
	serverConfig "github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/queue"
	tasks_client "github.com/zibbp/ganymede/internal/tasks/client"
	tasks_worker "github.com/zibbp/ganymede/internal/tasks/worker"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
)

func main() {
	ctx := context.Background()

	envConfig := config.GetEnvConfig()
	envAppConfig := config.GetEnvApplicationConfig()
	_, err := serverConfig.Init()
	if err != nil {
		log.Panic().Err(err).Msg("Error initializing server config")
	}

	if os.Getenv("DEVELOPMENT") == "true" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	log.Info().Str("commit", utils.Commit).Str("build_time", utils.BuildTime).Msg("starting worker")

	dbString := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=%s sslrootcert=%s", envAppConfig.DB_USER, envAppConfig.DB_PASS, envAppConfig.DB_HOST, envAppConfig.DB_PORT, envAppConfig.DB_NAME, envAppConfig.DB_SSL, envAppConfig.DB_SSL_ROOT_CERT)

	db := database.NewDatabase(ctx, database.DatabaseConnectionInput{
		DBString: dbString,
		IsWorker: false,
	})

	riverClient, err := tasks_client.NewRiverClient(tasks_client.RiverClientInput{
		DB_URL: dbString,
	})
	if err != nil {
		log.Panic().Err(err).Msg("Error creating river worker")
	}

	var platformTwitch platform.Platform
	// setup twitch platform
	if envConfig.TwitchClientId != "" && envConfig.TwitchClientSecret != "" {
		platformTwitch = &platform.TwitchConnection{
			ClientId:     envConfig.TwitchClientId,
			ClientSecret: envConfig.TwitchClientSecret,
		}
		_, err = platformTwitch.Authenticate(ctx)
		if err != nil {
			log.Panic().Err(err).Msg("Error authenticating to Twitch")
		}
	}

	channelService := channel.NewService(db, platformTwitch)
	vodService := vod.NewService(db, riverClient, platformTwitch)
	queueService := queue.NewService(db, vodService, channelService, riverClient)
	blockedVodsService := blocked.NewService(db)
	// twitchService := twitch.NewService()
	archiveService := archive.NewService(db, channelService, vodService, queueService, blockedVodsService, riverClient, platformTwitch)
	liveService := live.NewService(db, archiveService, platformTwitch)

	// initialize river
	riverWorkerClient, err := tasks_worker.NewRiverWorker(tasks_worker.RiverWorkerInput{
		DB_URL:                  dbString,
		DB:                      db,
		PlatformTwitch:          platformTwitch,
		VideoDownloadWorkers:    envConfig.MaxVideoDownloadExecutions,
		VideoPostProcessWorkers: envConfig.MaxVideoConvertExecutions,
		ChatDownloadWorkers:     envConfig.MaxChatDownloadExecutions,
		ChatRenderWorkers:       envConfig.MaxChatRenderExecutions,
		SpriteThumbnailWorkers:  envConfig.MaxVideoSpriteThumbnailExecutions,
	})
	if err != nil {
		log.Panic().Err(err).Msg("Error creating river worker")
	}

	// get periodic tasks
	periodicTasks, err := riverWorkerClient.GetPeriodicTasks(liveService)
	if err != nil {
		log.Panic().Err(err).Msg("Error getting periodic tasks")
	}

	for _, task := range periodicTasks {
		riverWorkerClient.Client.PeriodicJobs().Add(task)
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
