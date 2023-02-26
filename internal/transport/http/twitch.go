package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/twitch"
)

type TwitchService interface {
	GetVodByID(id string) (twitch.Vod, error)
	GetCategories() ([]*ent.TwitchCategory, error)
}

// GetTwitchUser godoc
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
func (h *Handler) GetTwitchUser(c echo.Context) error {
	name := c.QueryParam("name")
	if name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "channel name query param is required")
	}
	channel, err := twitch.API.GetUserByLogin(name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, channel)
}

// GetTwitchVod godoc
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
//	@Router			/twitch/vod [get]
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

// GQLGetTwitchVideo godoc
//
//	@Summary		Get a twitch video
//	@Description	Get a twitch video by id (uses twitch graphql api)
//	@Tags				twitch
//	@Accept			json
//	@Produce		json
//	@Param			id	query		string	true	"Twitch video id"
//	@Success		200	{object}	twitch.Video
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/twitch/gql/video [get]
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

// GetTwitchCategories godoc
//
//	@Summary		Get a list of twitch categories
//	@Description	Get a list of twitch categories
//	@Tags			twitch
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	twitch.Category
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/twitch/categories [get]
func (h *Handler) GetTwitchCategories(c echo.Context) error {
	categories, err := h.Service.TwitchService.GetCategories()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, categories)
}
