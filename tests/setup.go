package tests

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	osExec "os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/zibbp/ganymede/internal/server"
	"github.com/zibbp/ganymede/internal/worker"
)

var (
	TestPostgresDatabase = "ganymede"
	TestPostgresUser     = "ganymede"
	TestPostgresPassword = "ganymede"
)

// setupPostgresTestContainer sets up a postgres container for testing
func setupPostgresTestContainer(ctx context.Context) (*postgres.PostgresContainer, error) {
	pgContainer, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase(TestPostgresDatabase),
		postgres.WithUsername(TestPostgresUser),
		postgres.WithPassword(TestPostgresPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)

	if err != nil {
		return nil, err
	}

	return pgContainer, nil
}

// getRootPath returns the root path of the project
// Git is the simplest way to get the root path of the project
func getRootPath(t *testing.T) string {
	e := osExec.Command("git", "rev-parse", "--show-toplevel")
	out, err := e.Output()
	if err != nil {
		t.Logf("Could not get root path: %v", err)
	}
	return strings.Trim(string(out), "\n")
}
func setupEnvironment(t *testing.T, postgresHost string, postgresPort string) {
	// Load .env file if available (for local development)
	envPath := filepath.Join(getRootPath(t), ".env")
	_ = godotenv.Load(envPath)

	t.Log(envPath)

	// Set the environment variables specific to the test
	assert.NoError(t, os.Setenv("TESTS_LOGGING", "true")) // Disable logging for tests
	assert.NoError(t, os.Setenv("DEBUG", "true"))
	assert.NoError(t, os.Setenv("DB_HOST", postgresHost))
	assert.NoError(t, os.Setenv("DB_PORT", postgresPort))
	assert.NoError(t, os.Setenv("DB_NAME", TestPostgresDatabase))
	assert.NoError(t, os.Setenv("DB_USER", TestPostgresUser))
	assert.NoError(t, os.Setenv("DB_PASS", TestPostgresPassword))

	// Set paths
	// set temporary directories
	videosDir, err := os.MkdirTemp("/tmp", "ganymede-tests")
	assert.NoError(t, err)
	assert.NoError(t, os.Setenv("VIDEOS_DIR", videosDir))
	t.Log("VIDEOS_DIR", videosDir)

	tempDir, err := os.MkdirTemp("/tmp", "ganymede-tests")
	assert.NoError(t, err)
	assert.NoError(t, os.Setenv("TEMP_DIR", tempDir))
	t.Log("TEMP_DIR", tempDir)

	configDir, err := os.MkdirTemp("/tmp", "ganymede-tests")
	assert.NoError(t, err)
	assert.NoError(t, os.Setenv("CONFIG_DIR", configDir))
	t.Log("CONFIG_DIR", configDir)

	logsDir, err := os.MkdirTemp("/tmp", "ganymede-tests")
	assert.NoError(t, err)
	assert.NoError(t, os.Setenv("LOGS_DIR", logsDir))
	t.Log("LOGS_DIR", logsDir)
}

// Setup initializes the integration test environment.
// It setups up the entire application and returns the various services for testing.
// A Postgres Testcontainer is used to provide a real database for further tersting.
// Used for service tests in internal/<service>/<service>_test.go
func Setup(t *testing.T) (*server.Application, error) {
	// Skip tests that require secret environment variables (e.g. if running in CI against a fork)
	if os.Getenv("SKIP_SECRET_TESTS") == "true" {
		t.Skip("Skipping test that requires secret environment variables")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	postgresContainer, err := setupPostgresTestContainer(ctx)
	if err != nil {
		t.Fatalf("Could not start postgres container: %v", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		Cleanup(t)
		if postgresContainer != nil {
			_ = postgresContainer.Terminate(ctx)
		}
	})

	postgresPort, err := postgresContainer.MappedPort(ctx, "5432")
	assert.NoError(t, err)

	postgresIp, err := postgresContainer.Host(ctx)
	assert.NoError(t, err)

	t.Log("Postgres IP:", postgresIp)

	// Setup env vars
	setupEnvironment(t, postgresIp, postgresPort.Port())

	// Create the application. This does not start the HTTP server
	app, err := server.SetupApplication(ctx)
	if err != nil {
		return nil, err
	}

	// Start worker
	workerClient, err := worker.SetupWorker(ctx)
	if err != nil {
		return nil, err
	}

	go func() {
		if err := workerClient.Start(); err != nil {
			log.Panic().Err(err).Msg("Error running river worker")
		}
	}()

	return app, nil

}

// SetupHTTP is similar to Setup but starts the HTTP server for testing end-to-end http requests.
// Used for tests in internal/transport/http
func SetupHTTP(t *testing.T) (*httpexpect.Expect, error) {
	// Skip tests that require secret environment variables (e.g. if running in CI against a fork)
	if os.Getenv("SKIP_SECRET_TESTS") == "true" {
		t.Skip("Skipping test that requires secret environment variables")
	}
	ctx := context.Background()

	postgresContainer, err := setupPostgresTestContainer(ctx)
	if err != nil {
		t.Fatalf("Could not start postgres container: %v", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		Cleanup(t)
		if postgresContainer != nil {
			_ = postgresContainer.Terminate(ctx)
		}
	})

	postgresPort, err := postgresContainer.MappedPort(ctx, "5432")
	assert.NoError(t, err)

	postgresIp, err := postgresContainer.Host(ctx)
	assert.NoError(t, err)

	t.Log("Postgres IP:", postgresIp)

	// Setup env vars
	setupEnvironment(t, postgresIp, postgresPort.Port())

	// Get free port for Ganymede to run on
	port, err := getFreePort()
	assert.NoError(t, err)
	assert.NoError(t, os.Setenv("APP_PORT", fmt.Sprintf("%d", port)))

	// Start the application
	go func() {
		err = server.Run(ctx)
		assert.NoError(t, err)
	}()

	// Wait for the application to start
	time.Sleep(5 * time.Second)

	e := httpexpect.WithConfig(httpexpect.Config{
		BaseURL:  fmt.Sprintf("http://localhost:%d/api/v1", port),
		Reporter: httpexpect.NewAssertReporter(t),
		Client: &http.Client{
			Jar: httpexpect.NewCookieJar(),
		},
	})

	// // Start worker
	// workerClient, err := worker.SetupWorker(ctx)
	// if err != nil {
	// 	return nil, err
	// }

	// go func() {
	// 	if err := workerClient.Start(); err != nil {
	// 		log.Panic().Err(err).Msg("Error running river worker")
	// 	}
	// }()

	return e, nil
}

// Cleanup cleans up the test environment
func Cleanup(t *testing.T) {
	// Cleanup temporary directories
	assert.NoError(t, os.RemoveAll(os.Getenv("VIDEOS_DIR")))
	assert.NoError(t, os.RemoveAll(os.Getenv("TEMP_DIR")))
	assert.NoError(t, os.RemoveAll(os.Getenv("CONFIG_DIR")))
	assert.NoError(t, os.RemoveAll(os.Getenv("LOGS_DIR")))
}

// getFreePort returns a free port on the host
func getFreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = listener.Close()
	}()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}
