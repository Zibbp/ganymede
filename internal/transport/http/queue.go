package http

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/queue"
	"net/http"
)

type QueueService interface {
	CreateQueueItem(c echo.Context, queueDto queue.Queue, vID uuid.UUID) (*ent.Queue, error)
	GetQueueItems(c echo.Context) ([]*ent.Queue, error)
}

type CreateQueueRequest struct {
	VodID string `json:"vod_id" validate:"required"`
}

func (h *Handler) CreateQueueItem(c echo.Context) error {
	cqt := new(CreateQueueRequest)
	if err := c.Bind(cqt); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(cqt); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	vID, err := uuid.Parse(cqt.VodID)
	if err != nil {
		return err
	}

	cqtDto := queue.Queue{}

	que, err := h.Service.QueueService.CreateQueueItem(c, cqtDto, vID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, que)
}

func (h *Handler) GetQueueItems(c echo.Context) error {
	q, err := h.Service.QueueService.GetQueueItems(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, q)
}
