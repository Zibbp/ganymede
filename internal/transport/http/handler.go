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
	"github.com/zibbp/ganymede/internal/api_key"
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
	NotificationService NotificationService
	ApiKeyService       ApiKeyService
	PlatformTwitch      platform.Platform
}

type Handler struct {
	Server         *echo.Echo
	Service        Services
	SessionManager *scs.SessionManager
	RiverUIServer  *riverui.Handler
}

var sessionManager *scs.SessionManager

// apiKeyService is the package-level ApiKeyService used by the auth
// middleware. It is wired in NewHandler, mirroring the sessionManager
// pattern above so middleware functions stay parameter-free and chain
// cleanly with Echo's middleware signature.
var apiKeyService *api_key.Service

func NewHandler(database *database.Database, authService AuthService, channelService ChannelService, vodService VodService, queueService QueueService, archiveService ArchiveService, adminService AdminService, userService UserService, liveService LiveService, playbackService PlaybackService, metricsService MetricsService, playlistService PlaylistService, taskService TaskService, chapterService ChapterService, categoryService CategoryService, blockedVideoService BlockedVideoService, notificationService NotificationService, apiKeySvc *api_key.Service, platformTwitch platform.Platform, riverUIServer *riverui.Handler) *Handler {
	log.Debug().Msg("creating route handler")
	envConfig := config.GetEnvConfig()

	// Stash the ApiKeyService at package scope so the auth middleware
	// (which has the plain echo.MiddlewareFunc signature) can reach it
	// without each route having to wrap a closure.
	apiKeyService = apiKeySvc

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
			NotificationService: notificationService,
			ApiKeyService:       apiKeySvc,
			PlatformTwitch:      platformTwitch,
		},
		SessionManager: sessionManager,
		RiverUIServer:  riverUIServer,
	}

	// Enable gzip compression for API routes only.
	//
	// We use HasPrefix("/api/") rather than Contains("/api") because the
	// catch-all at the bottom of mapRoutes proxies every non-matching
	// path to the Next.js frontend, which serves its own gzipped
	// responses. A frontend path that happens to contain the substring
	// "api" (e.g. /admin/api-keys) was being double-gzipped: Next.js
	// gzipped once, Echo gzipped again, and the browser decoded only the
	// outer layer — leaving raw gzip bytes as the page body.
	h.Server.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Skipper: func(c echo.Context) bool {
			return !strings.HasPrefix(c.Request().URL.Path, "/api/")
		},
	}))

	// Use sessions
	h.Server.Use(session.LoadAndSave(sessionManager))

	// Middleware
	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	h.Server.HideBanner = true

	// If frontend is external then allow cors
	// AllowOriginFunc reflects the request origin so credentials work (browsers
	// reject Access-Control-Allow-Origin: * with credentials).
	h.Server.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOriginFunc: func(origin string) (bool, error) {
			return true, nil
		},
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
	env := config.GetEnvConfig()
	// Use one handler for both GET + HEAD
	videosH := echo.WrapHandler(http.StripPrefix(env.VideosDir, http.FileServer(http.Dir(env.VideosDir))))
	tempH := echo.WrapHandler(http.StripPrefix(env.TempDir, http.FileServer(http.Dir(env.TempDir))))

	h.Server.GET(env.VideosDir+"/*", videosH)
	h.Server.HEAD(env.VideosDir+"/*", videosH)

	h.Server.GET(env.TempDir+"/*", tempH)
	h.Server.HEAD(env.TempDir+"/*", tempH)

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
	//
	// Write/admin endpoints accept either a session cookie or an API
	// key. GETs stay public.
	channelGroup := e.Group("/channel")
	channelGroup.POST("", h.CreateChannel, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.EditorRole, utils.ApiKeyScopeChannelWrite))
	channelGroup.GET("", h.GetChannels)
	channelGroup.GET("/:id", h.GetChannel)
	channelGroup.GET("/name/:name", h.GetChannelByName)
	channelGroup.PUT("/:id", h.UpdateChannel, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.EditorRole, utils.ApiKeyScopeChannelWrite))
	channelGroup.DELETE("/:id", h.DeleteChannel, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.AdminRole, utils.ApiKeyScopeChannelAdmin))
	channelGroup.POST("/:id/update-image", h.UpdateChannelImage, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.EditorRole, utils.ApiKeyScopeChannelWrite))

	// VOD
	//
	// Write/admin endpoints (POST/PUT/DELETE for the VOD itself) accept
	// either a session cookie or an API key — see issue #1070, where
	// external scripts need to delete VODs after archiving them. Read
	// endpoints stay unauthenticated as before.
	vodGroup := e.Group("/vod")
	vodGroup.POST("", h.CreateVod, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.EditorRole, utils.ApiKeyScopeVodWrite))
	vodGroup.GET("", h.GetVods)
	vodGroup.GET("/:id", h.GetVod)
	vodGroup.GET("/external_id/:external_id", h.GetVod)
	vodGroup.GET("/search", h.SearchVods)
	vodGroup.PUT("/:id", h.UpdateVod, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.EditorRole, utils.ApiKeyScopeVodWrite))
	vodGroup.DELETE("/:id", h.DeleteVod, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.AdminRole, utils.ApiKeyScopeVodAdmin))
	vodGroup.GET("/:id/playlist", h.GetVodPlaylists)
	vodGroup.GET("/:id/clips", h.GetVodClips)
	vodGroup.GET("/paginate", h.GetVodsPagination)
	vodGroup.GET("/:id/chat", h.GetVodChatComments)
	vodGroup.GET("/:id/chat/seek", h.GetNumberOfVodChatCommentsFromTime)
	vodGroup.GET("/:id/chat/userid", h.GetUserIdFromChat)
	vodGroup.GET("/:id/chat/emotes", h.GetChatEmotes)
	vodGroup.GET("/:id/chat/badges", h.GetChatBadges)
	vodGroup.GET("/:id/chat/histogram", h.GetVodChatHistogram)
	vodGroup.POST("/:id/lock", h.LockVod, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.EditorRole, utils.ApiKeyScopeVodWrite))
	vodGroup.POST("/:id/generate-static-thumbnail", h.GenerateStaticThumbnail, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.EditorRole, utils.ApiKeyScopeVodWrite))
	vodGroup.POST("/:id/generate-sprite-thumbnails", h.GenerateSpriteThumbnails, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.EditorRole, utils.ApiKeyScopeVodWrite))
	vodGroup.GET("/:id/thumbnails/vtt", h.GetVodSpriteThumbnails)
	vodGroup.POST("/:id/ffprobe", h.GetFFprobe, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.ArchiverRole, utils.ApiKeyScopeVodWrite))

	// Queue
	//
	// Issue #1070 calls out "running actions" — i.e. starting tasks from
	// scripts. The queue is the surface where archive/transcode jobs
	// live, so we accept API keys on every queue endpoint. Read endpoints
	// require read scope (matches ArchiverRole), writes require write
	// scope (matches EditorRole), and POST/DELETE that previously
	// required AdminRole now require admin scope.
	queueGroup := e.Group("/queue")
	queueGroup.POST("", h.CreateQueueItem, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.AdminRole, utils.ApiKeyScopeQueueAdmin))
	queueGroup.GET("", h.GetQueueItems, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.ArchiverRole, utils.ApiKeyScopeQueueRead))
	queueGroup.GET("/:id", h.GetQueueItem, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.ArchiverRole, utils.ApiKeyScopeQueueRead))
	queueGroup.PUT("/:id", h.UpdateQueueItem, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.EditorRole, utils.ApiKeyScopeQueueWrite))
	queueGroup.DELETE("/:id", h.DeleteQueueItem, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.AdminRole, utils.ApiKeyScopeQueueAdmin))
	queueGroup.GET("/:id/tail", h.ReadQueueLogFile, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.ArchiverRole, utils.ApiKeyScopeQueueRead))
	queueGroup.POST("/:id/stop", h.StopQueueItem, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.AdminRole, utils.ApiKeyScopeQueueAdmin))
	queueGroup.POST("/task/start", h.StartQueueTask, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.ArchiverRole, utils.ApiKeyScopeQueueWrite))

	// Twitch
	twitchGroup := e.Group("/twitch")
	twitchGroup.GET("/channel", h.GetTwitchChannel)
	twitchGroup.GET("/video", h.GetTwitchVideo)
	// twitchGroup.GET("/gql/video", h.GQLGetTwitchVideo)
	// twitchGroup.GET("/categories", h.GetTwitchCategories)

	// Archive
	//
	// All POSTs accept either a session cookie or an API key. Archive
	// channel/video are write-tier (Archiver role); the chat converter
	// is admin-tier (Admin role).
	archiveGroup := e.Group("/archive")
	archiveGroup.POST("/channel", h.ArchiveChannel, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.ArchiverRole, utils.ApiKeyScopeArchiveWrite))
	archiveGroup.POST("/video", h.ArchiveVideo, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.ArchiverRole, utils.ApiKeyScopeArchiveWrite))
	archiveGroup.POST("/convert-twitch-live-chat", h.ConvertTwitchChat, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.AdminRole, utils.ApiKeyScopeArchiveAdmin))

	// Admin: system stats and info.
	//
	// Read-only system endpoints accept either a session cookie or an
	// API key with system:read. The /admin/api-keys management endpoints
	// further down stay session-only — minting keys requires the admin
	// web UI to prevent key-mints-key escalation.
	adminGroup := e.Group("/admin")
	adminGroup.GET("/video-statistics", h.GetVideoStatistics, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.AdminRole, utils.ApiKeyScopeSystemRead))
	adminGroup.GET("/system-overview", h.GetSystemOverview, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.AdminRole, utils.ApiKeyScopeSystemRead))
	adminGroup.GET("/storage-distribution", h.GetStorageDistribution, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.AdminRole, utils.ApiKeyScopeSystemRead))
	adminGroup.GET("/info", h.GetInfo, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.AdminRole, utils.ApiKeyScopeSystemRead))

	// Admin: API keys. Session-only — admins must use the web UI to mint
	// or revoke keys. This avoids the chicken-and-egg of needing a key
	// to manage keys, and means a stolen key cannot mint or escalate
	// other keys.
	adminGroup.GET("/api-keys", h.ListApiKeys, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	adminGroup.POST("/api-keys", h.CreateApiKey, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	adminGroup.DELETE("/api-keys/:id", h.DeleteApiKey, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))

	// User
	//
	// All endpoints require AdminRole for sessions. API keys are gated
	// at user:read for GETs, user:write for PUT, user:admin for DELETE
	// — same tier-by-method pattern used elsewhere.
	userGroup := e.Group("/user")
	userGroup.GET("", h.GetUsers, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.AdminRole, utils.ApiKeyScopeUserRead))
	userGroup.GET("/:id", h.GetUser, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.AdminRole, utils.ApiKeyScopeUserRead))
	userGroup.PUT("/:id", h.UpdateUser, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.AdminRole, utils.ApiKeyScopeUserWrite))
	userGroup.DELETE("/:id", h.DeleteUser, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.AdminRole, utils.ApiKeyScopeUserAdmin))

	// Config
	configGroup := e.Group("/config")
	configGroup.GET("", h.GetConfig, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.AdminRole, utils.ApiKeyScopeConfigRead))
	configGroup.PUT("", h.UpdateConfig, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.AdminRole, utils.ApiKeyScopeConfigWrite))

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
	//
	// All write endpoints accept either a session cookie or an API key.
	// Issue #1070's second use case: scripts that auto-create or
	// reorder playlists. GET endpoints stay unauthenticated.
	playlistGroup := e.Group("/playlist")
	playlistGroup.GET("/:id", h.GetPlaylist)
	playlistGroup.GET("", h.GetPlaylists)
	playlistGroup.POST("", h.CreatePlaylist, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.EditorRole, utils.ApiKeyScopePlaylistWrite))
	playlistGroup.POST("/:id", h.AddVodToPlaylist, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.EditorRole, utils.ApiKeyScopePlaylistWrite))
	playlistGroup.DELETE("/:id/vod", h.DeleteVodFromPlaylist, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.EditorRole, utils.ApiKeyScopePlaylistWrite))
	playlistGroup.DELETE("/:id", h.DeletePlaylist, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.EditorRole, utils.ApiKeyScopePlaylistWrite))
	playlistGroup.PUT("/:id", h.UpdatePlaylist, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.EditorRole, utils.ApiKeyScopePlaylistWrite))
	playlistGroup.PUT("/:id/multistream/delay", h.SetVodDelayOnPlaylistMultistream, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.EditorRole, utils.ApiKeyScopePlaylistWrite))
	playlistGroup.PUT("/:id/rules", h.SetPlaylistRules, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.EditorRole, utils.ApiKeyScopePlaylistWrite))
	playlistGroup.GET("/:id/rules", h.GetPlaylistRules, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.EditorRole, utils.ApiKeyScopePlaylistRead))
	playlistGroup.POST("/:id/rules/test", h.TestPlaylistRules, AuthAPIKeyOrSessionMiddleware, AuthGetUserMiddleware, RequireRoleOrScope(utils.EditorRole, utils.ApiKeyScopePlaylistWrite))

	// Task
	taskGroup := e.Group("/task")
	taskGroup.POST("/start", h.StartTask, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))

	// Notification
	notificationGroup := e.Group("/notification")
	notificationGroup.GET("", h.GetNotifications, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	notificationGroup.GET("/:id", h.GetNotification, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	notificationGroup.POST("", h.CreateNotification, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	notificationGroup.PUT("/:id", h.UpdateNotification, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	notificationGroup.DELETE("/:id", h.DeleteNotification, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))
	notificationGroup.POST("/:id/test", h.TestNotification, AuthGuardMiddleware, AuthGetUserMiddleware, AuthUserRoleMiddleware(utils.AdminRole))

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
