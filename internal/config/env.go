package config

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-envconfig"
)

type EnvApplicationConfig struct {
	DB_HOST          string `env:"DB_HOST, required"`
	DB_PORT          string `env:"DB_PORT, required"`
	DB_USER          string `env:"DB_USER, required"`
	DB_PASS          string `env:"DB_PASS, required"`
	DB_NAME          string `env:"DB_NAME, required"`
	DB_SSL           string `env:"DB_SSL, default=disable"`
	DB_SSL_ROOT_CERT string `env:"DB_SSL_ROOT_CERT, default="`
	JWTSecret        string `env:"JWT_SECRET, required"`
	JWTRefreshSecret string `env:"JWT_REFRESH_SECRET, required"`
	FrontendHost     string `env:"FRONTEND_HOST, default="`
}

// EnvConfig represents the environment variables for the application
type EnvConfig struct {
	// application
	Development  bool   `env:"DEVELOPMENT"`
	DEBUG        bool   `env:"DEBUG, default=false"`
	CookieDomain string `env:"COOKIE_DOMAIN, default="`
	// customizable paths
	VideosDir string `env:"VIDEOS_DIR, default=/data/videos"`
	TempDir   string `env:"TEMP_DIR, default=/data/temp"`
	ConfigDir string `env:"CONFIG_DIR, default=/data/config"`
	LogsDir   string `env:"LOGS_DIR, default=/data/logs"`
	// platform variables
	TwitchClientId     string `env:"TWITCH_CLIENT_ID, default="`
	TwitchClientSecret string `env:"TWITCH_CLIENT_SECRET, default="`

	// worker config
	MaxChatDownloadExecutions  int `env:"MAX_CHAT_DOWNLOAD_EXECUTIONS, default=3"`
	MaxChatRenderExecutions    int `env:"MAX_CHAT_RENDER_EXECUTIONS, default=2"`
	MaxVideoDownloadExecutions int `env:"MAX_VIDEO_DOWNLOAD_EXECUTIONS, default=2"`
	MaxVideoConvertExecutions  int `env:"MAX_VIDEO_CONVERT_EXECUTIONS, default=3"`

	// oauth OIDC
	OAuthEnabled      bool   `env:"OAUTH_ENABLED, default=false"`
	OAuthProviderURL  string `env:"OAUTH_PROVIDER_URL, default="`
	OAuthClientID     string `env:"OAUTH_CLIENT_ID, default="`
	OAuthClientSecret string `env:"OAUTH_CLIENT_SECRET, default="`
	OAuthRedirectURL  string `env:"OAUTH_REDIRECT_URL, default="`
}

// GetEnvConfig returns the environment variables for the application
func GetEnvConfig() EnvConfig {
	ctx := context.Background()

	var c EnvConfig
	if err := envconfig.Process(ctx, &c); err != nil {
		log.Panic().Err(err).Msg("error getting env config")
	}
	return c
}

func GetEnvApplicationConfig() EnvApplicationConfig {
	ctx := context.Background()

	var c EnvApplicationConfig
	if err := envconfig.Process(ctx, &c); err != nil {
		log.Panic().Err(err).Msg("error getting env config")
	}
	return c
}
