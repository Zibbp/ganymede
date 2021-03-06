package database

import (
	"context"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/utils"
	"golang.org/x/crypto/bcrypt"
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

	isSeeded := viper.Get("db_seeded").(bool)
	if !isSeeded {
		log.Debug().Msg("seeding database")
		if err := seedDatabase(client); err != nil {
			return nil, err
		}
		viper.Set("db_seeded", true)
		err := viper.WriteConfig()
		if err != nil {
			return nil, fmt.Errorf("error writing config: %v", err)
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
