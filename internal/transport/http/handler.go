package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	session "github.com/canidam/echo-scs-session"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	echoSwagger "github.com/swaggo/echo-swagger"
	_ "github.com/zibbp/ganymede/docs"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/utils"
	"riverqueue.com/riverui"
)

type Services struct {
	AuthService         AuthService
	ChannelService      ChannelService
	VodService          VodService
	QueueService        QueueService
	ArchiveService      ArchiveService
	AdminService        AdminService
	UserService         UserService
	LiveService         LiveService
	PlaybackService     PlaybackService
	MetricsService      MetricsService
	PlaylistService     PlaylistService
	TaskService         TaskService
	ChapterService      ChapterService
	CategoryService     CategoryService
	BlockedVideoService BlockedVideoService
	PlatformTwitch      platform.Platform
}

type Handler struct {
	Server         *echo.Echo
	Service        Services
	SessionManager *scs.SessionManager
	RiverUIServer  *riverui.Server
}

var sessionManager *scs.SessionManager

func NewHandler(database *database.Database, authService AuthService, channelService ChannelService, vodService VodService, queueService QueueService, archiveService ArchiveService, adminService AdminService, userService UserService, liveService LiveService, playbackService PlaybackService, metricsService MetricsService, playlistService PlaylistService, taskService TaskService, chapterService ChapterService, categoryService CategoryService, blockedVideoService BlockedVideoService, platformTwitch platform.Platform, riverUIServer *riverui.Server) *Handler {
	log.Debug().Msg("creating route handler")
	envConfig := config.GetEnvConfig()

	sessionManager = scs.New()
	sessionManager.Store = pgxstore.New(database.ConnPool)
	// 30 days session lifetime
	sessionManager.Lifetime = (24 * time.Hour) * 30
	// Expire session if no activity for 7 days
	sessionManager.IdleTimeout = (24 * time.Hour) * 7

	h := &Handler{
		Server: echo.New(),
		Service: Services{
			AuthService:         authService,
			ChannelService:      channelService,
			VodService:          vodService,
			QueueService:        queueService,
			ArchiveService:      archiveService,
			AdminService:        adminService,
			UserService:         userService,
			LiveService:         liveService,
			PlaybackService:     playbackService,
			MetricsService:      metricsService,
			PlaylistService:     playlistService,
			TaskService:         taskService,
			ChapterService:      chapterService,
			CategoryService:     categoryService,
			BlockedVideoService: blockedVideoService,
			PlatformTwitch:      platformTwitch,
		},
		SessionManager: sessionManager,
		RiverUIServer:  riverUIServer,
	}

	// Enable gzip compression for API routes
	h.Server.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Skipper: func(c echo.Context) bool {
			return !strings.Contains(c.Request().URL.Path, "/api")
		},
	}))

	// Use sessions
	h.Server.Use(session.LoadAndSave(sessionManager))

	// Middleware
	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	h.Server.HideBanner = true

	// If frontend is external then allow cors
	h.Server.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowMethods:     []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
		AllowCredentials: true,
	}))

	// Enable request logging in debug
	if envConfig.DEBUG {
		logger := log.Logger
		h.Server.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
			LogURI:    true,
			LogStatus: true,
			LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
				if !strings.Contains(v.URI, "/api") {
					return nil
				}
				logger.Info().
					Str("URI", v.URI).
					Int("status", v.Status).
					Msg("request")

				return nil
			},
		}))
	}

	h.mapRoutes()

	return h
}

func (h *Handler) mapRoutes() {
	log.Debug().Msg("mapping routes")

	// Basic health route
	h.Server.GET("/health", func(c echo.Context) error {
		return c.String(200, "OK")
	})

	// Setup Prometheus metrics route
	h.Server.GET("/metrics", func(c echo.Context) error {
		r, err := h.GatherMetrics()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		handler := promhttp.HandlerFor(r, promhttp.HandlerOpts{})
		handler.ServeHTTP(c.Response(), c.Request())
		return nil
	})

	// Static files if not using nginx
	envConfig := config.GetEnvConfig()
	h.Server.Static(envConfig.VideosDir, envConfig.VideosDir)

	// RiverUI
	h.Server.Any("/riverui/", echo.WrapHandler(h.RiverUIServer), AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	h.Server.Any("/riverui/*", echo.WrapHandler(h.RiverUIServer), AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))

	// Swagger
	h.Server.GET("/swagger/*", echoSwagger.WrapHandler)

	// Proxy frontend server
	frontendURL, _ := url.Parse("http://127.0.0.1:3000")
	h.Server.Any("/*", echo.WrapHandler(http.StripPrefix("/", httputil.NewSingleHostReverseProxy(frontendURL))))

	// create v1 group and setup v1 routes
	v1 := h.Server.Group("/api/v1")
	groupV1Routes(v1, h)
}

func groupV1Routes(e *echo.Group, h *Handler) {

	// Auth
	authGroup := e.Group("/auth")
	authGroup.POST("/register", h.Register)
	authGroup.POST("/login", h.Login)
	authGroup.POST("/logout", h.Logout, AuthGuardMiddleware)
	authGroup.GET("/me", h.Me, AuthGuardMiddleware, AuthGetUserMiddleware)
	authGroup.POST("/change-password", h.ChangePassword, AuthGuardMiddleware, AuthGetUserMiddleware)
	authGroup.GET("/oauth/login", h.OAuthLogin)
	authGroup.GET("/oauth/callback", h.OAuthCallback)

	// Channel
	channelGroup := e.Group("/channel")
	channelGroup.POST("", h.CreateChannel, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	channelGroup.GET("", h.GetChannels)
	channelGroup.GET("/:id", h.GetChannel)
	channelGroup.GET("/name/:name", h.GetChannelByName)
	channelGroup.PUT("/:id", h.UpdateChannel, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	channelGroup.DELETE("/:id", h.DeleteChannel, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	channelGroup.POST("/:id/update-image", h.UpdateChannelImage, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))

	// VOD
	vodGroup := e.Group("/vod")
	vodGroup.POST("", h.CreateVod, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	vodGroup.GET("", h.GetVods)
	vodGroup.GET("/:id", h.GetVod)
	vodGroup.GET("/external_id/:external_id", h.GetVod)
	vodGroup.GET("/search", h.SearchVods)
	vodGroup.PUT("/:id", h.UpdateVod, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	vodGroup.DELETE("/:id", h.DeleteVod, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	vodGroup.GET("/:id/playlist", h.GetVodPlaylists)
	vodGroup.GET("/:id/clips", h.GetVodClips)
	vodGroup.GET("/paginate", h.GetVodsPagination)
	vodGroup.GET("/:id/chat", h.GetVodChatComments)
	vodGroup.GET("/:id/chat/seek", h.GetNumberOfVodChatCommentsFromTime)
	vodGroup.GET("/:id/chat/userid", h.GetUserIdFromChat)
	vodGroup.GET("/:id/chat/emotes", h.GetChatEmotes)
	vodGroup.GET("/:id/chat/badges", h.GetChatBadges)
	vodGroup.GET("/:id/chat/histogram", h.GetVodChatHistogram)
	vodGroup.POST("/:id/lock", h.LockVod, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	vodGroup.POST("/:id/generate-static-thumbnail", h.GenerateStaticThumbnail, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	vodGroup.POST("/:id/generate-sprite-thumbnails", h.GenerateSpriteThumbnails, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	vodGroup.GET("/:id/thumbnails/vtt", h.GetVodSpriteThumbnails)

	// Queue
	queueGroup := e.Group("/queue")
	queueGroup.POST("", h.CreateQueueItem, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	queueGroup.GET("", h.GetQueueItems, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.ArchiverRole))
	queueGroup.GET("/:id", h.GetQueueItem, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.ArchiverRole))
	queueGroup.PUT("/:id", h.UpdateQueueItem, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	queueGroup.DELETE("/:id", h.DeleteQueueItem, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	queueGroup.GET("/:id/tail", h.ReadQueueLogFile, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.ArchiverRole))
	queueGroup.POST("/:id/stop", h.StopQueueItem, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	queueGroup.POST("/task/start", h.StartQueueTask, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.ArchiverRole))

	// Twitch
	twitchGroup := e.Group("/twitch")
	twitchGroup.GET("/channel", h.GetTwitchChannel)
	twitchGroup.GET("/video", h.GetTwitchVideo)
	// twitchGroup.GET("/gql/video", h.GQLGetTwitchVideo)
	// twitchGroup.GET("/categories", h.GetTwitchCategories)

	// Archive
	archiveGroup := e.Group("/archive")
	archiveGroup.POST("/channel", h.ArchiveChannel, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.ArchiverRole))
	archiveGroup.POST("/video", h.ArchiveVideo, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.ArchiverRole))
	archiveGroup.POST("/convert-twitch-live-chat", h.ConvertTwitchChat, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))

	// Admin
	adminGroup := e.Group("/admin")
	adminGroup.GET("/video-statistics", h.GetVideoStatistics, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	adminGroup.GET("/system-overview", h.GetSystemOverview, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	adminGroup.GET("/storage-distribution", h.GetStorageDistribution, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	adminGroup.GET("/info", h.GetInfo, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))

	// User
	userGroup := e.Group("/user")
	userGroup.GET("", h.GetUsers, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	userGroup.GET("/:id", h.GetUser, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	userGroup.PUT("/:id", h.UpdateUser, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	userGroup.DELETE("/:id", h.DeleteUser, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))

	// Config
	configGroup := e.Group("/config")
	configGroup.GET("", h.GetConfig, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	configGroup.PUT("", h.UpdateConfig, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))

	// Live
	liveGroup := e.Group("/live")
	liveGroup.GET("", h.GetLiveWatchedChannels, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	liveGroup.POST("", h.AddLiveWatchedChannel, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	liveGroup.PUT("/:id", h.UpdateLiveWatchedChannel, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	liveGroup.DELETE("/:id", h.DeleteLiveWatchedChannel, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	liveGroup.GET("/check", h.Check, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	// liveGroup.GET("/vod", h.CheckVodWatchedChannels, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	// liveGroup.POST("/archive", h.ArchiveLiveChannel, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.ArchiverRole))

	// Playback
	playbackGroup := e.Group("/playback")
	playbackGroup.GET("", h.GetAllProgress, AuthGuardMiddleware, AuthGetUserMiddleware)
	playbackGroup.GET("/:id", h.GetProgress, AuthGuardMiddleware, AuthGetUserMiddleware)
	playbackGroup.POST("/progress", h.UpdateProgress, AuthGuardMiddleware, AuthGetUserMiddleware)
	playbackGroup.POST("/status", h.UpdateStatus, AuthGuardMiddleware, AuthGetUserMiddleware)
	playbackGroup.DELETE("/:id", h.DeleteProgress, AuthGuardMiddleware, AuthGetUserMiddleware)
	playbackGroup.GET("/last", h.GetLastPlaybacks, AuthGuardMiddleware, AuthGetUserMiddleware)
	playbackGroup.POST("/start", h.StartPlayback)

	// Playlist
	playlistGroup := e.Group("/playlist")
	playlistGroup.GET("/:id", h.GetPlaylist)
	playlistGroup.GET("", h.GetPlaylists)
	playlistGroup.POST("", h.CreatePlaylist, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	playlistGroup.POST("/:id", h.AddVodToPlaylist, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	playlistGroup.DELETE("/:id/vod", h.DeleteVodFromPlaylist, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	playlistGroup.DELETE("/:id", h.DeletePlaylist, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	playlistGroup.PUT("/:id", h.UpdatePlaylist, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	playlistGroup.PUT("/:id/multistream/delay", h.SetVodDelayOnPlaylistMultistream, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))

	// Exec
	execGroup := e.Group("/exec")
	execGroup.POST("/ffprobe", h.GetFfprobeData, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.ArchiverRole))

	// Task
	taskGroup := e.Group("/task")
	taskGroup.POST("/start", h.StartTask, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))

	// Notification
	notificationGroup := e.Group("/notification")
	notificationGroup.POST("/test", h.TestNotification, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))

	// Chapter
	chapterGroup := e.Group("/chapter")
	chapterGroup.GET("/video/:videoId", h.GetVideoChapters)
	chapterGroup.GET("/video/:videoId/webvtt", h.GetWebVTTChapters)

	// Category
	categoryGroup := e.Group("/category")
	categoryGroup.GET("", h.GetCategories)

	// Blocked
	blockedGroup := e.Group("/blocked-video")
	blockedGroup.GET("", h.GetBlockedVideos)
	blockedGroup.POST("/:id", h.CreateBlockedVideo, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	blockedGroup.DELETE("/:id", h.DeleteBlockedVideo, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.EditorRole))
	blockedGroup.GET("/:id", h.IsVideoBlocked)
}

func (h *Handler) Serve(ctx context.Context) error {
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "4000"
	}
	// Run the server in a goroutine
	serverErrCh := make(chan error, 1)
	go func() {
		if err := h.Server.Start(fmt.Sprintf(":%s", appPort)); err != nil && err != http.ErrServerClosed {
			serverErrCh <- err
		}
		close(serverErrCh)
	}()

	// Listen for the context to be canceled or an error to occur in the server
	select {
	case <-ctx.Done():
		log.Info().Msg("Context canceled, shutting down the server")
	case err := <-serverErrCh:
		if err != nil {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}

	// Shutdown the server with a timeout of 10 seconds
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := h.Server.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("failed to shutdown server")
	}

	return nil
}
