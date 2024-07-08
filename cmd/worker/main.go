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
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/config"
	serverConfig "github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/platform"
	platform_twitch "github.com/zibbp/ganymede/internal/platform/twitch"
	"github.com/zibbp/ganymede/internal/queue"
	tasks_client "github.com/zibbp/ganymede/internal/tasks/client"
	tasks_worker "github.com/zibbp/ganymede/internal/tasks/worker"
	"github.com/zibbp/ganymede/internal/twitch"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
)

func main() {
	ctx := context.Background()

	if os.Getenv("ENV") == "dev" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	log.Info().Str("commit", utils.Commit).Str("build_time", utils.BuildTime).Msg("starting worker")

	serverConfig.NewConfig(false)

	// authenticate to Twitch
	err := twitch.Authenticate()
	if err != nil {
		log.Fatal().Msgf("Unable to authenticate to Twitch: %v", err)
	}

	envConfig := config.GetEnvConfig()

	dbString := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=%s", envConfig.DB_USER, envConfig.DB_PASS, envConfig.DB_HOST, envConfig.DB_PORT, envConfig.DB_NAME, envConfig.DB_SSL)

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

	channelService := channel.NewService(db)
	vodService := vod.NewService(db)
	queueService := queue.NewService(db, vodService, channelService, riverClient)
	twitchService := twitch.NewService()
	archiveService := archive.NewService(db, channelService, vodService, queueService, riverClient)
	liveService := live.NewService(db, twitchService, archiveService)

	// create platform service
	var platformService platform.PlatformService[platform_twitch.TwitchVideoInfo, platform_twitch.TwitchLivestreamInfo, platform_twitch.TwitchChannel, platform_twitch.TwitchCategory]
	platformService, err = platform_twitch.NewTwitchPlatformService(
		envConfig.TwitchClientId,
		envConfig.TwitchClientSecret,
	)
	if err != nil {
		log.Panic().Err(err).Msg("Error creating platform service")
	}

	// initialize river
	riverWorkerClient, err := tasks_worker.NewRiverWorker(tasks_worker.RiverWorkerInput{
		DB_URL:                  dbString,
		DB:                      db,
		PlatformService:         platformService,
		VideoDownloadWorkers:    envConfig.MaxVideoDownloadExecutions,
		VideoPostProcessWorkers: envConfig.MaxVideoConvertExecutions,
		ChatDownloadWorkers:     envConfig.MaxChatDownloadExecutions,
		ChatRenderWorkers:       envConfig.MaxChatRenderExecutions,
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
