package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	entLive "github.com/zibbp/ganymede/ent/live"
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
	SQLDB    *sql.DB
	ConnPool *pgxpool.Pool
}

const (
	migrationLockKey      int64 = 9876543210
	riverMigrationVersion       = 7
)

func DB() *Database {
	return db
}

// NewDatabase creates the application's database pools and runs all schema
// migrations. Both the API and worker call this function so either process can
// safely be the first one started after an upgrade.
func NewDatabase(ctx context.Context, input DatabaseConnectionInput) *Database {
	var sqlDB *sql.DB
	var err error
	maxRetries := 5
	retryDelay := time.Second * 3

	// Connect to the database with retries
	func() {
		for i := range maxRetries {
			sqlDB, err = sql.Open("pgx", input.DBString)
			if err == nil {
				err = sqlDB.PingContext(ctx)
			}
			if err == nil {
				return
			}
			if sqlDB != nil {
				_ = sqlDB.Close()
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
		log.Panic().Err(err).Msg("failed to open database")
	}

	// Ent and River's insertion-only client share this database/sql pool. This
	// is what makes domain writes and River inserts atomically composable.
	drv := entsql.OpenDB(dialect.Postgres, sqlDB)
	client := ent.NewClient(ent.Driver(drv))

	// The API keeps a pgx pool for its session store. Workers use their River
	// execution pool instead and do not need another duplicate pgx pool.
	var connPool *pgxpool.Pool
	if !input.IsWorker {
		connPool, err = pgxpool.New(ctx, input.DBString)
		if err != nil {
			_ = client.Close()
			log.Panic().Err(err).Msg("error creating pgx connection pool")
		}
	}

	if err := migrate(ctx, sqlDB, client); err != nil {
		_ = client.Close()
		if connPool != nil {
			connPool.Close()
		}
		log.Panic().Err(err).Msg("failed to migrate database")
	}

	// Only the API process seeds interactive application data. Schema
	// migration itself is deliberately identical in both process roles.
	if !input.IsWorker {
		users, err := client.User.Query().All(ctx)
		if err != nil {
			log.Panic().Err(err).Msg("error querying users")
		}
		if len(users) == 0 {
			log.Debug().Msg("seeding database")
			if err := seedDatabase(ctx, client); err != nil {
				log.Panic().Err(err).Msg("error seeding database")
			}
		}
	}

	db = &Database{
		Client:   client,
		SQLDB:    sqlDB,
		ConnPool: connPool,
	}

	return db
}

// migrate serializes Ent and River migrations with a transaction-scoped,
// application-owned advisory lock. Transaction-scoped locks cannot leak when a
// process exits or a context is cancelled.
func migrate(ctx context.Context, sqlDB *sql.DB, client *ent.Client) error {
	lockTx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration lock transaction: %w", err)
	}
	defer func() { _ = lockTx.Rollback() }()

	if _, err := lockTx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", migrationLockKey); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}

	hasLiveVodResolution := columnExists(ctx, sqlDB, entLive.Table, entLive.FieldVodResolution)
	if err := client.Schema.Create(ctx); err != nil {
		return fmt.Errorf("run Ent migrations: %w", err)
	}
	if !hasLiveVodResolution {
		backfillLiveVodResolution(ctx, sqlDB)
	}
	dropOrphanedColumns(ctx, sqlDB)

	riverMigrator, err := rivermigrate.New(riverdatabasesql.New(sqlDB), nil)
	if err != nil {
		return fmt.Errorf("create River migrator: %w", err)
	}
	if _, err := riverMigrator.Migrate(ctx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{TargetVersion: riverMigrationVersion}); err != nil {
		return fmt.Errorf("run River migrations through version %d: %w", riverMigrationVersion, err)
	}
	if _, err := riverMigrator.Validate(ctx, &rivermigrate.ValidateOpts{TargetVersion: riverMigrationVersion}); err != nil {
		return fmt.Errorf("validate River migrations through version %d: %w", riverMigrationVersion, err)
	}

	if err := lockTx.Commit(); err != nil {
		return fmt.Errorf("release migration lock: %w", err)
	}
	return nil
}

// WithTx runs Ent operations against the same database/sql transaction that
// can be passed to River's InsertTx method.
func (d *Database) WithTx(ctx context.Context, fn func(*ent.Client, *sql.Tx) error) error {
	tx, err := d.SQLDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	txDriver := &boundEntTxDriver{conn: entsql.Conn{ExecQuerier: tx}}
	txClient := ent.NewClient(ent.Driver(txDriver))
	if err := fn(txClient, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// boundEntTxDriver lets generated Ent builders use an existing database/sql
// transaction. Builders call Driver.Tx internally, so returning this same
// driver with no-op transaction methods keeps every statement on the outer
// transaction; WithTx remains the sole owner of commit and rollback.
type boundEntTxDriver struct {
	conn entsql.Conn
}

func (d *boundEntTxDriver) Exec(ctx context.Context, query string, args, v any) error {
	return d.conn.Exec(ctx, query, args, v)
}

func (d *boundEntTxDriver) Query(ctx context.Context, query string, args, v any) error {
	return d.conn.Query(ctx, query, args, v)
}

func (d *boundEntTxDriver) Tx(context.Context) (dialect.Tx, error) { return d, nil }
func (d *boundEntTxDriver) Dialect() string                        { return dialect.Postgres }
func (*boundEntTxDriver) Close() error                             { return nil }
func (*boundEntTxDriver) Commit() error                            { return nil }
func (*boundEntTxDriver) Rollback() error                          { return nil }

var _ dialect.Driver = (*boundEntTxDriver)(nil)

// Close releases every pool owned by Database.
func (d *Database) Close() error {
	if d == nil {
		return nil
	}
	if d.ConnPool != nil {
		d.ConnPool.Close()
	}
	if d.Client != nil {
		return d.Client.Close()
	}
	if d.SQLDB != nil {
		return d.SQLDB.Close()
	}
	return nil
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
type sqlExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type sqlQueryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func dropOrphanedColumns(ctx context.Context, conn sqlExecutor) {
	statements := []struct{ name, sql string }{
		{
			// Removed when ApiKey.scope (single ENUM) was replaced by
			// ApiKey.scopes (JSON list of resource:tier strings).
			name: "api_keys.scope",
			sql:  "ALTER TABLE api_keys DROP COLUMN IF EXISTS scope",
		},
	}
	for _, s := range statements {
		if _, err := conn.ExecContext(ctx, s.sql); err != nil {
			// Don't panic — a missing column or a permissions issue
			// shouldn't block the server from starting. Log and continue.
			log.Warn().Err(err).Str("statement", s.name).Msg("drop orphaned column failed")
		}
	}
}

func columnExists(ctx context.Context, conn sqlQueryRower, tableName, columnName string) bool {
	var exists bool
	err := conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = CURRENT_SCHEMA()
				AND table_name = $1
				AND column_name = $2
		)
	`, tableName, columnName).Scan(&exists)
	if err != nil {
		log.Warn().Err(err).Str("table", tableName).Str("column", columnName).Msg("column existence check failed")
		return false
	}
	return exists
}

func backfillLiveVodResolution(ctx context.Context, conn sqlExecutor) {
	stmt := fmt.Sprintf(
		`UPDATE %s SET %s = %s WHERE %s IS NULL OR %s = '' OR %s = 'best'`,
		entLive.Table,
		entLive.FieldVodResolution,
		entLive.FieldResolution,
		entLive.FieldVodResolution,
		entLive.FieldVodResolution,
		entLive.FieldVodResolution,
	)
	if _, err := conn.ExecContext(ctx, stmt); err != nil {
		log.Warn().Err(err).Msg("backfill live vod resolution failed")
	}
}

func seedDatabase(ctx context.Context, client *ent.Client) error {

	// Create initial user
	hashPass, err := bcrypt.GenerateFromPassword([]byte("ganymede"), 14)
	if err != nil {
		return fmt.Errorf("error hashing password: %v", err)
	}
	_, err = client.User.Create().SetUsername("admin").SetPassword(string(hashPass)).SetRole(utils.Role("admin")).Save(ctx)
	if err != nil {
		return fmt.Errorf("error creating user: %v", err)
	}

	return nil
}
