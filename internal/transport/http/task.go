package http

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
)

type TaskService interface {
	StartTask(ctx context.Context, task string) error
}

type StartTaskRequest struct {
	Task string `json:"task" validate:"required,oneof=check_live check_vod check_clips get_jwks storage_migration prune_videos save_chapters update_stream_vod_ids generate_sprite_thumbnails update_video_storage_usage"`
}

// StartTask godoc
//
//	@Summary		Start a task
//	@Description	Start a task
//	@Tags			task
//	@Accept			json
//	@Produce		json
//	@Param			body	body	StartTaskRequest	true	"StartTaskRequest"
//	@Success		200
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/task/start [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) StartTask(c echo.Context) error {
	str := new(StartTaskRequest)
	if err := c.Bind(str); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(str); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := h.Service.TaskService.StartTask(c.Request().Context(), str.Task); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusOK)
}
