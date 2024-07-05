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
	"github.com/zibbp/ganymede/internal/vod"
)

type Config struct {
	MAX_CHAT_DOWNLOAD_EXECUTIONS  int    `default:"5"`
	MAX_CHAT_RENDER_EXECUTIONS    int    `default:"3"`
	MAX_VIDEO_DOWNLOAD_EXECUTIONS int    `default:"5"`
	MAX_VIDEO_CONVERT_EXECUTIONS  int    `default:"3"`
	TEMPORAL_URL                  string `default:"temporal:7233"`
}

type Logger struct {
	logger *zerolog.Logger
}

func (l *Logger) Debug(msg string, keyvals ...interface{}) {
	if len(keyvals)%2 != 0 {
		l.logger.Debug().Msgf(msg)
		return
	}

	fields := make(map[string]interface{})
	for i := 0; i < len(keyvals); i += 2 {
		if key, ok := keyvals[i].(string); ok {
			fields[key] = keyvals[i+1]
		}
	}

	l.logger.Debug().Fields(fields).Msg(msg)
}

func (l *Logger) Info(msg string, keyvals ...interface{}) {
	if len(keyvals)%2 != 0 {
		l.logger.Info().Msgf(msg)
		return
	}

	fields := make(map[string]interface{})
	for i := 0; i < len(keyvals); i += 2 {
		if key, ok := keyvals[i].(string); ok {
			fields[key] = keyvals[i+1]
		}
	}

	l.logger.Info().Fields(fields).Msg(msg)
}

func (l *Logger) Warn(msg string, keyvals ...interface{}) {
	if len(keyvals)%2 != 0 {
		l.logger.Warn().Msgf(msg)
		return
	}

	fields := make(map[string]interface{})
	for i := 0; i < len(keyvals); i += 2 {
		if key, ok := keyvals[i].(string); ok {
			fields[key] = keyvals[i+1]
		}
	}

	l.logger.Warn().Fields(fields).Msg(msg)
}

func (l *Logger) Error(msg string, keyvals ...interface{}) {
	if len(keyvals)%2 != 0 {
		l.logger.Error().Msgf(msg)
		return
	}

	fields := make(map[string]interface{})
	for i := 0; i < len(keyvals); i += 2 {
		if key, ok := keyvals[i].(string); ok {
			fields[key] = keyvals[i+1]
		}
	}

	l.logger.Error().Fields(fields).Msg(msg)
}

func main() {
	ctx := context.Background()

	if os.Getenv("ENV") == "dev" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	// var config Config
	// err := envconfig.Process("", &config)
	// if err != nil {
	// 	log.Fatal().Msgf("Unable to process environment variables: %v", err)
	// }

	// log.Info().Msgf("Starting worker with config: %+v", config)

	// initializte main program config
	// this needs to be removed in the future to decouple the worker from the server
	serverConfig.NewConfig(false)

	// logger := zerolog.New(os.Stdout).With().Timestamp().Logger().With().Str("service", "worker").Logger()

	// clientOptions := client.Options{
	// 	HostPort: config.TEMPORAL_URL,
	// 	Logger:   &Logger{logger: &logger},
	// }

	// c, err := client.Dial(clientOptions)
	// if err != nil {
	// 	log.Fatal().Msgf("Unable to create client: %v", err)
	// }
	// defer c.Close()

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
	var platformService platform.PlatformService[platform_twitch.TwitchVideoInfo, platform_twitch.TwitchLivestreamInfo, platform_twitch.TwitchChannel]
	platformService, err = platform_twitch.NewTwitchPlatformService(
		envConfig.TwitchClientId,
		envConfig.TwitchClientSecret,
	)
	if err != nil {
		log.Panic().Err(err).Msg("Error creating platform service")
	}

	// initialize river
	riverWorkerClient, err := tasks_worker.NewRiverWorker(tasks_worker.RiverWorkerInput{
		DB_URL: dbString,
	}, db, platformService)
	if err != nil {
		log.Panic().Err(err).Msg("Error creating river worker")
	}

	// get periodic tasks
	periodicTasks := riverWorkerClient.GetPeriodicTasks(liveService)

	for _, task := range periodicTasks {
		riverWorkerClient.Client.PeriodicJobs().Add(task)
	}

	// Start your worker in a goroutine
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

	// // Initialize the temporal client for the worker
	// temporal.InitializeTemporalClient()

	// taskQueues := map[string]int{
	// 	"archive":        100,
	// 	"chat-download":  config.MAX_CHAT_DOWNLOAD_EXECUTIONS,
	// 	"chat-render":    config.MAX_CHAT_RENDER_EXECUTIONS,
	// 	"video-download": config.MAX_VIDEO_DOWNLOAD_EXECUTIONS,
	// 	"video-convert":  config.MAX_VIDEO_CONVERT_EXECUTIONS,
	// }

	// // create worker interrupt channel
	// interrupt := make(chan os.Signal, 1)

	// for queueName, maxActivites := range taskQueues {
	// 	hostname, err := os.Hostname()
	// 	if err != nil {
	// 		log.Fatal().Msgf("Unable to get hostname: %v", err)
	// 	}
	// 	// create workers
	// 	w := worker.New(c, queueName, worker.Options{
	// 		MaxConcurrentActivityExecutionSize: maxActivites,
	// 		Identity:                           hostname,
	// 		OnFatalError: func(err error) {
	// 			log.Error().Msgf("Worker encountered fatal error: %v", err)
	// 		},
	// 	})

	// 	w.RegisterWorkflow(workflows.ArchiveVideoWorkflow)
	// 	w.RegisterWorkflow(workflows.SaveTwitchVideoInfoWorkflow)
	// 	w.RegisterWorkflow(workflows.CreateDirectoryWorkflow)
	// 	w.RegisterWorkflow(workflows.DownloadTwitchThumbnailsWorkflow)
	// 	w.RegisterWorkflow(workflows.ArchiveTwitchVideoWorkflow)
	// 	w.RegisterWorkflow(workflows.DownloadTwitchVideoWorkflow)
	// 	w.RegisterWorkflow(workflows.PostprocessVideoWorkflow)
	// 	w.RegisterWorkflow(workflows.MoveVideoWorkflow)
	// 	w.RegisterWorkflow(workflows.ArchiveTwitchChatWorkflow)
	// 	w.RegisterWorkflow(workflows.DownloadTwitchChatWorkflow)
	// 	w.RegisterWorkflow(workflows.RenderTwitchChatWorkflow)
	// 	w.RegisterWorkflow(workflows.MoveTwitchChatWorkflow)
	// 	w.RegisterWorkflow(workflows.ArchiveLiveVideoWorkflow)
	// 	w.RegisterWorkflow(workflows.ArchiveTwitchLiveVideoWorkflow)
	// 	w.RegisterWorkflow(workflows.DownloadTwitchLiveChatWorkflow)
	// 	w.RegisterWorkflow(workflows.DownloadTwitchLiveThumbnailsWorkflow)
	// 	w.RegisterWorkflow(workflows.DownloadTwitchLiveThumbnailsWorkflowWait)
	// 	w.RegisterWorkflow(workflows.DownloadTwitchLiveVideoWorkflow)
	// 	w.RegisterWorkflow(workflows.SaveTwitchLiveVideoInfoWorkflow)
	// 	w.RegisterWorkflow(workflows.ArchiveTwitchLiveChatWorkflow)
	// 	w.RegisterWorkflow(workflows.ConvertTwitchLiveChatWorkflow)
	// 	w.RegisterWorkflow(workflows.SaveTwitchVideoChapters)
	// 	w.RegisterWorkflow(workflows.UpdateTwitchLiveStreamArchivesWithVodIds)

	// 	w.RegisterActivity(activities.ArchiveVideoActivity)
	// 	w.RegisterActivity(activities.SaveTwitchVideoInfo)
	// 	w.RegisterActivity(activities.CreateDirectory)
	// 	w.RegisterActivity(activities.DownloadTwitchThumbnails)
	// 	w.RegisterActivity(activities.DownloadTwitchVideo)
	// 	w.RegisterActivity(activities.PostprocessVideo)
	// 	w.RegisterActivity(activities.MoveVideo)
	// 	w.RegisterActivity(activities.DownloadTwitchChat)
	// 	w.RegisterActivity(activities.RenderTwitchChat)
	// 	w.RegisterActivity(activities.MoveChat)
	// 	w.RegisterActivity(activities.DownloadTwitchLiveChat)
	// 	w.RegisterActivity(activities.DownloadTwitchLiveThumbnails)
	// 	w.RegisterActivity(activities.DownloadTwitchLiveVideo)
	// 	w.RegisterActivity(activities.SaveTwitchLiveVideoInfo)
	// 	w.RegisterActivity(activities.KillTwitchLiveChatDownload)
	// 	w.RegisterActivity(activities.ConvertTwitchLiveChat)
	// 	w.RegisterActivity(activities.TwitchSaveVideoChapters)
	// 	w.RegisterActivity(activities.UpdateTwitchLiveStreamArchivesWithVodIds)

	// 	err = w.Start()
	// 	if err != nil {
	// 		log.Fatal().Msgf("Unable to start worker: %v", err)
	// 	}

	// }

	// <-interrupt

}
