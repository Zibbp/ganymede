package http

import (
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/twitch"
	"net/http"
)

type TwitchService interface {
	GetUserByLogin(c echo.Context, login string) (twitch.Channel, error)
	GetVodByID(id string) (twitch.Vod, error)
}

func (h *Handler) GetTwitchUser(c echo.Context) error {
	name := c.QueryParam("name")
	if name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "channel name query param is required")
	}
	channel, err := h.Service.TwitchService.GetUserByLogin(c, name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, channel)
}

func (h *Handler) GetTwitchVod(c echo.Context) error {
	vodID := c.QueryParam("id")
	if vodID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "id query param is required")
	}
	vod, err := h.Service.TwitchService.GetVodByID(vodID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, vod)
}
