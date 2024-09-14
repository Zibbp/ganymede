package database

import (
	"context"
	"fmt"

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

func NewDatabase(ctx context.Context, input DatabaseConnectionInput) *Database {
	client, err := ent.Open("postgres", input.DBString)

	if err != nil {
		log.Fatal().Err(err).Msg("error connecting to database")
	}

	if !input.IsWorker {
		// Run auto migration
		if err := client.Schema.Create(context.Background()); err != nil {
			log.Fatal().Err(err).Msg("error running auto migration")
		}
		// check if any users exist
		users, err := client.User.Query().All(context.Background())
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

	connPool, err := pgxpool.New(ctx, input.DBString)
	if err != nil {
		log.Panic().Err(err).Msg("error connecting to database")
	}
	// defer connPool.Close()

	db = &Database{
		Client:   client,
		ConnPool: connPool,
	}

	return db
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
