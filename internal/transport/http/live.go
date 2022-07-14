package http

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"net/http"
)

type LiveService interface {
	GetLiveWatchedChannels(c echo.Context) ([]*ent.Live, error)
	AddLiveWatchedChannel(c echo.Context, cID uuid.UUID) (*ent.Live, error)
	DeleteLiveWatchedChannel(c echo.Context, lID uuid.UUID) error
	Check() error
}

type AddWatchedChannelRequest struct {
	ChannelID string `json:"channel_id" validate:"required"`
}

func (h *Handler) GetLiveWatchedChannels(c echo.Context) error {
	channels, err := h.Service.LiveService.GetLiveWatchedChannels(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, channels)
}

func (h *Handler) AddLiveWatchedChannel(c echo.Context) error {
	ccr := new(AddWatchedChannelRequest)
	if err := c.Bind(ccr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(ccr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	cUUID, err := uuid.Parse(ccr.ChannelID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	l, err := h.Service.LiveService.AddLiveWatchedChannel(c, cUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, l)
}

func (h *Handler) DeleteLiveWatchedChannel(c echo.Context) error {
	id := c.Param("id")
	lID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	err = h.Service.LiveService.DeleteLiveWatchedChannel(c, lID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusOK)
}

func (h *Handler) Check(c echo.Context) error {
	err := h.Service.LiveService.Check()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, "ok")
}
