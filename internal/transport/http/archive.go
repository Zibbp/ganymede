package http

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/utils"
)

type ArchiveService interface {
	ArchiveTwitchChannel(cName string) (*ent.Channel, error)
	ArchiveTwitchVod(vID string, quality string, chat bool, renderChat bool) (*archive.TwitchVodResponse, error)
	RestartTask(c echo.Context, qID uuid.UUID, task string, cont bool) error
}

type ArchiveChannelRequest struct {
	ChannelName string `json:"channel_name" validate:"required"`
}
type ArchiveVodRequest struct {
	VodID      string           `json:"vod_id" validate:"required"`
	Quality    utils.VodQuality `json:"quality" validate:"required,oneof=best source 720p60 480p30 360p30 160p30"`
	Chat       bool             `json:"chat"`
	RenderChat bool             `json:"render_chat"`
}

type RestartTaskRequest struct {
	QueueID string `json:"queue_id" validate:"required"`
	Task    string `json:"task" validate:"required,oneof=vod_create_folder vod_download_thumbnail vod_save_info video_download video_convert video_move chat_download chat_convert chat_render chat_move"`
	Cont    bool   `json:"cont"`
}

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

func (h *Handler) RestartTask(c echo.Context) error {
	rtr := new(RestartTaskRequest)
	if err := c.Bind(rtr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(rtr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	qUUID, err := uuid.Parse(rtr.QueueID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	err = h.Service.ArchiveService.RestartTask(c, qUUID, rtr.Task, rtr.Cont)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusOK)
}
