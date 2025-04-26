package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnvConfig(t *testing.T) {
	assert.NoError(t, os.Setenv("VIDEOS_DIR", "/custom/videos"))
	assert.NoError(t, os.Setenv("TWITCH_CLIENT_ID", "client_id"))
	assert.NoError(t, os.Setenv("TWITCH_CLIENT_SECRET", "client_secret"))

	env := GetEnvConfig()

	assert.Equal(t, "/custom/videos", env.VideosDir)

	assert.NoError(t, os.Unsetenv("VIDEOS_DIR"))
}

func TestGetEnvRequiredConfig(t *testing.T) {
	assert.NoError(t, os.Setenv("DB_HOST", "localhost"))
	assert.NoError(t, os.Setenv("DB_PORT", "5432"))
	assert.NoError(t, os.Setenv("DB_USER", "postgres"))
	assert.NoError(t, os.Setenv("DB_PASS", "password"))
	assert.NoError(t, os.Setenv("DB_NAME", "ganymede"))
	assert.NoError(t, os.Setenv("JWT_SECRET", "secret"))
	assert.NoError(t, os.Setenv("JWT_REFRESH_SECRET", "refresh_secret"))
	assert.NoError(t, os.Setenv("FRONTEND_HOST", "localhost"))

	env := GetEnvApplicationConfig()

	assert.Equal(t, "localhost", env.DB_HOST)
	assert.Equal(t, "5432", env.DB_PORT)
	assert.Equal(t, "postgres", env.DB_USER)
	assert.Equal(t, "password", env.DB_PASS)
	assert.Equal(t, "ganymede", env.DB_NAME)

	assert.NoError(t, os.Unsetenv("DB_HOST"))
	assert.NoError(t, os.Unsetenv("DB_PORT"))
	assert.NoError(t, os.Unsetenv("DB_USER"))
	assert.NoError(t, os.Unsetenv("DB_PASS"))
	assert.NoError(t, os.Unsetenv("DB_NAME"))
	assert.NoError(t, os.Unsetenv("FRONTEND_HOST"))
}

func TestGetEnvRequiredConfigMissing(t *testing.T) {
	assert.Panics(t, func() { GetEnvApplicationConfig() })
}
