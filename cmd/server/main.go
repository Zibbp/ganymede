package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/zibbp/ganymede/internal/admin"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/auth"
	"github.com/zibbp/ganymede/internal/blocked"
	"github.com/zibbp/ganymede/internal/category"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/chapter"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
	_ "github.com/zibbp/ganymede/internal/kv"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/metrics"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/playback"
	"github.com/zibbp/ganymede/internal/playlist"
	"github.com/zibbp/ganymede/internal/queue"
	"github.com/zibbp/ganymede/internal/scheduler"
	"github.com/zibbp/ganymede/internal/task"
	tasks_client "github.com/zibbp/ganymede/internal/tasks/client"
	transportHttp "github.com/zibbp/ganymede/internal/transport/http"
	"github.com/zibbp/ganymede/internal/user"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
)

//	@title			Ganymede API
//	@version		1.0
//	@description	Authentication is handled using JWT tokens. The tokens are set as access-token and refresh-token cookies.
//	@description	For information regarding which role is authorized for which endpoint, see the http handler https://github.com/Zibbp/ganymede/blob/main/internal/transport/http/handler.go.

//	@contact.name	Zibbp
//	@contact.url	https://github.com/zibbp/ganymede

//	@license.name	GPL-3.0

//	@host		localhost:4000
//	@BasePath	/api/v1

//	@securityDefinitions.apikey	ApiKeyCookieAuth
//	@in							cookie
//	@name						access-token

//	@securityDefinitions.refreshToken	ApiKeyCookieRefresh
//	@in									cookie
//	@name								refresh-token

func Run() error {
	ctx := context.Background()

	envConfig := config.GetEnvConfig()
	_, err := config.Init()
	if err != nil {
		log.Panic().Err(err).Msg("error getting config")
	}

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	if envConfig.DEBUG {
		log.Info().Msg("debug mode enabled")
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	dbString := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=%s", envConfig.DB_USER, envConfig.DB_PASS, envConfig.DB_HOST, envConfig.DB_PORT, envConfig.DB_NAME, envConfig.DB_SSL)

	db := database.NewDatabase(ctx, database.DatabaseConnectionInput{
		DBString: dbString,
		IsWorker: false,
	})

	// application migrations
	// check if VideosDir changed
	if err := db.VideosDirMigrate(ctx, envConfig.VideosDir); err != nil {
		return fmt.Errorf("error migrating videos dir: %v", err)
	}
	if err := db.TempDirMigrate(ctx, envConfig.TempDir); err != nil {
		return fmt.Errorf("error migrating videos dir: %v", err)
	}

	// Initialize river client
	riverClient, err := tasks_client.NewRiverClient(tasks_client.RiverClientInput{
		DB_URL: dbString,
	})
	if err != nil {
		return fmt.Errorf("error creating river client: %v", err)
	}

	err = riverClient.RunMigrations()
	if err != nil {
		return fmt.Errorf("error running migrations: %v", err)
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

	_, err = platformTwitch.GetVideo(ctx, "2200478055", true, true)
	if err != nil {
		log.Panic().Err(err).Msg("Error authenticating to Twitch")
	}

	authService := auth.NewService(db)
	channelService := channel.NewService(db, platformTwitch)
	vodService := vod.NewService(db, platformTwitch)
	queueService := queue.NewService(db, vodService, channelService, riverClient)
	blockedVodService := blocked.NewService(db)
	archiveService := archive.NewService(db, channelService, vodService, queueService, blockedVodService, riverClient, platformTwitch)
	adminService := admin.NewService(db)
	userService := user.NewService(db)
	// configService := config.NewService(db)
	liveService := live.NewService(db, archiveService, platformTwitch)
	schedulerService := scheduler.NewService(liveService, archiveService)
	playbackService := playback.NewService(db)
	metricsService := metrics.NewService(db, riverClient)
	playlistService := playlist.NewService(db)
	taskService := task.NewService(db, liveService, riverClient)
	chapterService := chapter.NewService(db)
	categoryService := category.NewService(db)

	httpHandler := transportHttp.NewHandler(authService, channelService, vodService, queueService, archiveService, adminService, userService, liveService, schedulerService, playbackService, metricsService, playlistService, taskService, chapterService, categoryService, blockedVodService, platformTwitch)

	if err := httpHandler.Serve(); err != nil {
		return err
	}

	return nil
}

func main() {
	if os.Getenv("ENV") == "dev" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	log.Info().Str("commit", utils.Commit).Str("build_time", utils.BuildTime).Msg("starting server")
	if err := Run(); err != nil {
		log.Fatal().Err(err).Msg("failed to run")
	}
}
