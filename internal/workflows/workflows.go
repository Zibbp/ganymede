package workflows

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/temporal"
	"go.temporal.io/sdk/client"
)

type StartWorkflowResponse struct {
	WorkflowId string `json:"workflow_id"`
	RunId      string `json:"run_id"`
}

func StartWorkflow(ctx context.Context, workflowName string) (StartWorkflowResponse, error) {
	// TODO: develop a better way to do this

	var startWorkflowResponse StartWorkflowResponse

	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: "archive",
	}

	we, err := temporal.GetTemporalClient().Client.ExecuteWorkflow(ctx, workflowOptions, workflowName)
	if err != nil {
		log.Error().Err(err).Msg("failed to start workflow")
		return startWorkflowResponse, err
	}

	startWorkflowResponse.WorkflowId = we.GetID()
	startWorkflowResponse.RunId = we.GetRunID()

	return startWorkflowResponse, nil
}

func StartWorkflowGenerateThumbnailsForVideo(ctx context.Context, videoId string) (StartWorkflowResponse, error) {
	var startWorkflowResponse StartWorkflowResponse

	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: "generate-thumbnails",
	}

	we, err := temporal.GetTemporalClient().Client.ExecuteWorkflow(ctx, workflowOptions, GenerateThumbnailsForVideo, videoId)
	if err != nil {
		log.Error().Err(err).Msg("failed to start workflow")
		return startWorkflowResponse, err
	}

	startWorkflowResponse.WorkflowId = we.GetID()
	startWorkflowResponse.RunId = we.GetRunID()

	return startWorkflowResponse, nil
}
