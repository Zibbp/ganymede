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
	AuthService    AuthService
	ChannelService ChannelService
	VodService     VodService
	QueueService   QueueService
	TwitchService  TwitchService
	ArchiveService ArchiveService
}

type Handler struct {
	Server  *echo.Echo
	Service Services
}

func NewHandler(authService AuthService, channelService ChannelService, vodService VodService, queueService QueueService, twitchService TwitchService, archiveService ArchiveService) *Handler {
	log.Debug().Msg("creating new handler")

	h := &Handler{
		Server: echo.New(),
		Service: Services{
			AuthService:    authService,
			ChannelService: channelService,
			VodService:     vodService,
			QueueService:   queueService,
			TwitchService:  twitchService,
			ArchiveService: archiveService,
		},
	}

	// Middleware
	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	h.mapRoutes()

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

	// Channel
	channelGroup := e.Group("/channel")
	channelGroup.POST("", h.CreateChannel, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	channelGroup.GET("", h.GetChannels)
	channelGroup.GET("/:id", h.GetChannel)
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
	queueGroup.GET("", h.GetQueueItems)

	// Twitch
	twitchGroup := e.Group("/twitch")
	twitchGroup.GET("/channel", h.GetTwitchUser)
	twitchGroup.GET("/vod", h.GetTwitchVod)

	// Archive
	archiveGroup := e.Group("/archive")
	archiveGroup.POST("/channel", h.ArchiveTwitchChannel, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
	archiveGroup.POST("/vod", h.ArchiveTwitchVod, authMiddleware, auth.GetUserMiddleware, auth.UserRoleMiddleware(utils.EditorRole))
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
