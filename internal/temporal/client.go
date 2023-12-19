package temporal

import (
	"github.com/rs/zerolog/log"

	"go.temporal.io/sdk/client"
)

var temporalClient *Temporal

type Temporal struct {
	Client client.Client
}

func InitializeTemporalClient() {
	clientOptions := client.Options{
		HostPort: "dev.tycho:7233",
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
