package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestProcessFileSecrets_ValidFile(t *testing.T) {
	tempDir := t.TempDir()
	secretFile := filepath.Join(tempDir, "secret.txt")
	secretContent := "my-secret-value"

	err := os.WriteFile(secretFile, []byte(secretContent), 0600)
	require.NoError(t, err)

	envKey := "TEST_SECRET_FILE"
	targetKey := "TEST_SECRET"

	err = os.Setenv(envKey, secretFile)
	require.NoError(t, err)
	defer os.Unsetenv(envKey)
	defer os.Unsetenv(targetKey)

	processFileSecrets()

	value := os.Getenv(targetKey)
	assert.Equal(t, secretContent, value)
}

func TestProcessFileSecrets_ValidFileWithWhitespace(t *testing.T) {
	tempDir := t.TempDir()
	secretFile := filepath.Join(tempDir, "secret.txt")
	secretContent := "  my-secret-with-whitespace  \n\t"
	expectedContent := "my-secret-with-whitespace"

	err := os.WriteFile(secretFile, []byte(secretContent), 0600)
	require.NoError(t, err)

	envKey := "DB_PASS_FILE"
	targetKey := "DB_PASS"

	err = os.Setenv(envKey, secretFile)
	require.NoError(t, err)
	defer os.Unsetenv(envKey)
	defer os.Unsetenv(targetKey)

	processFileSecrets()

	value := os.Getenv(targetKey)
	assert.Equal(t, expectedContent, value)
}

func TestProcessFileSecrets_FileNotFound(t *testing.T) {
	envKey := "MISSING_SECRET_FILE"
	targetKey := "MISSING_SECRET"
	nonExistentFile := "/path/to/nonexistent/file.txt"

	err := os.Setenv(envKey, nonExistentFile)
	require.NoError(t, err)
	defer os.Unsetenv(envKey)
	defer os.Unsetenv(targetKey)

	processFileSecrets()

	value := os.Getenv(targetKey)
	assert.Empty(t, value)
}

func TestProcessFileSecrets_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	secretFile := filepath.Join(tempDir, "empty.txt")

	err := os.WriteFile(secretFile, []byte(""), 0600)
	require.NoError(t, err)

	envKey := "EMPTY_SECRET_FILE"
	targetKey := "EMPTY_SECRET"

	err = os.Setenv(envKey, secretFile)
	require.NoError(t, err)
	defer os.Unsetenv(envKey)
	defer os.Unsetenv(targetKey)

	processFileSecrets()

	value := os.Getenv(targetKey)
	assert.Equal(t, "", value)
}

func TestProcessFileSecrets_MultipleSecrets(t *testing.T) {
	tempDir := t.TempDir()

	secrets := map[string]string{
		"DB_PASS_FILE":            "database-password",
		"TWITCH_CLIENT_SECRET_FILE": "twitch-secret",
		"JWT_SECRET_FILE":         "jwt-signing-key",
	}

	expectedTargets := map[string]string{
		"DB_PASS":            "database-password",
		"TWITCH_CLIENT_SECRET": "twitch-secret", 
		"JWT_SECRET":         "jwt-signing-key",
	}

	for envKey, content := range secrets {
		secretFile := filepath.Join(tempDir, envKey+".txt")
		err := os.WriteFile(secretFile, []byte(content), 0600)
		require.NoError(t, err)

		err = os.Setenv(envKey, secretFile)
		require.NoError(t, err)
		defer os.Unsetenv(envKey)
	}

	for targetKey := range expectedTargets {
		defer os.Unsetenv(targetKey)
	}

	processFileSecrets()

	for targetKey, expectedValue := range expectedTargets {
		value := os.Getenv(targetKey)
		assert.Equal(t, expectedValue, value, "Failed to load secret for %s", targetKey)
	}
}

func TestProcessFileSecrets_NoFileSuffix(t *testing.T) {
	regularVars := map[string]string{
		"REGULAR_VAR":    "regular-value",
		"ANOTHER_CONFIG": "another-value",
		"NO_SUFFIX":      "no-suffix-value",
	}

	for key, value := range regularVars {
		err := os.Setenv(key, value)
		require.NoError(t, err)
		defer os.Unsetenv(key)
	}

	processFileSecrets()

	for key, expectedValue := range regularVars {
		value := os.Getenv(key)
		assert.Equal(t, expectedValue, value)
	}
}

func TestProcessFileSecrets_MalformedEnvVar(t *testing.T) {
	envKey := "MALFORMED_FILE"
	targetKey := "MALFORMED"

	err := os.Setenv(envKey, "")
	require.NoError(t, err)
	defer os.Unsetenv(envKey)
	defer os.Unsetenv(targetKey)

	processFileSecrets()

	value := os.Getenv(targetKey)
	assert.Empty(t, value)
}

