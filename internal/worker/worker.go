package worker

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/blocked"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/chapter"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/queue"
	tasks_client "github.com/zibbp/ganymede/internal/tasks/client"
	tasks_worker "github.com/zibbp/ganymede/internal/tasks/worker"
	"github.com/zibbp/ganymede/internal/vod"
)

// SetupWorker sets up the worker
func SetupWorker(ctx context.Context) (*tasks_worker.RiverWorkerClient, error) {
	envConfig := config.GetEnvConfig()
	envAppConfig := config.GetEnvApplicationConfig()
	_, err := config.Init()
	if err != nil {
		return nil, err
	}

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	if envConfig.DEBUG {
		log.Info().Msg("debug mode enabled")
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Disable logging for tests
	if os.Getenv("TESTS_LOGGING") == "false" {
		zerolog.SetGlobalLevel(zerolog.Disabled)
	}

	dbString := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=%s sslrootcert=%s", envAppConfig.DB_USER, envAppConfig.DB_PASS, envAppConfig.DB_HOST, envAppConfig.DB_PORT, envAppConfig.DB_NAME, envAppConfig.DB_SSL, envAppConfig.DB_SSL_ROOT_CERT)

	db := database.NewDatabase(ctx, database.DatabaseConnectionInput{
		DBString: dbString,
		IsWorker: false,
	})

	riverClient, err := tasks_client.NewRiverClient(tasks_client.RiverClientInput{
		DB_URL: dbString,
	})
	if err != nil {
		return nil, err
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
			return nil, err
		}
	}

	chapterService := chapter.NewService(db)
	channelService := channel.NewService(db, platformTwitch)
	vodService := vod.NewService(db, riverClient, platformTwitch)
	queueService := queue.NewService(db, vodService, channelService, riverClient)
	blockedVodsService := blocked.NewService(db)
	// twitchService := twitch.NewService()
	archiveService := archive.NewService(db, channelService, vodService, queueService, blockedVodsService, riverClient, platformTwitch)
	liveService := live.NewService(db, archiveService, platformTwitch, chapterService)

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
		return nil, err
	}

	// get periodic tasks
	periodicTasks, err := riverWorkerClient.GetPeriodicTasks(liveService)
	if err != nil {
		log.Panic().Err(err).Msg("Error getting periodic tasks")
	}

	for _, task := range periodicTasks {
		riverWorkerClient.Client.PeriodicJobs().Add(task)
	}

	return riverWorkerClient, nil
}
