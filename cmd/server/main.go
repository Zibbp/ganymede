package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	_ "github.com/zibbp/ganymede/internal/kv"
	"github.com/zibbp/ganymede/internal/server"
	"github.com/zibbp/ganymede/internal/utils"
)

//	@title			Ganymede API
//	@version		1.0
//	@description	Ganymede authenticates programmatic API clients with scoped API keys. Send the complete key in the Authorization header as a Bearer token when API-key authentication is enabled.
//	@description	API keys are created by an administrator in the web UI and the complete key is shown only once. Keys use resource:tier scopes such as vod:read, vod:write, and *:admin.
//	@description	For interactive browser use, sign in through the web UI or POST /auth/login. This establishes an HTTP-only session cookie that the browser sends automatically with same-origin requests, including requests from this Swagger UI.
//	@description	Browser sessions use the admin > editor > archiver > user role hierarchy. They are not external API credentials and are therefore not entered in the Swagger Authorize dialog.
//	@description	Operations marked ApiKeyAuth accept API keys. Browser-only operations, including API-key management and per-user playback state, require an interactive session and do not accept API keys.

//	@BasePath	/api/v1

//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						Authorization
//	@description				Programmatic API authentication. Enter `Bearer gym_{prefix}_{secret}`. Create keys in the admin UI; the complete key is shown only once.

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if os.Getenv("DEVELOPMENT") == "true" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	log.Info().Str("commit", utils.Commit).Str("tag", utils.Tag).Str("build_time", utils.BuildTime).Msg("starting server")

	if err := server.Run(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to run")
	}
}
