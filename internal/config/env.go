package config

import (
	"context"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-envconfig"
)

// EnvApplicationConfig represents the 'core' environment variables for the application.
type EnvApplicationConfig struct {
	DB_HOST          string `env:"DB_HOST, required"`
	DB_PORT          string `env:"DB_PORT, required"`
	DB_USER          string `env:"DB_USER, required"`
	DB_PASS          string `env:"DB_PASS, required"`
	DB_NAME          string `env:"DB_NAME, required"`
	DB_SSL           string `env:"DB_SSL, default=disable"`
	DB_SSL_ROOT_CERT string `env:"DB_SSL_ROOT_CERT, default="`
}

// EnvConfig represents the 'application' environment variables for the application.
type EnvConfig struct {
	// application
	Development bool `env:"DEVELOPMENT"`
	DEBUG       bool `env:"DEBUG, default=false"`
	// customizable paths
	VideosDir            string `env:"VIDEOS_DIR, default=/data/videos"`
	TempDir              string `env:"TEMP_DIR, default=/data/temp"`
	ConfigDir            string `env:"CONFIG_DIR, default=/data/config"`
	LogsDir              string `env:"LOGS_DIR, default=/data/logs"`
	PathMigrationEnabled bool   `env:"PATH_MIGRATION_ENABLED, default=true"`
	// platform variables
	TwitchClientId     string `env:"TWITCH_CLIENT_ID, required"`
	TwitchClientSecret string `env:"TWITCH_CLIENT_SECRET, required"`

	// worker config
	MaxChatDownloadExecutions         int `env:"MAX_CHAT_DOWNLOAD_EXECUTIONS, default=3"`
	MaxChatRenderExecutions           int `env:"MAX_CHAT_RENDER_EXECUTIONS, default=2"`
	MaxVideoDownloadExecutions        int `env:"MAX_VIDEO_DOWNLOAD_EXECUTIONS, default=2"`
	MaxVideoConvertExecutions         int `env:"MAX_VIDEO_CONVERT_EXECUTIONS, default=3"`
	MaxVideoSpriteThumbnailExecutions int `env:"MAX_VIDEO_SPRITE_THUMBNAIL_EXECUTIONS, default=2"`

	// oauth OIDC
	OAuthEnabled      bool   `env:"OAUTH_ENABLED, default=false"`
	OAuthProviderURL  string `env:"OAUTH_PROVIDER_URL, default="`
	OAuthClientID     string `env:"OAUTH_CLIENT_ID, default="`
	OAuthClientSecret string `env:"OAUTH_CLIENT_SECRET, default="`
	OAuthRedirectURL  string `env:"OAUTH_REDIRECT_URL, default="`

	// frontend
	CDN_URL string `env:"CDN_URL, default="` // Populate if using an external host for the static files (Nginx, S3, etc). By default Ganymede will serve the VIDEOS_DIR directory.
}

const fileSuffix = "_FILE"

// processFileSecrets iterates through environment variables, checks for the _FILE suffix.
// If found, it reads the content of the file specified by the variable's value
// and sets a new environment variable without the _FILE suffix.
func processFileSecrets() {
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		envKeyFile := parts[0]
		filePath := parts[1]

		if !strings.HasSuffix(envKeyFile, fileSuffix) {
			continue
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Error().
				Str("env_var", envKeyFile).
				Str("file_path", filePath).
				Err(err).
				Msg("failed to read secret file")
			continue
		}

		targetKey := strings.TrimSuffix(envKeyFile, fileSuffix)
		secretValue := strings.TrimSpace(string(content))

		if err := os.Setenv(targetKey, secretValue); err != nil {
			log.Error().
				Str("env_var", targetKey).
				Err(err).
				Msg("failed to set environment variable from secret file")
		} else {
			log.Debug().
				Str("env_var", targetKey).
				Str("source_file", filePath).
				Msg("successfully loaded secret from file")
		}
	}
}

// GetEnvConfig returns the environment variables for the application
func GetEnvConfig() EnvConfig {
	processFileSecrets()

	ctx := context.Background()

	var c EnvConfig
	if err := envconfig.Process(ctx, &c); err != nil {
		log.Panic().Err(err).Msg("error getting env config")
	}
	return c
}

func GetEnvApplicationConfig() EnvApplicationConfig {
	processFileSecrets()

	ctx := context.Background()

	var c EnvApplicationConfig
	if err := envconfig.Process(ctx, &c); err != nil {
		log.Panic().Err(err).Msg("error getting env config")
	}
	return c
}
