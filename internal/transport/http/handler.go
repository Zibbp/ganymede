package http

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	echoSwagger "github.com/swaggo/echo-swagger"
	_ "github.com/zibbp/ganymede/docs"
	"github.com/zibbp/ganymede/internal/auth"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/utils"
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
	SchedulerService    SchedulerService
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
	Server  *echo.Echo
	Service Services
}

func NewHandler(authService AuthService, channelService ChannelService, vodService VodService, queueService QueueService, archiveService ArchiveService, adminService AdminService, userService UserService, liveService LiveService, schedulerService SchedulerService, playbackService PlaybackService, metricsService MetricsService, playlistService PlaylistService, taskService TaskService, chapterService ChapterService, categoryService CategoryService, blockedVideoService BlockedVideoService, platformTwitch platform.Platform) *Handler {
	log.Debug().Msg("creating new handler")
	env := config.GetEnvApplicationConfig()

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
			SchedulerService:    schedulerService,
			PlaybackService:     playbackService,
			MetricsService:      metricsService,
			PlaylistService:     playlistService,
			TaskService:         taskService,
			ChapterService:      chapterService,
			CategoryService:     categoryService,
			BlockedVideoService: blockedVideoService,
			PlatformTwitch:      platformTwitch,
		},
	}

	// Middleware
	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	h.Server.HideBanner = true

	h.Server.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{env.FrontendHost},
		AllowMethods:     []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
		AllowCredentials: true,
	}))

	h.mapRoutes()

	// Start scheduler
	go h.Service.SchedulerService.StartLiveScheduler()

	return h
}

func (h *Handler) mapRoutes() {
	log.Debug().Msg("mapping routes")

	h.Server.GET("/", func(c echo.Context) error {
		return c.String(200, "Ganymede API")
	})

	h.Server.GET("/health", func(c echo.Context) error {
		return c.String(200, "OK")
	})

	h.Server.GET("/metrics", func(c echo.Context) error {
		r, err := h.GatherMetrics()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		handler := promhttp.HandlerFor(r, promhttp.HandlerOpts{})
		handler.ServeHTTP(c.Response(), c.Request())
		return nil
	})

	// Static files
	envConfig := config.GetEnvConfig()
	h.Server.Static(envConfig.VideosDir, envConfig.VideosDir)

	// Swagger
	h.Server.GET("/swagger/*", echoSwagger.WrapHandler)

	v1 := h.Server.Group("/api/v1")
	groupV1Routes(v1, h)
}

func groupV1Routes(e *echo.Group, h *Handler) {

	//auth.GuardMiddleware := middleware.JWTWithConfig(middleware.JWTConfig{
	//	Claims:                  &auth.Claims{},
	//	SigningKey:              []byte(auth.GetJWTSecret()),
	//	TokenLookup:             "cookie:access-token",
	//	ErrorHandlerWithContext: auth.JWTErrorChecker,
	//})

	// Demo route for testing JWT and roles
	e.GET("/demo", func(c echo.Context) error {
		return c.JSON(http.StatusOK, "Demo Route")
	}, auth.GuardMiddleware)

	// Auth
	authGroup := e.Group("/auth")
	authGroup.POST("/register", h.Register)
	authGroup.POST("/login", h.Login)
	authGroup.POST("/refresh", h.Refresh)
	authGroup.GET("/me", h.Me, auth.GuardMiddleware, auth.GetUserMiddleware)
	authGroup.POST("/change-password", h.ChangePassword, auth.GuardMiddleware, auth.GetUserMiddleware)
	authGroup.GET("/oauth/login", h.OAuthLogin)
	authGroup.GET("/oauth/callback", h.OAuthCallback)
	authGroup.GET("/oauth/refresh", h.OAuthTokenRefresh)
	authGroup.GET("/oauth/logout", h.OAuthLogout)

	// Channel
	channelGroup := e.Group("/channel")
	channelGroup.POST("", h.CreateChannel, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	channelGroup.GET("", h.GetChannels)
	channelGroup.GET("/:id", h.GetChannel)
	channelGroup.GET("/name/:name", h.GetChannelByName)
	channelGroup.PUT("/:id", h.UpdateChannel, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	channelGroup.DELETE("/:id", h.DeleteChannel, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))
	channelGroup.POST("/update-image", h.UpdateChannelImage, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))

	// VOD
	vodGroup := e.Group("/vod")
	vodGroup.POST("", h.CreateVod, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	vodGroup.GET("", h.GetVods)
	vodGroup.GET("/:id", h.GetVod)
	vodGroup.GET("/search", h.SearchVods)
	vodGroup.PUT("/:id", h.UpdateVod, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	vodGroup.DELETE("/:id", h.DeleteVod, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))
	vodGroup.GET("/:id/playlist", h.GetVodPlaylists)
	vodGroup.GET("/paginate", h.GetVodsPagination)
	vodGroup.GET("/:id/chat", h.GetVodChatComments)
	vodGroup.GET("/:id/chat/seek", h.GetNumberOfVodChatCommentsFromTime)
	vodGroup.GET("/:id/chat/userid", h.GetUserIdFromChat)
	vodGroup.GET("/:id/chat/emotes", h.GetChatEmotes)
	vodGroup.GET("/:id/chat/badges", h.GetChatBadges)
	vodGroup.POST("/:id/lock", h.LockVod, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))

	// Queue
	queueGroup := e.Group("/queue")
	queueGroup.POST("", h.CreateQueueItem, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))
	queueGroup.GET("", h.GetQueueItems, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.ArchiverRole))
	queueGroup.GET("/:id", h.GetQueueItem, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.ArchiverRole))
	queueGroup.PUT("/:id", h.UpdateQueueItem, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	queueGroup.DELETE("/:id", h.DeleteQueueItem, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))
	queueGroup.GET("/:id/tail", h.ReadQueueLogFile, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.ArchiverRole))
	queueGroup.POST("/:id/stop", h.StopQueueItem, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))
	queueGroup.POST("/task/start", h.StartQueueTask, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.ArchiverRole))

	// Twitch
	twitchGroup := e.Group("/twitch")
	twitchGroup.GET("/channel", h.GetTwitchChannel)
	twitchGroup.GET("/video", h.GetTwitchVideo)
	// twitchGroup.GET("/gql/video", h.GQLGetTwitchVideo)
	// twitchGroup.GET("/categories", h.GetTwitchCategories)

	// Archive
	archiveGroup := e.Group("/archive")
	archiveGroup.POST("/channel", h.ArchiveChannel, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.ArchiverRole))
	archiveGroup.POST("/video", h.ArchiveVideo, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.ArchiverRole))
	archiveGroup.POST("/convert-twitch-live-chat", h.ConvertTwitchChat, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))

	// Admin
	adminGroup := e.Group("/admin")
	adminGroup.GET("/stats", h.GetStats, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))
	adminGroup.GET("/info", h.GetInfo, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))

	// User
	userGroup := e.Group("/user")
	userGroup.GET("", h.GetUsers, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))
	userGroup.GET("/:id", h.GetUser, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))
	userGroup.PUT("/:id", h.UpdateUser, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))
	userGroup.DELETE("/:id", h.DeleteUser, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))

	// Config
	configGroup := e.Group("/config")
	configGroup.GET("", h.GetConfig, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))
	configGroup.PUT("", h.UpdateConfig, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))

	// Live
	liveGroup := e.Group("/live")
	liveGroup.GET("", h.GetLiveWatchedChannels, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	liveGroup.POST("", h.AddLiveWatchedChannel, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	liveGroup.POST("/multiple", h.AddMultipleLiveWatchedChannel, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	liveGroup.PUT("/:id", h.UpdateLiveWatchedChannel, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	liveGroup.DELETE("/:id", h.DeleteLiveWatchedChannel, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	liveGroup.GET("/check", h.Check, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	// liveGroup.GET("/vod", h.CheckVodWatchedChannels, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	// liveGroup.POST("/archive", h.ArchiveLiveChannel, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.ArchiverRole))

	// Playback
	playbackGroup := e.Group("/playback")
	playbackGroup.GET("", h.GetAllProgress, auth.GuardMiddleware, auth.GetUserMiddleware)
	playbackGroup.GET("/:id", h.GetProgress, auth.GuardMiddleware, auth.GetUserMiddleware)
	playbackGroup.POST("/progress", h.UpdateProgress, auth.GuardMiddleware, auth.GetUserMiddleware)
	playbackGroup.POST("/status", h.UpdateStatus, auth.GuardMiddleware, auth.GetUserMiddleware)
	playbackGroup.DELETE("/:id", h.DeleteProgress, auth.GuardMiddleware, auth.GetUserMiddleware)
	playbackGroup.GET("/last", h.GetLastPlaybacks, auth.GuardMiddleware, auth.GetUserMiddleware)
	playbackGroup.POST("/start", h.StartPlayback)

	// Playlist
	playlistGroup := e.Group("/playlist")
	playlistGroup.GET("/:id", h.GetPlaylist)
	playlistGroup.GET("", h.GetPlaylists)
	playlistGroup.POST("", h.CreatePlaylist, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	playlistGroup.POST("/:id", h.AddVodToPlaylist, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	playlistGroup.DELETE("/:id/vod", h.DeleteVodFromPlaylist, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	playlistGroup.DELETE("/:id", h.DeletePlaylist, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	playlistGroup.PUT("/:id", h.UpdatePlaylist, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))

	// Exec
	execGroup := e.Group("/exec")
	execGroup.POST("/ffprobe", h.GetFfprobeData, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.ArchiverRole))

	// Task
	taskGroup := e.Group("/task")
	taskGroup.POST("/start", h.StartTask, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))

	// Notification
	notificationGroup := e.Group("/notification")
	notificationGroup.POST("/test", h.TestNotification, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))

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
	blockedGroup.POST("/:id", h.CreateBlockedVideo, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	blockedGroup.DELETE("/:id", h.DeleteBlockedVideo, auth.GuardMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	blockedGroup.GET("/:id", h.IsVideoBlocked)
}

func (h *Handler) Serve() error {
	go func() {
		if err := h.Server.Start(":4000"); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()
	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := h.Server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to shutdown server")
	}

	return nil
}
