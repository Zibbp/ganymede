package database

import (
	"context"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"os"
)

type Database struct {
	Client *ent.Client
}

func NewDatabase() (*Database, error) {
	log.Debug().Msg("setting up database connection")

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")
	dbSSL := os.Getenv("DB_SSL")

	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPass, dbName, dbSSL)

	client, err := ent.Open("postgres", connectionString)

	if err != nil {
		return nil, err
	}

	// Run auto migration
	if err := client.Schema.Create(context.Background()); err != nil {
		return nil, err
	}

	return &Database{Client: client}, nil
}
