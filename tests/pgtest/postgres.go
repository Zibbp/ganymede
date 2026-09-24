// Package pgtest provides isolated PostgreSQL without platform credentials or workers.
package pgtest

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func ConnectionString(t *testing.T) string {
	t.Helper()
	ctx := t.Context()
	container, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase("ganymede"), postgres.WithUsername("ganymede"), postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(time.Minute)))
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, container)
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	return dsn
}

func Open(t *testing.T) *sql.DB {
	t.Helper()
	ctx := t.Context()
	dsn := ConnectionString(t)
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.PingContext(ctx))
	return db
}
