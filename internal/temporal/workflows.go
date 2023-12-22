package temporal

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/dto"
	"go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

type WorkflowHistory struct {
	*history.HistoryEvent
}

func GetActiveWorkflows(ctx context.Context) ([]*workflow.WorkflowExecutionInfo, error) {
	w, err := temporalClient.Client.ListOpenWorkflow(ctx, &workflowservice.ListOpenWorkflowExecutionsRequest{})
	if err != nil {
		log.Error().Err(err).Msg("failed to list open workflows")
		return nil, nil
	}

	return w.Executions, nil

}

func GetClosedWorkflows(ctx context.Context) ([]*workflow.WorkflowExecutionInfo, error) {
	w, err := temporalClient.Client.ListClosedWorkflow(ctx, &workflowservice.ListClosedWorkflowExecutionsRequest{})
	if err != nil {
		log.Error().Err(err).Msg("failed to list closed workflows")
		return nil, nil
	}

	return w.Executions, nil
}

func GetWorkflowById(ctx context.Context, workflowId string, runId string) (*workflow.WorkflowExecutionInfo, error) {
	w, err := temporalClient.Client.DescribeWorkflowExecution(ctx, workflowId, runId)
	if err != nil {
		log.Error().Err(err).Msg("failed to describe workflow")
		return nil, nil
	}

	return w.WorkflowExecutionInfo, nil
}

func GetWorkflowHistory(ctx context.Context, workflowId string, runId string) ([]*history.HistoryEvent, error) {
	iterator := temporalClient.Client.GetWorkflowHistory(ctx, workflowId, runId, false, 1)

	var history []*history.HistoryEvent
	for iterator.HasNext() {
		event, err := iterator.Next()
		if err != nil {
			log.Error().Err(err).Msg("failed to get workflow history")
			return nil, nil
		}

		history = append(history, event)
	}

	return history, nil
}

func RestartArchiveWorkflow(ctx context.Context, videoId uuid.UUID, workflowName string) (string, error) {
	// fetch items to create a dto.ArchiveVideoInput
	var input dto.ArchiveVideoInput

	vod, err := database.DB().Client.Vod.Query().Where(entVod.ID(videoId)).WithChannel().WithQueue().Only(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch vod")
		return "", nil
	}

	// check if a live watch exists
	liveWatch, err := vod.Edges.Channel.QueryLive().Only(context.Background())
	if err != nil {
		if _, ok := err.(*ent.NotFoundError); ok {
			log.Debug().Msg("no live watch found")
		} else {
			log.Error().Err(err).Msg("failed to fetch live watch")
			return "", nil
		}
	}

	input.Vod = vod
	input.Channel = vod.Edges.Channel
	input.Queue = vod.Edges.Queue
	input.VideoID = vod.ExtID
	input.Type = string(vod.Type)
	input.Platform = string(vod.Platform)
	input.Resolution = vod.Resolution
	input.RenderChat = input.Queue.RenderChat
	input.DownloadChat = true
	input.LiveWatchChannel = liveWatch

	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: "archive",
	}

	workflowRun, err := temporalClient.Client.ExecuteWorkflow(ctx, workflowOptions, workflowName, input)
	if err != nil {
		log.Error().Err(err).Msg("failed to start workflow")
		return "", nil
	}

	log.Info().Msgf("Started workflow %s", workflowRun.GetID())

	return workflowRun.GetID(), nil
}
