package http

import (
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/utils"
	"net/http"
)

type ArchiveService interface {
	ArchiveTwitchChannel(c echo.Context, cName string) (*ent.Channel, error)
	ArchiveTwitchVod(c echo.Context, vID string, quality string, chat bool) (*archive.TwitchVodResponse, error)
}

type ArchiveChannelRequest struct {
	ChannelName string `json:"channel_name" validate:"required"`
}
type ArchiveVodRequest struct {
	VodID   string           `json:"vod_id" validate:"required"`
	Quality utils.VodQuality `json:"quality" validate:"required,oneof=source 720p60 480p30 360p30 160p30"`
	Chat    bool             `json:"chat"`
}

func (h *Handler) ArchiveTwitchChannel(c echo.Context) error {
	acr := new(ArchiveChannelRequest)
	if err := c.Bind(acr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(acr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	channel, err := h.Service.ArchiveService.ArchiveTwitchChannel(c, acr.ChannelName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, channel)
}

func (h *Handler) ArchiveTwitchVod(c echo.Context) error {
	avr := new(ArchiveVodRequest)
	if err := c.Bind(avr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(avr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	vod, err := h.Service.ArchiveService.ArchiveTwitchVod(c, avr.VodID, string(avr.Quality), avr.Chat)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, vod)
}
