package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnvConfig(t *testing.T) {
	os.Setenv("VIDEOS_DIR", "/custom/videos")

	env := GetEnvConfig()

	assert.Equal(t, "/custom/videos", env.VideosDir)

	os.Unsetenv("VIDEOS_DIR")
}

func TestGetEnvRequiredConfig(t *testing.T) {
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASS", "password")
	os.Setenv("DB_NAME", "ganymede")
	os.Setenv("JWT_SECRET", "secret")
	os.Setenv("JWT_REFRESH_SECRET", "refresh_secret")
	os.Setenv("FRONTEND_HOST", "localhost")

	env := GetEnvApplicationConfig()

	assert.Equal(t, "localhost", env.DB_HOST)
	assert.Equal(t, "5432", env.DB_PORT)
	assert.Equal(t, "postgres", env.DB_USER)
	assert.Equal(t, "password", env.DB_PASS)
	assert.Equal(t, "ganymede", env.DB_NAME)
	assert.Equal(t, "secret", env.JWTSecret)
	assert.Equal(t, "refresh_secret", env.JWTRefreshSecret)
	assert.Equal(t, "localhost", env.FrontendHost)

	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_PORT")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_PASS")
	os.Unsetenv("DB_NAME")
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("JWT_REFRESH_SECRET")
	os.Unsetenv("FRONTEND_HOST")
}

func TestGetEnvRequiredConfigMissing(t *testing.T) {
	assert.Panics(t, func() { GetEnvApplicationConfig() })
}
