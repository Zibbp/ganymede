package config

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-envconfig"
)

type EnvConfig struct {
	DB_HOST            string `env:"DB_HOST, required"`
	DB_PORT            string `env:"DB_PORT, required"`
	DB_USER            string `env:"DB_USER, required"`
	DB_PASS            string `env:"DB_PASS, required"`
	DB_NAME            string `env:"DB_NAME, required"`
	DB_SSL             string `env:"DB_SSL, default=disable"`
	DB_SSL_ROOT_CERT   string `env:"DB_SSL_ROOT_CERT, default="`
	VideosDir          string `env:"VIDEOS_DIR, default=/vods"`
	TempDir            string `env:"TEMP_DIR, default=/tmp"`
	TwitchClientId     string `env:"TWITCH_CLIENT_ID, default="`
	TwitchClientSecret string `env:"TWITCH_CLIENT_SECRET, default="`
}

func GetEnvConfig() EnvConfig {
	ctx := context.Background()

	var c EnvConfig
	if err := envconfig.Process(ctx, &c); err != nil {
		log.Panic().Err(err).Msg("error getting env config")
	}
	return c
}
