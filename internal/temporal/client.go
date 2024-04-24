package temporal

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/durationpb"

	"go.temporal.io/api/namespace/v1"
	"go.temporal.io/api/workflowservice/v1"
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

	// update temporal default namespace retention
	namespaceClient, err := client.NewNamespaceClient(clientOptions)
	if err != nil {
		log.Error().Msgf("Unable to create namespace client: %v", err)
	}

	// 30 day ttl
	retentionTtlTime := 30 * 24 * time.Hour

	retentionTtl := durationpb.Duration{
		Seconds: int64(retentionTtlTime.Seconds()),
	}

	err = namespaceClient.Update(context.Background(), &workflowservice.UpdateNamespaceRequest{
		Namespace: "default",
		Config: &namespace.NamespaceConfig{
			WorkflowExecutionRetentionTtl: &retentionTtl,
		},
	})
	if err != nil {
		log.Error().Msgf("Unable to update default namespace: %v", err)
	}

	log.Info().Msgf("Connected to temporal at %s", clientOptions.HostPort)

	temporalClient = &Temporal{Client: c}
}

func GetTemporalClient() *Temporal {
	return temporalClient
}
