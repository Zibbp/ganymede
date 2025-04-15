package http

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/live"
)

type LiveService interface {
	GetLiveWatchedChannels(c echo.Context) ([]*ent.Live, error)
	AddLiveWatchedChannel(c echo.Context, liveDto live.Live) (*ent.Live, error)
	DeleteLiveWatchedChannel(c echo.Context, lID uuid.UUID) error
	UpdateLiveWatchedChannel(c echo.Context, liveDto live.Live) (*ent.Live, error)
	Check(ctx context.Context) error
	// ArchiveLiveChannel(c echo.Context, archiveDto live.ArchiveLive) error
}

type AddWatchedChannelRequest struct {
	WatchLive              bool                `json:"watch_live" validate:"boolean"`
	WatchVod               bool                `json:"watch_vod" validate:"boolean"`
	DownloadArchives       bool                `json:"download_archives" validate:"boolean"`
	DownloadHighlights     bool                `json:"download_highlights" validate:"boolean"`
	DownloadUploads        bool                `json:"download_uploads" validate:"boolean"`
	ChannelID              string              `json:"channel_id" validate:"required"`
	Resolution             string              `json:"resolution" validate:"required,oneof=best 1080p 720p 480p 360p 160p audio"`
	ArchiveChat            bool                `json:"archive_chat" validate:"boolean"`
	RenderChat             bool                `json:"render_chat" validate:"boolean"`
	DownloadSubOnly        bool                `json:"download_sub_only" validate:"boolean"`
	Categories             []string            `json:"categories"`
	ApplyCategoriesToLive  bool                `json:"apply_categories_to_live" validate:"boolean"`
	VideoAge               int64               `json:"video_age"` // restrict fetching videos to a certain age
	Regex                  []AddLiveTitleRegex `json:"regex"`
	WatchClips             bool                `json:"watch_clips" validate:"boolean"`
	ClipsLimit             int                 `json:"clips_limit" validate:"number,gte=1"`
	ClipsIntervalDays      int                 `json:"clips_interval_days" validate:"number,gte=1"`
	ClipsIgnoreLastChecked bool                `json:"clips_ignore_last_checked" validate:"boolean"`
}

type AddLiveTitleRegex struct {
	Regex         string `json:"regex" validate:"required"`
	Negative      bool   `json:"negative" validate:"boolean"`
	ApplyToVideos bool   `json:"apply_to_videos" validate:"boolean"`
}

type UpdateWatchedChannelRequest struct {
	WatchLive              bool                `json:"watch_live" validate:"boolean"`
	WatchVod               bool                `json:"watch_vod" validate:"boolean"`
	DownloadArchives       bool                `json:"download_archives" validate:"boolean"`
	DownloadHighlights     bool                `json:"download_highlights" validate:"boolean"`
	DownloadUploads        bool                `json:"download_uploads" validate:"boolean"`
	Resolution             string              `json:"resolution" validate:"required,oneof=best 1080p 720p 480p 360p 160p audio"`
	ArchiveChat            bool                `json:"archive_chat" validate:"boolean"`
	RenderChat             bool                `json:"render_chat" validate:"boolean"`
	DownloadSubOnly        bool                `json:"download_sub_only" validate:"boolean"`
	Categories             []string            `json:"categories"`
	ApplyCategoriesToLive  bool                `json:"apply_categories_to_live" validate:"boolean"`
	VideoAge               int64               `json:"video_age"` // retrict fetching videos to a certain age
	Regex                  []AddLiveTitleRegex `json:"regex"`
	WatchClips             bool                `json:"watch_clips" validate:"boolean"`
	ClipsLimit             int                 `json:"clips_limit" validate:"number,gte=1"`
	ClipsIntervalDays      int                 `json:"clips_interval_days" validate:"number,gte=1"`
	ClipsIgnoreLastChecked bool                `json:"clips_ignore_last_checked" validate:"boolean"`
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
	ChannelID              string `json:"channel_id" validate:"required"`
	Resolution             string `json:"resolution" validate:"required,oneof=best 1080p 720p 480p 360p 160p audio"`
	ArchiveChat            bool   `json:"archive_chat"`
	RenderChat             bool   `json:"render_chat"`
	WatchClips             bool   `json:"watch_clips" validate:"boolean"`
	ClipsLimit             int    `json:"clips_limit" validate:"number,gte=1"`
	ClipsIntervalDays      int    `json:"clips_interval_days" validate:"number,gte=1"`
	ClipsIgnoreLastChecked bool   `json:"clips_ignore_last_checked" validate:"boolean"`
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
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return SuccessResponse(c, channels, "wathed channels")
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
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(ccr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	cUUID, err := uuid.Parse(ccr.ChannelID)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	if len(ccr.Categories) == 0 && ccr.ApplyCategoriesToLive {
		return ErrorResponse(c, http.StatusBadRequest, "categories cannot be empty if apply_categories_to_live is true")
	}

	liveDto := live.Live{
		ID:                     cUUID,
		WatchLive:              ccr.WatchLive,
		WatchVod:               ccr.WatchVod,
		DownloadArchives:       ccr.DownloadArchives,
		DownloadHighlights:     ccr.DownloadHighlights,
		DownloadUploads:        ccr.DownloadUploads,
		IsLive:                 false,
		ArchiveChat:            ccr.ArchiveChat,
		Resolution:             ccr.Resolution,
		RenderChat:             ccr.RenderChat,
		DownloadSubOnly:        ccr.DownloadSubOnly,
		Categories:             ccr.Categories,
		ApplyCategoriesToLive:  ccr.ApplyCategoriesToLive,
		VideoAge:               ccr.VideoAge,
		WatchClips:             ccr.WatchClips,
		ClipsLimit:             ccr.ClipsLimit,
		ClipsIntervalDays:      ccr.ClipsIntervalDays,
		ClipsIgnoreLastChecked: ccr.ClipsIgnoreLastChecked,
	}

	for _, regex := range ccr.Regex {
		if err := c.Validate(regex); err != nil {
			return ErrorResponse(c, http.StatusBadRequest, err.Error())
		}
		liveDto.TitleRegex = append(liveDto.TitleRegex, ent.LiveTitleRegex{
			Negative:      regex.Negative,
			Regex:         regex.Regex,
			ApplyToVideos: regex.ApplyToVideos,
		})
	}

	l, err := h.Service.LiveService.AddLiveWatchedChannel(c, liveDto)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return SuccessResponse(c, l, "created watched channel")
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
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	ccr := new(UpdateWatchedChannelRequest)
	if err := c.Bind(ccr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(ccr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	if len(ccr.Categories) == 0 && ccr.ApplyCategoriesToLive {
		return ErrorResponse(c, http.StatusBadRequest, "categories cannot be empty if apply_categories_to_live is true")
	}

	liveDto := live.Live{
		ID:                     lID,
		WatchLive:              ccr.WatchLive,
		WatchVod:               ccr.WatchVod,
		DownloadArchives:       ccr.DownloadArchives,
		DownloadHighlights:     ccr.DownloadHighlights,
		DownloadUploads:        ccr.DownloadUploads,
		ArchiveChat:            ccr.ArchiveChat,
		Resolution:             ccr.Resolution,
		RenderChat:             ccr.RenderChat,
		DownloadSubOnly:        ccr.DownloadSubOnly,
		Categories:             ccr.Categories,
		ApplyCategoriesToLive:  ccr.ApplyCategoriesToLive,
		VideoAge:               ccr.VideoAge,
		WatchClips:             ccr.WatchClips,
		ClipsLimit:             ccr.ClipsLimit,
		ClipsIntervalDays:      ccr.ClipsIntervalDays,
		ClipsIgnoreLastChecked: ccr.ClipsIgnoreLastChecked,
	}

	for _, regex := range ccr.Regex {
		if err := c.Validate(regex); err != nil {
			return ErrorResponse(c, http.StatusBadRequest, err.Error())
		}
		liveDto.TitleRegex = append(liveDto.TitleRegex, ent.LiveTitleRegex{
			Negative:      regex.Negative,
			Regex:         regex.Regex,
			ApplyToVideos: regex.ApplyToVideos,
		})
	}

	l, err := h.Service.LiveService.UpdateLiveWatchedChannel(c, liveDto)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return SuccessResponse(c, l, "updated watched channel")
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
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	err = h.Service.LiveService.DeleteLiveWatchedChannel(c, lID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return c.NoContent(http.StatusOK)
}

func (h *Handler) Check(c echo.Context) error {
	err := h.Service.LiveService.Check(c.Request().Context())
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, "", "deleted watched channel")
}
