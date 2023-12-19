package temporal

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
)

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
