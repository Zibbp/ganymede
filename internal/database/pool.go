package database

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MinPoolConns is the smallest per-pool cap accepted by DB_MAX_CONNS. Startup
// migration holds one SQL connection for the advisory-lock transaction while
// the migration work itself needs another.
const MinPoolConns = 2

// MaxPoolConns is the largest cap pgxpool can represent (MaxConns is int32).
const MaxPoolConns = math.MaxInt32

// ValidateMaxConns checks a DB_MAX_CONNS value; 0 means "library defaults".
func ValidateMaxConns(maxConns int) error {
	if maxConns == 0 {
		return nil
	}
	if maxConns < MinPoolConns || maxConns > MaxPoolConns {
		return fmt.Errorf("DB_MAX_CONNS must be 0 (unset) or between %d and %d, got %d", MinPoolConns, MaxPoolConns, maxConns)
	}
	return nil
}

// ApplySQLPoolLimits caps open connections on a database/sql pool. Idle
// retention keeps the database/sql default. A non-positive maxConns leaves
// the pool untouched.
func ApplySQLPoolLimits(db *sql.DB, maxConns int) {
	if maxConns <= 0 {
		return
	}
	db.SetMaxOpenConns(maxConns)
}

// NewPgxPoolConfig parses dsn and caps MaxConns when maxConns is positive.
// An invalid maxConns returns an error.
func NewPgxPoolConfig(dsn string, maxConns int) (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pgx pool config: %w", err)
	}
	if err := ValidateMaxConns(maxConns); err != nil {
		return nil, err
	}
	if maxConns > 0 {
		cfg.MaxConns = int32(maxConns)
		if cfg.MinConns > cfg.MaxConns {
			cfg.MinConns = cfg.MaxConns
		}
	}
	return cfg, nil
}

// NewPgxPool creates a pgx pool capped at maxConns when maxConns is positive.
func NewPgxPool(ctx context.Context, dsn string, maxConns int) (*pgxpool.Pool, error) {
	cfg, err := NewPgxPoolConfig(dsn, maxConns)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}
