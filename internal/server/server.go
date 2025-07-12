package server

import (
	"context"
	"fmt"
	"log/slog"
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
	"github.com/zibbp/ganymede/internal/task"
	tasks_client "github.com/zibbp/ganymede/internal/tasks/client"
	transportHttp "github.com/zibbp/ganymede/internal/transport/http"
	"github.com/zibbp/ganymede/internal/user"
	"github.com/zibbp/ganymede/internal/vod"
	"riverqueue.com/riverui"
)

type Application struct {
	EnvConfig         config.EnvConfig
	Database          *database.Database
	Store             *database.Database
	ArchiveService    *archive.Service
	PlatformTwitch    platform.Platform
	AdminService      *admin.Service
	AuthService       *auth.Service
	ChannelService    *channel.Service
	VodService        *vod.Service
	QueueService      *queue.Service
	UserService       *user.Service
	LiveService       *live.Service
	PlaybackService   *playback.Service
	MetricsService    *metrics.Service
	PlaylistService   *playlist.Service
	TaskService       *task.Service
	ChapterService    *chapter.Service
	CategoryService   *category.Service
	BlockedVodService *blocked.Service
	RiverUIServer     *riverui.Server
	RiverClient       *tasks_client.RiverClient
}

func SetupApplication(ctx context.Context) (*Application, error) {
	envConfig := config.GetEnvConfig()
	envAppConfig := config.GetEnvApplicationConfig()
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

	// Disable logging for tests
	if os.Getenv("TESTS_LOGGING") == "false" {
		zerolog.SetGlobalLevel(zerolog.Disabled)
	}

	dbString := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=%s sslrootcert=%s", envAppConfig.DB_USER, envAppConfig.DB_PASS, envAppConfig.DB_HOST, envAppConfig.DB_PORT, envAppConfig.DB_NAME, envAppConfig.DB_SSL, envAppConfig.DB_SSL_ROOT_CERT)

	db := database.NewDatabase(ctx, database.DatabaseConnectionInput{
		DBString: dbString,
		IsWorker: false,
	})

	// application migrations
	if envConfig.PathMigrationEnabled {
		// check if VideosDir changed
		if err := db.VideosDirMigrate(ctx, envConfig.VideosDir); err != nil {
			return nil, fmt.Errorf("error migrating videos dir: %v", err)
		}
		if err := db.TempDirMigrate(ctx, envConfig.TempDir); err != nil {
			return nil, fmt.Errorf("error migrating videos dir: %v", err)
		}
	}

	// Initialize river client
	riverClient, err := tasks_client.NewRiverClient(tasks_client.RiverClientInput{
		DB_URL: dbString,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating river client: %v", err)
	}

	err = riverClient.RunMigrations()
	if err != nil {
		return nil, fmt.Errorf("error running migrations: %v", err)
	}

	// Setup RiverUI server
	riverUIOpts := &riverui.ServerOpts{
		Client: riverClient.Client,
		DB:     riverClient.PgxPool,
		Logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
		Prefix: "/riverui",
	}
	riverUIServer, err := riverui.NewServer(riverUIOpts)
	if err != nil {
		return nil, fmt.Errorf("error creating riverui server: %v", err)
	}

	go func() {
		if err := riverUIServer.Start(ctx); err != nil {
			log.Error().Err(err).Msg("error running riverui server")
		}
	}()

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

	authService := auth.NewService(db, &envConfig)
	channelService := channel.NewService(db, platformTwitch)
	vodService := vod.NewService(db, riverClient, platformTwitch)
	queueService := queue.NewService(db, vodService, channelService, riverClient)
	blockedVodService := blocked.NewService(db)
	archiveService := archive.NewService(db, channelService, vodService, queueService, blockedVodService, riverClient, platformTwitch)
	adminService := admin.NewService(db)
	userService := user.NewService(db)
	chapterService := chapter.NewService(db)
	liveService := live.NewService(db, archiveService, platformTwitch, chapterService)
	playbackService := playback.NewService(db)
	metricsService := metrics.NewService(db, riverClient)
	playlistService := playlist.NewService(db)
	taskService := task.NewService(db, liveService, riverClient)
	categoryService := category.NewService(db)

	return &Application{
		EnvConfig:         envConfig,
		Database:          db,
		AuthService:       authService,
		ChannelService:    channelService,
		VodService:        vodService,
		QueueService:      queueService,
		BlockedVodService: blockedVodService,
		ArchiveService:    archiveService,
		AdminService:      adminService,
		UserService:       userService,
		LiveService:       liveService,
		PlaybackService:   playbackService,
		MetricsService:    metricsService,
		PlaylistService:   playlistService,
		TaskService:       taskService,
		ChapterService:    chapterService,
		CategoryService:   categoryService,
		PlatformTwitch:    platformTwitch,
		RiverUIServer:     riverUIServer,
		RiverClient:       riverClient,
	}, nil
}

func Run(ctx context.Context) error {

	app, err := SetupApplication(ctx)
	if err != nil {
		return err
	}

	httpHandler := transportHttp.NewHandler(app.Database, app.AuthService, app.ChannelService, app.VodService, app.QueueService, app.ArchiveService, app.AdminService, app.UserService, app.LiveService, app.PlaybackService, app.MetricsService, app.PlaylistService, app.TaskService, app.ChapterService, app.CategoryService, app.BlockedVodService, app.PlatformTwitch, app.RiverUIServer)

	if err := httpHandler.Serve(ctx); err != nil {
		return err
	}

	return nil
}
