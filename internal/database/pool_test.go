package database

import (
	"database/sql"
	"math"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Not covered here: the worker River pool wiring, which needs a live Postgres.
const testDSN = "user=u password=p host=127.0.0.1 port=1 dbname=d sslmode=disable"

func TestNewDatabaseRejectsInvalidMaxConns(t *testing.T) {
	t.Parallel()
	// Validation runs before connecting, so no database is needed.
	assert.Panics(t, func() {
		NewDatabase(t.Context(), DatabaseConnectionInput{DBString: testDSN, MaxConns: 1})
	})
}

func TestApplySQLPoolLimits(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		maxConns int
		want     int
	}{
		{"unset keeps unlimited", 0, 0},
		{"set caps pool", 3, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			db, err := sql.Open("pgx", testDSN)
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })

			ApplySQLPoolLimits(db, tc.maxConns)
			assert.Equal(t, tc.want, db.Stats().MaxOpenConnections)
		})
	}
}

func TestNewPgxPoolConfig(t *testing.T) {
	t.Parallel()
	cfg, err := NewPgxPoolConfig(testDSN, 0)
	require.NoError(t, err)
	def, err := pgxpool.ParseConfig(testDSN)
	require.NoError(t, err)
	assert.Equal(t, def.MaxConns, cfg.MaxConns)

	cfg, err = NewPgxPoolConfig(testDSN, 2)
	require.NoError(t, err)
	assert.Equal(t, int32(2), cfg.MaxConns)

	cfg, err = NewPgxPoolConfig(testDSN+" pool_min_conns=5", 2)
	require.NoError(t, err)
	assert.Equal(t, int32(2), cfg.MinConns)

	_, err = NewPgxPoolConfig(testDSN+" pool_max_conns=notanumber", 2)
	assert.Error(t, err)

	_, err = NewPgxPoolConfig(testDSN, -1)
	assert.Error(t, err)

	cfg, err = NewPgxPoolConfig(testDSN, math.MaxInt32)
	require.NoError(t, err)
	assert.Equal(t, int32(math.MaxInt32), cfg.MaxConns)

	// Values that would wrap when narrowed to int32 must be rejected.
	for _, n := range []int{math.MaxInt32 + 1, 4294967297, 4294967298} {
		_, err = NewPgxPoolConfig(testDSN, n)
		assert.Error(t, err, n)
	}
}

func TestValidateMaxConns(t *testing.T) {
	t.Parallel()
	assert.NoError(t, ValidateMaxConns(0))
	assert.NoError(t, ValidateMaxConns(MinPoolConns))
	assert.Error(t, ValidateMaxConns(1))
	assert.Error(t, ValidateMaxConns(-1))
	assert.NoError(t, ValidateMaxConns(MaxPoolConns))
	assert.Error(t, ValidateMaxConns(MaxPoolConns+1))
}
