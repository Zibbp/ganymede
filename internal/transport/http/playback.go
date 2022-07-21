package http

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/auth"
	"net/http"
)

type PlaybackService interface {
	UpdateProgress(c *auth.CustomContext, vID uuid.UUID, time int) error
	GetProgress(c *auth.CustomContext, vID uuid.UUID) (*ent.Playback, error)
	GetAllProgress(c *auth.CustomContext) ([]*ent.Playback, error)
}

type UpdateProgressRequest struct {
	VodID string `json:"vod_id" validate:"required"`
	Time  int    `json:"time" validate:"required"`
}

func (h *Handler) UpdateProgress(c echo.Context) error {
	cc := c.(*auth.CustomContext)
	upr := new(UpdateProgressRequest)
	if err := c.Bind(upr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(upr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	vID, err := uuid.Parse(upr.VodID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid vod id")
	}
	err = h.Service.PlaybackService.UpdateProgress(cc, vID, upr.Time)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, "ok")
}

func (h *Handler) GetProgress(c echo.Context) error {
	cc := c.(*auth.CustomContext)
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid vod id")
	}
	playbackEntry, err := h.Service.PlaybackService.GetProgress(cc, vID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, playbackEntry)
}

func (h *Handler) GetAllProgress(c echo.Context) error {
	cc := c.(*auth.CustomContext)
	playbackEntries, err := h.Service.PlaybackService.GetAllProgress(cc)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, playbackEntries)
}
