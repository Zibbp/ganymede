package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/utils"
)

type ArchiveService interface {
	ArchiveTwitchChannel(cName string) (*ent.Channel, error)
	ArchiveTwitchVod(vID string, quality string, chat bool, renderChat bool) (*archive.TwitchVodResponse, error)
}

type ArchiveChannelRequest struct {
	ChannelName string `json:"channel_name" validate:"required"`
}
type ArchiveVodRequest struct {
	VodID      string           `json:"vod_id" validate:"required"`
	Quality    utils.VodQuality `json:"quality" validate:"required,oneof=best source 720p60 480p30 360p30 160p30 480p 360p 160p audio"`
	Chat       bool             `json:"chat"`
	RenderChat bool             `json:"render_chat"`
}

// ArchiveTwitchChannel godoc
//
//	@Summary		Archive a twitch channel
//	@Description	Archive a twitch channel (creates channel in database and download profile image)
//	@Tags			archive
//	@Accept			json
//	@Produce		json
//	@Param			channel	body		ArchiveChannelRequest	true	"Channel"
//	@Success		200		{object}	ent.Channel
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/archive/channel [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) ArchiveTwitchChannel(c echo.Context) error {
	acr := new(ArchiveChannelRequest)
	if err := c.Bind(acr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(acr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	channel, err := h.Service.ArchiveService.ArchiveTwitchChannel(acr.ChannelName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, channel)
}

// ArchiveTwitchVod godoc
//
//	@Summary		Archive a twitch vod
//	@Description	Archive a twitch vod
//	@Tags			archive
//	@Accept			json
//	@Produce		json
//	@Param			vod	body		ArchiveVodRequest	true	"Vod"
//	@Success		200	{object}	archive.TwitchVodResponse
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/archive/vod [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) ArchiveTwitchVod(c echo.Context) error {
	avr := new(ArchiveVodRequest)
	if err := c.Bind(avr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(avr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	vod, err := h.Service.ArchiveService.ArchiveTwitchVod(avr.VodID, string(avr.Quality), avr.Chat, avr.RenderChat)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, vod)
}
