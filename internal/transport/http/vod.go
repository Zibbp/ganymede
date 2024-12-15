package http

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/riverqueue/river/rivertype"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/chat"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
)

type VodService interface {
	CreateVod(vod vod.Vod, cID uuid.UUID) (*ent.Vod, error)
	GetVods(c echo.Context) ([]*ent.Vod, error)
	GetVodsByChannel(c echo.Context, cUUID uuid.UUID) ([]*ent.Vod, error)
	GetVod(ctx context.Context, vID uuid.UUID, withChannel bool, withChapters bool, withMutedSegments bool, withQueue bool) (*ent.Vod, error)
	GetVodByExternalId(ctx context.Context, externalId string) (*ent.Vod, error)
	DeleteVod(c echo.Context, vID uuid.UUID, deleteFiles bool) error
	UpdateVod(c echo.Context, vID uuid.UUID, vod vod.Vod, cID uuid.UUID) (*ent.Vod, error)
	SearchVods(c echo.Context, query string, limit int, offset int, types []utils.VodType) (vod.Pagination, error)
	GetVodPlaylists(c echo.Context, vID uuid.UUID) ([]*ent.Playlist, error)
	GetVodsPagination(c echo.Context, limit int, offset int, channelId uuid.UUID, types []utils.VodType, playlistId uuid.UUID) (vod.Pagination, error)
	GetVodChatComments(c echo.Context, vodID uuid.UUID, start float64, end float64) (*[]chat.Comment, error)
	GetUserIdFromChat(c echo.Context, vodID uuid.UUID) (*int64, error)
	GetChatEmotes(ctx context.Context, vodID uuid.UUID) (*platform.Emotes, error)
	GetChatBadges(ctx context.Context, vodID uuid.UUID) (*platform.Badges, error)
	GetNumberOfVodChatCommentsFromTime(c echo.Context, vodID uuid.UUID, start float64, commentCount int64) (*[]chat.Comment, error)
	LockVod(c echo.Context, vID uuid.UUID, status bool) error
	GenerateStaticThumbnail(ctx context.Context, videoID uuid.UUID) (*rivertype.JobInsertResult, error)
}

type CreateVodRequest struct {
	ID               string              `json:"id"`
	ChannelID        string              `json:"channel_id" validate:"required"`
	ExtID            string              `json:"ext_id" validate:"min=1"`
	Platform         utils.VideoPlatform `json:"platform" validate:"required,oneof=twitch youtube"`
	Type             utils.VodType       `json:"type" validate:"required,oneof=archive live highlight upload clip"`
	Title            string              `json:"title" validate:"required,min=1"`
	Duration         int                 `json:"duration" validate:"required"`
	Views            int                 `json:"views" validate:"required"`
	Resolution       string              `json:"resolution"`
	Processing       bool                `json:"processing"`
	ThumbnailPath    string              `json:"thumbnail_path"`
	WebThumbnailPath string              `json:"web_thumbnail_path" validate:"required,min=1"`
	VideoPath        string              `json:"video_path" validate:"required,min=1"`
	ChatPath         string              `json:"chat_path"`
	ChatVideoPath    string              `json:"chat_video_path"`
	InfoPath         string              `json:"info_path"`
	CaptionPath      string              `json:"caption_path"`
	StreamedAt       string              `json:"streamed_at" validate:"required"`
	Locked           bool                `json:"locked"`
}

// CreateVod godoc
//
//	@Summary		Create a vod
//	@Description	Create a vod
//	@Tags			vods
//	@Accept			json
//	@Produce		json
//	@Param			body	body		CreateVodRequest	true	"Create vod request"
//	@Success		201		{object}	ent.Vod
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		409		{object}	utils.ErrorResponse
//	@Router			/vod [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) CreateVod(c echo.Context) error {
	var req CreateVodRequest
	if err := c.Bind(&req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	cUUID, err := uuid.Parse(req.ChannelID)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	// Parse streamed at time
	streamedAt, err := time.Parse(time.RFC3339, req.StreamedAt)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	var vodID uuid.UUID
	if req.ID == "" {
		vodID = uuid.New()
	} else {
		vID, err := uuid.Parse(req.ID)
		if err != nil {
			return ErrorResponse(c, http.StatusBadRequest, err.Error())
		}
		_, err = h.Service.VodService.GetVod(c.Request().Context(), vID, false, false, false, false)
		if err == nil {
			return ErrorResponse(c, http.StatusConflict, "vod already exists")
		}
		vodID = vID
	}

	cvrDto := vod.Vod{
		ID:               vodID,
		ExtID:            req.ExtID,
		Platform:         req.Platform,
		Type:             req.Type,
		Title:            req.Title,
		Duration:         req.Duration,
		Views:            req.Views,
		Resolution:       req.Resolution,
		Processing:       req.Processing,
		ThumbnailPath:    req.ThumbnailPath,
		WebThumbnailPath: req.WebThumbnailPath,
		VideoPath:        req.VideoPath,
		ChatPath:         req.ChatPath,
		ChatVideoPath:    req.ChatVideoPath,
		InfoPath:         req.InfoPath,
		CaptionPath:      req.CaptionPath,
		StreamedAt:       streamedAt,
		Locked:           req.Locked,
	}

	v, err := h.Service.VodService.CreateVod(cvrDto, cUUID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, v, "video created")
}

// GetVods godoc
//
//	@Summary		Get vods
//	@Description	Get vods
//	@Tags			vods
//	@Accept			json
//	@Produce		json
//	@Param			channel_id	query		string	false	"Channel ID"
//	@Success		200			{object}	[]ent.Vod
//	@Failure		400			{object}	utils.ErrorResponse
//	@Failure		500			{object}	utils.ErrorResponse
//	@Router			/vod [get]
func (h *Handler) GetVods(c echo.Context) error {
	cID := c.QueryParam("channel_id")
	if cID == "" {
		v, err := h.Service.VodService.GetVods(c)
		if err != nil {
			return ErrorResponse(c, http.StatusInternalServerError, err.Error())
		}
		return SuccessResponse(c, v, "videos")
	}
	cUUID, err := uuid.Parse(cID)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid channel id")
	}
	v, err := h.Service.VodService.GetVodsByChannel(c, cUUID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, v, "videos")
}

// GetVod godoc
//
//		@Summary		Get a vod
//		@Description	Get a vod
//		@Tags			vods
//		@Accept			json
//		@Produce		json
//		@Param			id				path		string	true	"Vod ID"
//		@Param			with_channel	query		string	false	"With channel"
//	 	@Param			with_chapters	query		string	false	"With chapters"
//		@Param			with_muted_segments	query	string	false	"With muted segments"
//		@Success		200				{object}	ent.Vod
//		@Failure		400				{object}	utils.ErrorResponse
//		@Failure		404				{object}	utils.ErrorResponse
//		@Failure		500				{object}	utils.ErrorResponse
//		@Router			/vod/{id} [get]
func (h *Handler) GetVod(c echo.Context) error {
	var id string
	var videoUUID uuid.UUID

	if c.Param("id") != "" {
		id = c.Param("id")

		vUUID, err := uuid.Parse(id)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid UUID provided: "+err.Error())
		}
		videoUUID = vUUID
	} else if c.Param("external_id") != "" {
		id = c.Param("external_id")

		v, err := h.Service.VodService.GetVodByExternalId(c.Request().Context(), id)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "VOD not found by external ID: "+err.Error())
		}
		videoUUID = v.ID
	} else {
		// If neither "id" nor "external_id" is provided, return an error
		return echo.NewHTTPError(http.StatusBadRequest, "Either 'id' or 'external_id' must be provided")
	}

	withChannel := false
	withChapters := false
	withMutedSegments := false
	withQueue := false

	wC := c.QueryParam("with_channel")
	if wC == "true" {
		withChannel = true
	}

	wChapters := c.QueryParam("with_chapters")
	if wChapters == "true" {
		withChapters = true
	}

	wMutedSegments := c.QueryParam("with_muted_segments")
	if wMutedSegments == "true" {
		withMutedSegments = true
	}

	wQueue := c.QueryParam("with_queue")
	if wQueue == "true" {
		withQueue = true
	}

	v, err := h.Service.VodService.GetVod(c.Request().Context(), videoUUID, withChannel, withChapters, withMutedSegments, withQueue)
	if err != nil {
		if err.Error() == "vod not found" {
			return ErrorResponse(c, http.StatusNotFound, err.Error())
		}
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, v, "video")
}

// DeleteVod godoc
//
//	@Summary		Delete a vod
//	@Description	Delete a vod
//	@Tags			vods
//	@Accept			json
//	@Produce		json
//	@Param			id	path	string	true	"Vod ID"
//	@Param			delete_files	query	string	false	"Delete files"
//	@Success		200
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/vod/{id} [delete]
//	@Security		ApiKeyCookieAuth
func (h *Handler) DeleteVod(c echo.Context) error {
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	// get query param of delete_files if exists
	deleteFiles := false
	dF := c.QueryParam("delete_files")
	if dF == "true" {
		deleteFiles = true
	}
	err = h.Service.VodService.DeleteVod(c, vID, deleteFiles)
	if err != nil {
		if err.Error() == "vod not found" {
			return ErrorResponse(c, http.StatusNotFound, err.Error())
		}
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusOK)
}

// UpdateVod godoc
//
//	@Summary		Update a vod
//	@Description	Update a vod
//	@Tags			vods
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Vod ID"
//	@Param			body	body		CreateVodRequest	true	"Vod"
//	@Success		200		{object}	ent.Vod
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		404		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/vod/{id} [put]
//	@Security		ApiKeyCookieAuth
func (h *Handler) UpdateVod(c echo.Context) error {
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	var req CreateVodRequest
	if err := c.Bind(&req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	cUUID, err := uuid.Parse(req.ChannelID)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	// Parse streamed at time
	streamedAt, err := time.Parse(time.RFC3339, req.StreamedAt)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	cvrDto := vod.Vod{
		ExtID:            req.ExtID,
		Platform:         req.Platform,
		Type:             req.Type,
		Title:            req.Title,
		Duration:         req.Duration,
		Views:            req.Views,
		Resolution:       req.Resolution,
		Processing:       req.Processing,
		ThumbnailPath:    req.ThumbnailPath,
		WebThumbnailPath: req.WebThumbnailPath,
		VideoPath:        req.VideoPath,
		ChatPath:         req.ChatPath,
		ChatVideoPath:    req.ChatVideoPath,
		InfoPath:         req.InfoPath,
		CaptionPath:      req.CaptionPath,
		StreamedAt:       streamedAt,
		Locked:           req.Locked,
	}

	v, err := h.Service.VodService.UpdateVod(c, vID, cvrDto, cUUID)
	if err != nil {
		if err.Error() == "vod not found" {
			return ErrorResponse(c, http.StatusNotFound, err.Error())
		}
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, v, "video updated")
}

// SearchVods godoc
//
//	@Summary		Search vods
//	@Description	Search vods
//	@Tags			vods
//	@Accept			json
//	@Produce		json
//	@Param			q		query		string	true	"Search query"
//	@Param			limit	query		integer	false	"Limit"		default(10)
//	@Param			offset	query		integer	false	"Offset"	default(0)
//	@Success		200		{array}		ent.Vod
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/vod/search [get]
func (h *Handler) SearchVods(c echo.Context) error {
	q := c.QueryParam("q")
	if q == "" {
		return ErrorResponse(c, http.StatusBadRequest, "q is required")
	}
	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid limit: %w", err).Error())
	}
	offset, err := strconv.Atoi(c.QueryParam("offset"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid offset: %w", err).Error())
	}
	vTypes := c.QueryParam("types")
	var types []utils.VodType
	if vTypes != "" {
		for _, vType := range strings.Split(vTypes, ",") {
			types = append(types, utils.VodType(vType))
		}
	}

	v, err := h.Service.VodService.SearchVods(c, q, limit, offset, types)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, v, "videos")
}

// GetVodPlaylists godoc
//
//	@Summary		Get vod playlists
//	@Description	Get vod playlists
//	@Tags			vods
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Vod ID"
//	@Success		200	{array}		[]ent.Playlist
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/vod/{id}/playlist [get]
func (h *Handler) GetVodPlaylists(c echo.Context) error {
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	v, err := h.Service.VodService.GetVodPlaylists(c, vID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, v, "playlists video is in")
}

// GetVodsPagination godoc
//
//	@Summary		Get vods pagination
//	@Description	Get vods pagination
//	@Tags			vods
//	@Accept			json
//	@Produce		json
//	@Param			limit		query		integer	false	"Limit"		default(10)
//	@Param			offset		query		integer	false	"Offset"	default(0)
//	@Param			channel_id	query		string	false	"Channel ID"
//	@Success		200			{object}	vod.Pagination
//	@Failure		400			{object}	utils.ErrorResponse
//	@Failure		500			{object}	utils.ErrorResponse
//	@Router			/vod/pagination [get]
func (h *Handler) GetVodsPagination(c echo.Context) error {
	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid limit: %w", err).Error())
	}
	offset, err := strconv.Atoi(c.QueryParam("offset"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid offset: %w", err).Error())
	}

	cID := c.QueryParam("channel_id")
	cUUID := uuid.Nil
	if cID != "" {
		cUUID, err = uuid.Parse(cID)
		if err != nil {
			return ErrorResponse(c, http.StatusBadRequest, err.Error())
		}
	}

	playlistId := c.QueryParam("playlist_id")
	playlistUUID := uuid.Nil
	if playlistId != "" {
		playlistUUID, err = uuid.Parse(playlistId)
		if err != nil {
			return ErrorResponse(c, http.StatusBadRequest, err.Error())
		}
	}

	vTypes := c.QueryParam("types")
	var types []utils.VodType
	if vTypes != "" {
		for _, vType := range strings.Split(vTypes, ",") {
			types = append(types, utils.VodType(vType))
		}
	}

	v, err := h.Service.VodService.GetVodsPagination(c, limit, offset, cUUID, types, playlistUUID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, v, "paginated videos")
}

// GetUserIdFromChat godoc
//
//	@Summary		Get user id from chat
//	@Description	Get user id from chat json file
//	@Tags			vods
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Vod ID"
//	@Success		200	{object}	int64
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/vod/{id}/chat/userid [get]
func (h *Handler) GetUserIdFromChat(c echo.Context) error {
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	v, err := h.Service.VodService.GetUserIdFromChat(c, vID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, v, "user id from chat")
}

// GetVodChatComments godoc
//
//	@Summary		Get vod chat comments
//	@Description	Get vod chat comments
//	@Tags			vods
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string	true	"Vod ID"
//	@Param			start	query		string	false	"Start time"
//	@Param			end		query		string	false	"End time"
//	@Success		200		{array}		[]chat.Comment
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		404		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/vod/{id}/chat [get]
func (h *Handler) GetVodChatComments(c echo.Context) error {
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	start := c.QueryParam("start")
	end := c.QueryParam("end")
	startFloat, err := strconv.ParseFloat(start, 64)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid start: %w", err).Error())
	}
	endFloat, err := strconv.ParseFloat(end, 64)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid end: %w", err).Error())
	}

	v, err := h.Service.VodService.GetVodChatComments(c, vID, startFloat, endFloat)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, v, fmt.Sprintf("comments for %s %f - %f", vID, startFloat, endFloat))
}

// GetChatEmotes godoc
//
//	@Summary		Get vod chat emotes
//	@Description	Get vod chat emotes
//	@Tags			vods
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Vod ID"
//	@Success		200	{array}		chat.GanymedeEmotes
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/vod/{id}/chat/emotes [get]
func (h *Handler) GetChatEmotes(c echo.Context) error {
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	emotes, err := h.Service.VodService.GetChatEmotes(c.Request().Context(), vID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return SuccessResponse(c, emotes.Emotes, "video emotes")
}

// GetChatBadges godoc
//
//	@Summary		Get vod chat badges
//	@Description	Get vod chat badges
//	@Tags			vods
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Vod ID"
//	@Success		200	{array}		chat.GanymedeBadges
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/vod/{id}/chat/badges [get]
func (h *Handler) GetChatBadges(c echo.Context) error {
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	badges, err := h.Service.VodService.GetChatBadges(c.Request().Context(), vID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return SuccessResponse(c, badges.Badges, "video badges")
}

// GetNumberOfVodChatComments godoc
//
//	@Summary		Get number of vod chat comments
//	@Description	Get N number of vod chat comments before the start time (used for seeking)
//	@Tags			vods
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string	true	"Vod ID"
//	@Param			start	query		string	false	"Start time"
//	@Param			count	query		string	false	"Count"
//	@Success		200		{object}	[]chat.Comment
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		404		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/vod/{id}/chat/seek [get]
func (h *Handler) GetNumberOfVodChatCommentsFromTime(c echo.Context) error {
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	start := c.QueryParam("start")
	count := c.QueryParam("count")
	startFloat, err := strconv.ParseFloat(start, 64)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid start: %w", err).Error())
	}
	countInt, err := strconv.Atoi(count)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid count: %w", err).Error())
	}
	if countInt < 1 {
		return ErrorResponse(c, http.StatusBadRequest, "count must be greater than 0")
	}

	v, err := h.Service.VodService.GetNumberOfVodChatCommentsFromTime(c, vID, startFloat, int64(countInt))
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, v, fmt.Sprintf("comments for %s from %f", vID, startFloat))
}

func (h *Handler) LockVod(c echo.Context) error {
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	status := true
	param := c.QueryParam("locked")
	if param == "false" {
		status = false
	}
	err = h.Service.VodService.LockVod(c, vID, status)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, "", "video locked")
}

func (h *Handler) GenerateStaticThumbnail(c echo.Context) error {
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	job, err := h.Service.VodService.GenerateStaticThumbnail(c.Request().Context(), vID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, nil, fmt.Sprintf("job created: %d", job.Job.ID))
}
