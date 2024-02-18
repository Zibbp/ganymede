package database

import (
	"context"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

var db *Database

type Database struct {
	Client *ent.Client
}

func InitializeDatabase(worker bool) {
	log.Debug().Msg("setting up database connection")

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")
	dbSSL := os.Getenv("DB_SSL")
	dbSSLTRootCert := os.Getenv("DB_SSL_ROOT_CERT")

	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s sslrootcert=%s",
		dbHost, dbPort, dbUser, dbPass, dbName, dbSSL, dbSSLTRootCert)

	client, err := ent.Open("postgres", connectionString)

	if err != nil {
		log.Fatal().Err(err).Msg("error connecting to database")
	}

	if !worker {
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
	db = &Database{Client: client}
}

func DB() *Database {
	return db
}

func NewDatabase() (*Database, error) {
	log.Debug().Msg("setting up database connection")

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")
	dbSSL := os.Getenv("DB_SSL")
	dbSSLTRootCert := os.Getenv("DB_SSL_ROOT_CERT")

	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s sslrootcert=%s",
		dbHost, dbPort, dbUser, dbPass, dbName, dbSSL, dbSSLTRootCert)

	client, err := ent.Open("postgres", connectionString)

	if err != nil {
		return nil, err
	}

	// Run auto migration
	if err := client.Schema.Create(context.Background()); err != nil {
		return nil, err
	}

	// check if any users exist
	users, err := client.User.Query().All(context.Background())
	if err != nil {
		return nil, err
	}
	// if no users exist, seed database
	if len(users) == 0 {
		// seed database
		log.Debug().Msg("seeding database")
		if err := seedDatabase(client); err != nil {
			return nil, err
		}
	}

	return &Database{Client: client}, nil
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
