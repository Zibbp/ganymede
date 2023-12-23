package http

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/temporal"
	"github.com/zibbp/ganymede/internal/workflows"
)

type StartWorkflowRequest struct {
	WorkflowName string `json:"workflow_name" validate:"required"`
}
type RestartArchiveWorkflowRequest struct {
	WorkflowName string `json:"workflow_name" validate:"required"`
	VideoID      string `json:"video_id" validate:"required"`
}

func (h *Handler) GetActiveWorkflows(c echo.Context) error {
	executions, err := temporal.GetActiveWorkflows(c.Request().Context())
	if err != nil {
		return err
	}

	return c.JSON(200, executions)

}

func (h *Handler) GetClosedWorkflows(c echo.Context) error {
	executions, err := temporal.GetClosedWorkflows(c.Request().Context())
	if err != nil {
		return err
	}

	return c.JSON(200, executions)
}

func (h *Handler) GetWorkflowById(c echo.Context) error {
	workflowId := c.Param("workflowId")
	runId := c.Param("runId")

	execution, err := temporal.GetWorkflowById(c.Request().Context(), workflowId, runId)
	if err != nil {
		return err
	}

	return c.JSON(200, execution)
}

func (h *Handler) GetWorkflowHistory(c echo.Context) error {
	workflowId := c.Param("workflowId")
	runId := c.Param("runId")

	history, err := temporal.GetWorkflowHistory(c.Request().Context(), workflowId, runId)
	if err != nil {
		return err
	}

	return c.JSON(200, history)
}

func (h *Handler) StartWorkflow(c echo.Context) error {
	var request StartWorkflowRequest
	err := c.Bind(&request)
	if err != nil {
		return err
	}

	// validate request
	if err := c.Validate(request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	workflowId, err := workflows.StartWorkflow(c.Request().Context(), request.WorkflowName)
	if err != nil {
		return err
	}

	return c.JSON(200, map[string]string{
		"workflow_id": workflowId,
	})
}

func (h *Handler) RestartArchiveWorkflow(c echo.Context) error {
	var request RestartArchiveWorkflowRequest
	err := c.Bind(&request)
	if err != nil {
		return err
	}

	// validate request
	if err := c.Validate(request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// create uuid
	videoId, err := uuid.Parse(request.VideoID)
	if err != nil {
		return err
	}

	// some workflows should not be restarted such as live video and chat downloads
	if request.WorkflowName == "ArchiveTwitchLiveVideoWorkflow" || request.WorkflowName == "ArchiveTwitchLiveChatWorkflow" || request.WorkflowName == "	DownloadTwitchLiveChatWorkflow" || request.WorkflowName == "DownloadTwitchLiveVideoWorkflow" {
		return echo.NewHTTPError(http.StatusBadRequest, "cannot restart live video or chat workflows")
	}

	workflowId, err := temporal.RestartArchiveWorkflow(c.Request().Context(), videoId, request.WorkflowName)
	if err != nil {
		return err
	}

	return c.JSON(200, map[string]string{
		"workflow_id": workflowId,
	})
}

func (h *Handler) GetVideoIdFromWorkflow(c echo.Context) error {
	workflowId := c.Param("workflowId")
	runId := c.Param("runId")

	id, err := temporal.GetVideoIdFromWorkflow(c.Request().Context(), workflowId, runId)
	if err != nil {
		return err
	}

	return c.JSON(200, id)
}
