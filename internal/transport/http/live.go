package http

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/utils"
)

type LiveService interface {
	GetLiveWatchedChannels(c echo.Context) ([]*ent.Live, error)
	AddLiveWatchedChannel(c echo.Context, liveDto live.Live) (*ent.Live, error)
	DeleteLiveWatchedChannel(c echo.Context, lID uuid.UUID) error
	UpdateLiveWatchedChannel(c echo.Context, liveDto live.Live) (*ent.Live, error)
	Check() error
	ConvertChat(c echo.Context, convertDto live.ConvertChat) error
	CheckVodWatchedChannels()
	ArchiveLiveChannel(c echo.Context, archiveDto live.ArchiveLive) error
}

type AddWatchedChannelRequest struct {
	WatchLive          bool     `json:"watch_live" `
	WatchVod           bool     `json:"watch_vod" `
	DownloadArchives   bool     `json:"download_archives" `
	DownloadHighlights bool     `json:"download_highlights" `
	DownloadUploads    bool     `json:"download_uploads"`
	ChannelID          string   `json:"channel_id" validate:"required"`
	Resolution         string   `json:"resolution" validate:"required,oneof=best source 720p60 480p30 360p30 160p30"`
	ArchiveChat        bool     `json:"archive_chat"`
	RenderChat         bool     `json:"render_chat"`
	DownloadSubOnly    bool     `json:"download_sub_only"`
	Categories         []string `json:"categories"`
}

type AddMultipleWatchedChannelRequest struct {
	WatchLive          bool     `json:"watch_live" `
	WatchVod           bool     `json:"watch_vod" `
	DownloadArchives   bool     `json:"download_archives" `
	DownloadHighlights bool     `json:"download_highlights" `
	DownloadUploads    bool     `json:"download_uploads"`
	ChannelID          []string `json:"channel_id" validate:"required"`
	Resolution         string   `json:"resolution" validate:"required,oneof=best source 720p60 480p30 360p30 160p30"`
	ArchiveChat        bool     `json:"archive_chat"`
	RenderChat         bool     `json:"render_chat"`
	DownloadSubOnly    bool     `json:"download_sub_only"`
	Categories         []string `json:"categories"`
}

type UpdateWatchedChannelRequest struct {
	WatchLive          bool     `json:"watch_live"`
	WatchVod           bool     `json:"watch_vod" `
	DownloadArchives   bool     `json:"download_archives" `
	DownloadHighlights bool     `json:"download_highlights" `
	DownloadUploads    bool     `json:"download_uploads"`
	Resolution         string   `json:"resolution" validate:"required,oneof=best source 720p60 480p30 360p30 160p30"`
	ArchiveChat        bool     `json:"archive_chat"`
	RenderChat         bool     `json:"render_chat"`
	DownloadSubOnly    bool     `json:"download_sub_only"`
	Categories         []string `json:"categories"`
}

type ConvertChatRequest struct {
	FileName      string `json:"file_name" validate:"required"`
	ChannelName   string `json:"channel_name" validate:"required"`
	VodID         string `json:"vod_id" validate:"required"`
	ChannelID     int    `json:"channel_id" validate:"required"`
	VodExternalID string `json:"vod_external_id" validate:"required"`
	ChatStart     string `json:"chat_start" validate:"required"`
}

type ArchiveLiveChannelRequest struct {
	ChannelID   string `json:"channel_id" validate:"required"`
	Resolution  string `json:"resolution" validate:"required,oneof=best source 720p60 480p 360p 160p 480p30 360p30 160p30"`
	ArchiveChat bool   `json:"archive_chat"`
	RenderChat  bool   `json:"render_chat"`
}

// GetLiveWatchedChannels godoc
//
//	@Summary		Get all watched channels
//	@Description	Get all watched channels
//	@Tags			Live
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	[]ent.Live
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/live [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetLiveWatchedChannels(c echo.Context) error {
	channels, err := h.Service.LiveService.GetLiveWatchedChannels(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, channels)
}

// AddLiveWatchedChannel godoc
//
//	@Summary		Add watched channel
//	@Description	Add watched channel
//	@Tags			Live
//	@Accept			json
//	@Produce		json
//	@Param			body	body		AddWatchedChannelRequest	true	"Add watched channel"
//	@Success		200		{object}	ent.Live
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/live [post]
//	@Security		ApiKeyCookieAuth
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
	liveDto := live.Live{
		ID:                 cUUID,
		WatchLive:          ccr.WatchLive,
		WatchVod:           ccr.WatchVod,
		DownloadArchives:   ccr.DownloadArchives,
		DownloadHighlights: ccr.DownloadHighlights,
		DownloadUploads:    ccr.DownloadUploads,
		IsLive:             false,
		ArchiveChat:        ccr.ArchiveChat,
		Resolution:         ccr.Resolution,
		RenderChat:         ccr.RenderChat,
		DownloadSubOnly:    ccr.DownloadSubOnly,
		Categories:         ccr.Categories,
	}
	l, err := h.Service.LiveService.AddLiveWatchedChannel(c, liveDto)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, l)
}

// AddMultipleLiveWatchedChannel godoc
//
//	@Summary		Add multiple watched channels at once
//	@Description	This is useful to add multiple channels at once if they all have the same settings
//	@Tags			Live
//	@Accept			json
//	@Produce		json
//	@Param			body	body		AddMultipleWatchedChannelRequest	true	"Add watched channel"
//	@Success		200		{object}	ent.Live
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/live/multiple [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) AddMultipleLiveWatchedChannel(c echo.Context) error {
	ccr := new(AddMultipleWatchedChannelRequest)
	if err := c.Bind(ccr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(ccr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var response []*ent.Live
	for _, cID := range ccr.ChannelID {
		cUUID, err := uuid.Parse(cID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		liveDto := live.Live{
			ID:                 cUUID,
			WatchLive:          ccr.WatchLive,
			WatchVod:           ccr.WatchVod,
			DownloadArchives:   ccr.DownloadArchives,
			DownloadHighlights: ccr.DownloadHighlights,
			DownloadUploads:    ccr.DownloadUploads,
			IsLive:             false,
			ArchiveChat:        ccr.ArchiveChat,
			Resolution:         ccr.Resolution,
			RenderChat:         ccr.RenderChat,
			DownloadSubOnly:    ccr.DownloadSubOnly,
			Categories:         ccr.Categories,
		}
		l, err := h.Service.LiveService.AddLiveWatchedChannel(c, liveDto)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		response = append(response, l)
	}

	return c.JSON(http.StatusOK, response)
}

// UpdateLiveWatchedChannel godoc
//
//	@Summary		Update watched channel
//	@Description	Update watched channel
//	@Tags			Live
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Channel ID"
//	@Param			body	body		UpdateWatchedChannelRequest	true	"Update watched channel"
//	@Success		200		{object}	ent.Live
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/live/{id} [put]
//	@Security		ApiKeyCookieAuth
func (h *Handler) UpdateLiveWatchedChannel(c echo.Context) error {
	id := c.Param("id")
	lID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	ccr := new(UpdateWatchedChannelRequest)
	if err := c.Bind(ccr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(ccr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	liveDto := live.Live{
		ID:                 lID,
		WatchLive:          ccr.WatchLive,
		WatchVod:           ccr.WatchVod,
		DownloadArchives:   ccr.DownloadArchives,
		DownloadHighlights: ccr.DownloadHighlights,
		DownloadUploads:    ccr.DownloadUploads,
		ArchiveChat:        ccr.ArchiveChat,
		Resolution:         ccr.Resolution,
		RenderChat:         ccr.RenderChat,
		DownloadSubOnly:    ccr.DownloadSubOnly,
		Categories:         ccr.Categories,
	}
	l, err := h.Service.LiveService.UpdateLiveWatchedChannel(c, liveDto)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, l)
}

// DeleteLiveWatchedChannel godoc
//
//	@Summary		Delete watched channel
//	@Description	Delete watched channel
//	@Tags			Live
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Channel ID"
//	@Success		200	{object}	ent.Live
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/live/{id} [delete]
//	@Security		ApiKeyCookieAuth
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

// ConvertChat godoc
//
//	@Summary		Convert chat
//	@Description	Adhoc convert chat endpoint. This is what happens when a live stream chat is converted to a "vod" chat.
//	@Tags			Live
//	@Accept			json
//	@Produce		json
//	@Param			body	body		ConvertChatRequest	true	"Convert chat"
//	@Success		200		{object}	string
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/live/chat-convert [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) ConvertChat(c echo.Context) error {
	ccr := new(ConvertChatRequest)
	if err := c.Bind(ccr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(ccr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	// Validate user input that is used for file name
	validVodID, err := utils.ValidateFileNameInput(ccr.VodID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	validVodExternalID, err := utils.ValidateFileNameInput(ccr.VodExternalID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	validFileName, err := utils.ValidateFileName(ccr.FileName)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	convertDto := live.ConvertChat{
		FileName:      validFileName,
		ChannelName:   ccr.ChannelName,
		VodID:         validVodID,
		ChannelID:     ccr.ChannelID,
		VodExternalID: validVodExternalID,
		ChatStart:     ccr.ChatStart,
	}
	err = h.Service.LiveService.ConvertChat(c, convertDto)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, "ok - converted chat found in /tmp/")
}

// CheckVodWatchedChannels godoc
//
//	@Summary		Check watched channels
//	@Description	Check watched channels if they are live. This is what runs every X seconds in the config.
//	@Tags			Live
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	string
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/live/check [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) CheckVodWatchedChannels(c echo.Context) error {
	go h.Service.LiveService.CheckVodWatchedChannels()

	return c.JSON(http.StatusOK, "ok")
}

// ArchiveLiveChannel godoc
//
//	@Summary		Archive a channel's live stream
//	@Description	Adhoc archive a channel's live stream.
//	@Tags			Live
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	string
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/live/archive [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) ArchiveLiveChannel(c echo.Context) error {
	alcr := new(ArchiveLiveChannelRequest)
	if err := c.Bind(alcr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(alcr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	// validate channel uuid
	cID, err := uuid.Parse(alcr.ChannelID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	archiveLiveDto := live.ArchiveLive{
		ChannelID:   cID,
		Resolution:  alcr.Resolution,
		ArchiveChat: alcr.ArchiveChat,
		RenderChat:  alcr.RenderChat,
	}

	err = h.Service.LiveService.ArchiveLiveChannel(c, archiveLiveDto)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, "ok")
}
