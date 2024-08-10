package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/zibbp/ganymede/internal/server"
)

// Setup initializes the integration test environment.
// It setups up the entire application and returns the various services for testing.
// A Postgres Testcontainer is used to provide a real database for further tersting.
func Setup(t *testing.T) (*server.Application, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create temporary postgres container to run the tests
	postgresContainer, err := postgres.Run(ctx,
		"postgres:14-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		return nil, err
	}

	port, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		return nil, err
	}

	// set environment variables
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", port.Port())
	os.Setenv("DB_USER", "user")
	os.Setenv("DB_PASS", "password")
	os.Setenv("DB_NAME", "test")
	os.Setenv("JWT_SECRET", "secret")
	os.Setenv("JWT_REFRESH_SECRET", "refresh_secret")
	os.Setenv("FRONTEND_HOST", "http://localhost:1234")

	// set temporary directories
	videosDir, err := os.MkdirTemp("/tmp", "ganymede-tests")
	if err != nil {
		return nil, err
	}
	os.Setenv("VIDEOS_DIR", videosDir)
	t.Log("VIDEOS_DIR", videosDir)

	tempDir, err := os.MkdirTemp("/tmp", "ganymede-tests")
	if err != nil {
		return nil, err
	}
	os.Setenv("TEMP_DIR", tempDir)
	t.Log("TEMP_DIR", tempDir)

	configDir, err := os.MkdirTemp("/tmp", "ganymede-tests")
	if err != nil {
		return nil, err
	}
	os.Setenv("CONFIG_DIR", configDir)
	t.Log("CONFIG_DIR", configDir)

	logsDir, err := os.MkdirTemp("/tmp", "ganymede-tests")
	if err != nil {
		return nil, err
	}
	os.Setenv("LOGS_DIR", logsDir)
	t.Log("LOGS_DIR", logsDir)

	// create the application. this does not start the HTTP server
	app, err := server.SetupApplication(ctx)
	if err != nil {
		return nil, err
	}

	return app, nil
}
