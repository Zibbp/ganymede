package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

var db *Database

type DatabaseConnectionInput struct {
	DBString string
	IsWorker bool
}

type Database struct {
	Client   *ent.Client
	ConnPool *pgxpool.Pool
}

func DB() *Database {
	return db
}

// NewDatabase creates a new database connection and runs auto migration if not a worker
func NewDatabase(ctx context.Context, input DatabaseConnectionInput) *Database {
	var client *ent.Client
	var err error
	maxRetries := 5
	retryDelay := time.Second * 3

	// Connect to the database with retries
	func() {
		for i := range maxRetries {
			client, err = ent.Open("postgres", input.DBString)
			if err == nil {
				return
			}
			log.Warn().Err(err).Msgf("error connecting to database, retrying (%d/%d)", i+1, maxRetries)

			if i == maxRetries-1 {
				return
			}

			timer := time.NewTimer(retryDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				err = fmt.Errorf("context cancelled during db connection retry: %w", ctx.Err())
				return
			case <-timer.C:
			}
		}
	}()

	if err != nil {
		log.Panic().Err(err).Msg("failed to open ent client")
	}

	// Create a pgx connection pool (used for advisory locking and other raw SQL)
	connPool, err := pgxpool.New(ctx, input.DBString)
	if err != nil {
		log.Panic().Err(err).Msg("error creating pgx connection pool")
	}

	// If this instance is responsible for migrations, acquire an advisory lock
	// so only one process runs schema migration at a time.
	if !input.IsWorker {
		const lockKey int64 = 9876543210 // arbitrary constant lock key
		conn, err := connPool.Acquire(ctx)
		if err != nil {
			client.Close() //nolint:errcheck
			connPool.Close()
			log.Panic().Err(err).Msg("failed to acquire pgx connection for migration lock")
		}
		// Ensure connection is released
		defer conn.Release()

		// Acquire exclusive advisory lock (blocks until obtained)
		if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
			client.Close() //nolint:errcheck
			connPool.Close()
			log.Panic().Err(err).Msg("failed to acquire advisory lock for migrations")
		}
		// Ensure we release the advisory lock once done
		defer func() {
			// use background context to ensure unlock happens even if original ctx is cancelled
			if _, err := conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", lockKey); err != nil {
				log.Warn().Err(err).Msg("failed to release advisory lock for migrations")
			}
		}()

		// Run auto migration (under lock)
		if err := client.Schema.Create(ctx); err != nil {
			log.Fatal().Err(err).Msg("error running auto migration")
		}

		// Post-migration housekeeping: drop columns that were removed from
		// the schema. ent's auto-migrate runs without WithDropColumn and
		// therefore leaves orphaned columns in place. Each statement here
		// must be idempotent (use IF EXISTS) and safe to repeat on every
		// boot. Runs under the same advisory lock as Schema.Create so two
		// servers booting simultaneously don't race.
		dropOrphanedColumns(ctx, conn)

		// check if any users exist
		users, err := client.User.Query().All(ctx)
		if err != nil {
			log.Panic().Err(err).Msg("error querying users")
		}
		// if no users exist, seed database
		if len(users) == 0 {
			// seed database
			log.Debug().Msg("seeding database")
			if err := seedDatabase(client); err != nil {
				log.Panic().Err(err).Msg("error seeding database")
			}
		}
	}

	db = &Database{
		Client:   client,
		ConnPool: connPool,
	}

	return db
}

// dropOrphanedColumns runs idempotent DROP COLUMN statements for fields
// that have been removed from the ent schema. ent's auto-migration is
// purposefully conservative — it never drops columns — so once a field
// is removed from ent/schema/*.go we still need to clean up the DB
// shape ourselves. Each statement uses IF EXISTS so it is safe to run
// on every boot, and on every existing deploy regardless of whether the
// column was ever present.
//
// Caller is expected to hold the migration advisory lock (see
// pg_advisory_lock above) so two booting servers don't race.
func dropOrphanedColumns(ctx context.Context, conn *pgxpool.Conn) {
	statements := []struct{ name, sql string }{
		{
			// Removed when ApiKey.scope (single ENUM) was replaced by
			// ApiKey.scopes (JSON list of resource:tier strings).
			name: "api_keys.scope",
			sql:  "ALTER TABLE api_keys DROP COLUMN IF EXISTS scope",
		},
	}
	for _, s := range statements {
		if _, err := conn.Exec(ctx, s.sql); err != nil {
			// Don't panic — a missing column or a permissions issue
			// shouldn't block the server from starting. Log and continue.
			log.Warn().Err(err).Str("statement", s.name).Msg("drop orphaned column failed")
		}
	}
}

func seedDatabase(client *ent.Client) error {

	// Create initial user
	hashPass, err := bcrypt.GenerateFromPassword([]byte("ganymede"), 14)
	if err != nil {
		return fmt.Errorf("error hashing password: %v", err)
	}
	_, err = client.User.Create().SetUsername("admin").SetPassword(string(hashPass)).SetRole(utils.Role("admin")).Save(context.Background())
	if err != nil {
		return fmt.Errorf("error creating user: %v", err)
	}

	return nil
}
