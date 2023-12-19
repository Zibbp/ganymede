package temporal

import (
	"os"

	"github.com/rs/zerolog/log"

	"go.temporal.io/sdk/client"
)

var temporalClient *Temporal

type Temporal struct {
	Client client.Client
}

func InitializeTemporalClient() {
	// TODO: config env parsed
	temporalUrl := os.Getenv("TEMPORAL_URL")
	clientOptions := client.Options{
		HostPort: temporalUrl,
	}

	c, err := client.Dial(clientOptions)
	if err != nil {
		log.Panic().Msgf("Unable to create client: %v", err)
	}

	log.Info().Msgf("Connected to temporal at %s", clientOptions.HostPort)

	temporalClient = &Temporal{Client: c}
}

func GetTemporalClient() *Temporal {
	return temporalClient
}
