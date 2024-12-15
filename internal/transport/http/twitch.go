package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/platform"
)

type TwitchService interface {
	GetTwitchVideo(id string) (platform.VideoInfo, error)
	GetTwitchChannel(name string) (platform.ChannelInfo, error)
}

// GetTwitchChannel godoc
//
//	@Summary		Get a twitch channel
//	@Description	Get a twitch user/channel by name (uses twitch api)
//	@Tags			twitch
//	@Accept			json
//	@Produce		json
//	@Param			name	query		string	true	"Twitch user login name"
//	@Success		200		{object}	twitch.Channel
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/twitch/channel [get]
func (h *Handler) GetTwitchChannel(c echo.Context) error {
	name := c.QueryParam("name")
	if name == "" {
		return ErrorResponse(c, http.StatusBadRequest, "channel name query param is required")
	}
	channel, err := h.Service.PlatformTwitch.GetChannel(c.Request().Context(), name)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, channel, "channel")
}

// GetTwitchVideo godoc
//
//	@Summary		Get a twitch vod
//	@Description	Get a twitch vod by id (uses twitch api)
//	@Tags			twitch
//	@Accept			json
//	@Produce		json
//	@Param			id	query		string	true	"Twitch vod id"
//	@Success		200	{object}	twitch.Vod
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/twitch/video [get]
func (h *Handler) GetTwitchVideo(c echo.Context) error {
	vodID := c.QueryParam("id")
	if vodID == "" {
		return ErrorResponse(c, http.StatusBadRequest, "id query param is required")
	}
	vod, err := h.Service.PlatformTwitch.GetVideo(c.Request().Context(), vodID, true, true)
	if err != nil {
		if err.Error() == "vod not found" {
			return ErrorResponse(c, http.StatusNotFound, err.Error())
		}
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, vod, "video")
}
