package http

import (
	"context"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/auth"
	"github.com/zibbp/ganymede/internal/utils"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type Services struct {
	AuthService      AuthService
	ChannelService   ChannelService
	VodService       VodService
	QueueService     QueueService
	TwitchService    TwitchService
	ArchiveService   ArchiveService
	AdminService     AdminService
	UserService      UserService
	ConfigService    ConfigService
	LiveService      LiveService
	SchedulerService SchedulerService
}

type Handler struct {
	Server  *echo.Echo
	Service Services
}

func NewHandler(authService AuthService, channelService ChannelService, vodService VodService, queueService QueueService, twitchService TwitchService, archiveService ArchiveService, adminService AdminService, userService UserService, configService ConfigService, liveService LiveService, schedulerService SchedulerService) *Handler {
	log.Debug().Msg("creating new handler")

	h := &Handler{
		Server: echo.New(),
		Service: Services{
			AuthService:      authService,
			ChannelService:   channelService,
			VodService:       vodService,
			QueueService:     queueService,
			TwitchService:    twitchService,
			ArchiveService:   archiveService,
			AdminService:     adminService,
			UserService:      userService,
			ConfigService:    configService,
			LiveService:      liveService,
			SchedulerService: schedulerService,
		},
	}

	// Middleware
	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	h.Server.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{os.Getenv("FRONTEND_HOST")},
		AllowMethods:     []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
		AllowCredentials: true,
	}))

	h.mapRoutes()

	// Start scheduler
	h.Service.SchedulerService.StartAppScheduler()
	// Start live scheduler as a goroutine
	// to avoid blocking application start
	go h.Service.SchedulerService.StartLiveScheduler()

	return h
}

func (h *Handler) mapRoutes() {
	log.Debug().Msg("mapping routes")

	h.Server.GET("/", func(c echo.Context) error {
		return c.String(200, "Hello, World!")
	})

	v1 := h.Server.Group("/api/v1")
	groupV1Routes(v1, h)
}

func groupV1Routes(e *echo.Group, h *Handler) {

	authMiddleware := middleware.JWTWithConfig(middleware.JWTConfig{
		Claims:                  &auth.Claims{},
		SigningKey:              []byte(auth.GetJWTSecret()),
		TokenLookup:             "cookie:access-token",
		ErrorHandlerWithContext: auth.JWTErrorChecker,
	})

	// Demo route for testing JWT and roles
	e.GET("/demo", func(c echo.Context) error {
		return c.JSON(http.StatusOK, "Demo Route")
	}, authMiddleware)

	// Auth
	authGroup := e.Group("/auth")
	authGroup.POST("/register", h.Register)
	authGroup.POST("/login", h.Login)
	authGroup.POST("/refresh", h.Refresh)
	authGroup.GET("/me", h.Me, authMiddleware)
	authGroup.POST("/change-password", h.ChangePassword, authMiddleware, auth.GetUserMiddleware)

	// Channel
	channelGroup := e.Group("/channel")
	channelGroup.POST("", h.CreateChannel, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	channelGroup.GET("", h.GetChannels)
	channelGroup.GET("/:id", h.GetChannel)
	channelGroup.GET("/name/:name", h.GetChannelByName)
	channelGroup.PUT("/:id", h.UpdateChannel, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	channelGroup.DELETE("/:id", h.DeleteChannel, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))

	// VOD
	vodGroup := e.Group("/vod")
	vodGroup.POST("", h.CreateVod, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	vodGroup.GET("", h.GetVods)
	vodGroup.GET("/:id", h.GetVod)
	vodGroup.PUT("/:id", h.UpdateVod, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	vodGroup.DELETE("/:id", h.DeleteVod, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))

	// Queue
	queueGroup := e.Group("/queue")
	queueGroup.POST("", h.CreateQueueItem, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))
	queueGroup.GET("", h.GetQueueItems, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.ArchiverRole))
	queueGroup.GET("/:id", h.GetQueueItem, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.ArchiverRole))
	queueGroup.PUT("/:id", h.UpdateQueueItem, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	queueGroup.DELETE("/:id", h.DeleteQueueItem, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))
	queueGroup.GET("/:id/log", h.ReadQueueLogFile, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.ArchiverRole))

	// Twitch
	twitchGroup := e.Group("/twitch")
	twitchGroup.GET("/channel", h.GetTwitchUser)
	twitchGroup.GET("/vod", h.GetTwitchVod)

	// Archive
	archiveGroup := e.Group("/archive")
	archiveGroup.POST("/channel", h.ArchiveTwitchChannel, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.ArchiverRole))
	archiveGroup.POST("/vod", h.ArchiveTwitchVod, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.ArchiverRole))
	archiveGroup.POST("/restart", h.RestartTask, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.ArchiverRole))

	// Admin
	adminGroup := e.Group("/admin")
	adminGroup.GET("/stats", h.GetStats, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))

	// User
	userGroup := e.Group("/user")
	userGroup.GET("", h.GetUsers, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))
	userGroup.GET("/:id", h.GetUser, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))
	userGroup.PUT("/:id", h.UpdateUser, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))
	userGroup.DELETE("/:id", h.DeleteUser, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))

	// Config
	configGroup := e.Group("/config")
	configGroup.GET("", h.GetConfig, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))
	configGroup.PUT("", h.UpdateConfig, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.AdminRole))

	// Live
	liveGroup := e.Group("/live")
	liveGroup.GET("", h.GetLiveWatchedChannels, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	liveGroup.POST("", h.AddLiveWatchedChannel, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	liveGroup.PUT("/:id", h.UpdateLiveWatchedChannel, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	liveGroup.DELETE("/:id", h.DeleteLiveWatchedChannel, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	liveGroup.GET("/check", h.Check, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
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
