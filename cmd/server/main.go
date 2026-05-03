package main

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	_ "github.com/zibbp/ganymede/internal/kv"
	"github.com/zibbp/ganymede/internal/server"
	"github.com/zibbp/ganymede/internal/utils"
)

//	@title			Ganymede API
//	@version		1.0
//	@description	Authentication is handled using JWT tokens. The tokens are set as access-token and refresh-token cookies.
//	@description	For information regarding which role is authorized for which endpoint, see the http handler https://github.com/Zibbp/ganymede/blob/main/internal/transport/http/handler.go.

//	@host		localhost:4000
//	@BasePath	/api/v1
//	@schemes	https

//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						Authorization
//	@description				Authorization: Bearer gym_<prefix>_<secret>. Mint a key via the admin UI.

//	@securityDefinitions.apikey	ApiKeyCookieAuth
//	@in							cookie
//	@name						access-token

func main() {
	ctx := context.Background()

	if os.Getenv("DEVELOPMENT") == "true" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	log.Info().Str("commit", utils.Commit).Str("tag", utils.Tag).Str("build_time", utils.BuildTime).Msg("starting server")

	if err := server.Run(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to run")
	}
}
