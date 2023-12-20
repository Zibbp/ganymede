package workflows

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/temporal"
	"go.temporal.io/sdk/client"
)

func StartWorkflow(ctx context.Context, workflowName string) (string, error) {
	// TODO: develop a better way to do this

	wfOptions := client.StartWorkflowOptions{
		ID:        workflowName,
		TaskQueue: "archive",
	}

	switch workflowName {
	case "save_chapters_for_twitch_videos":
		we, err := temporal.GetTemporalClient().Client.ExecuteWorkflow(ctx, wfOptions, SaveTwitchVideoChapters)
		if err != nil {
			log.Error().Err(err).Msg("failed to start workflow")
			return "", err
		}

		return we.GetID(), nil
	}

	return "", nil
}
