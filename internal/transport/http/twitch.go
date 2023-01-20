package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/twitch"
)

type TwitchService interface {
	GetVodByID(id string) (twitch.Vod, error)
}

func (h *Handler) GetTwitchUser(c echo.Context) error {
	name := c.QueryParam("name")
	if name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "channel name query param is required")
	}
	channel, err := twitch.GetUserByLogin(name)
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
		if err.Error() == "vod not found" {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, vod)
}

func (h *Handler) GQLGetTwitchVideo(c echo.Context) error {
	videoID := c.QueryParam("id")
	if videoID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "id query param is required")
	}
	video, err := twitch.GQLGetVideo(videoID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, video)
}
