package temporal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/dto"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

type WorkflowHistory struct {
	*history.HistoryEvent
}

type WorkflowVideoIdResult struct {
	VideoId         string `json:"video_id"`
	ExternalVideoId string `json:"external_video_id"`
}

type WorkflowExecutionResponse struct {
	Executions    []*workflow.WorkflowExecutionInfo `json:"executions"`
	NextPageToken string                            `json:"next_page_token"`
}

func GetActiveWorkflows(ctx context.Context, inputPageToken []byte) (*WorkflowExecutionResponse, error) {
	listRequest := &workflowservice.ListOpenWorkflowExecutionsRequest{
		MaximumPageSize: 30,
	}

	if inputPageToken != nil {
		listRequest.NextPageToken = inputPageToken
	}

	w, err := temporalClient.Client.ListOpenWorkflow(ctx, listRequest)
	if err != nil {
		log.Error().Err(err).Msg("failed to list closed workflows")
		return nil, nil
	}

	var nextPageToken string
	if w.NextPageToken != nil {
		token := string(w.NextPageToken)
		// base64 encode
		nextPageToken = base64.StdEncoding.EncodeToString([]byte(token))
	}

	return &WorkflowExecutionResponse{
		Executions:    w.Executions,
		NextPageToken: nextPageToken,
	}, nil
}

func GetClosedWorkflows(ctx context.Context, inputPageToken []byte) (*WorkflowExecutionResponse, error) {
	listRequest := &workflowservice.ListClosedWorkflowExecutionsRequest{
		MaximumPageSize: 30,
	}

	if inputPageToken != nil {
		listRequest.NextPageToken = inputPageToken
	}

	w, err := temporalClient.Client.ListClosedWorkflow(ctx, listRequest)
	if err != nil {
		log.Error().Err(err).Msg("failed to list closed workflows")
		return nil, nil
	}

	var nextPageToken string
	if w.NextPageToken != nil {
		token := string(w.NextPageToken)
		// base64 encode
		nextPageToken = base64.StdEncoding.EncodeToString([]byte(token))
	}

	return &WorkflowExecutionResponse{
		Executions:    w.Executions,
		NextPageToken: nextPageToken,
	}, nil
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

func GetVideoIdFromWorkflow(ctx context.Context, workflowId string, runId string) (WorkflowVideoIdResult, error) {
	var result WorkflowVideoIdResult
	history, err := GetWorkflowHistory(ctx, workflowId, runId)
	if err != nil {
		return WorkflowVideoIdResult{}, err
	}

	for _, event := range history {
		if event.GetEventType() == enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED {
			attributes := event.GetWorkflowExecutionStartedEventAttributes()
			if attributes != nil {
				input := attributes.Input
				if input != nil {
					data := input.Payloads[0].GetData()
					var input dto.ArchiveVideoInput
					err := json.Unmarshal(data, &input)
					if err != nil {
						return WorkflowVideoIdResult{}, fmt.Errorf("failed to unmarshal input: %w", err)
					}
					result.VideoId = input.Vod.ID.String()
					result.ExternalVideoId = input.Vod.ExtID
				}
			}
		}
	}

	return result, nil
}
