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
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	entChannel "github.com/zibbp/ganymede/ent/channel"
	entChapter "github.com/zibbp/ganymede/ent/chapter"
	"github.com/zibbp/ganymede/ent/predicate"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/chat"
	"github.com/zibbp/ganymede/internal/config"
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
	SearchVods(ctx context.Context, limit int, offset int, types []utils.VodType, predicates []predicate.Vod) (vod.Pagination, error)
	GetVodPlaylists(c echo.Context, vID uuid.UUID) ([]*ent.Playlist, error)
	GetVodsPagination(c echo.Context, limit int, offset int, channelId uuid.UUID, types []utils.VodType, playlistId uuid.UUID, processing bool) (vod.Pagination, error)
	GetVodChatComments(c echo.Context, vodID uuid.UUID, start float64, end float64) (*[]chat.Comment, error)
	GetUserIdFromChat(c echo.Context, vodID uuid.UUID) (*int64, error)
	GetChatEmotes(ctx context.Context, vodID uuid.UUID) (*platform.Emotes, error)
	GetChatBadges(ctx context.Context, vodID uuid.UUID) (*platform.Badges, error)
	GetNumberOfVodChatCommentsFromTime(c echo.Context, vodID uuid.UUID, start float64, commentCount int64) (*[]chat.Comment, error)
	LockVod(c echo.Context, vID uuid.UUID, status bool) error
	GenerateStaticThumbnail(ctx context.Context, videoID uuid.UUID) (*rivertype.JobInsertResult, error)
	GenerateSpriteThumbnails(ctx context.Context, videoID uuid.UUID) (*rivertype.JobInsertResult, error)
	GetVodClips(ctx context.Context, id uuid.UUID) ([]*ent.Vod, error)
	GetVodChatHistogram(ctx context.Context, vodID uuid.UUID, resolutionSeconds float64) (map[int]int, error)
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

type SearchQueryParams struct {
	Q      string   `query:"q" validate:"required"`
	Limit  int      `query:"limit" default:"10" validate:"number"`
	Offset int      `query:"offset" default:"0" validate:"number"`
	Fields []string `validate:"dive,oneof=title id ext_id chapter channel_name channel_id channel_ext_id"`
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
	// Parse query params
	var qp SearchQueryParams
	qp.Q = c.QueryParam("q")
	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid limit: %w", err).Error())
	}

	qp.Limit = limit
	offset, err := strconv.Atoi(c.QueryParam("offset"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, fmt.Errorf("invalid offset: %w", err).Error())
	}

	qp.Offset = offset
	vTypes := c.QueryParam("types")
	var types []utils.VodType
	if vTypes != "" {
		for _, vType := range strings.Split(vTypes, ",") {
			types = append(types, utils.VodType(vType))
		}
	}

	fieldsRaw := c.QueryParam("fields")
	var fields []string
	if fieldsRaw != "" {
		fields = strings.Split(fieldsRaw, ",")
	}
	qp.Fields = fields

	// Validate query params
	if err := c.Validate(&qp); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	// Default field to title if not provided
	if len(qp.Fields) == 0 {
		qp.Fields = []string{"title"}
	}

	// Create predicates based on the qp.Fields
	var predicates []predicate.Vod
	for _, field := range qp.Fields {
		switch field {
		case "title":
			predicates = append(predicates, entVod.TitleContainsFold(qp.Q))
		case "id":
			if id, err := uuid.Parse(qp.Q); err == nil {
				predicates = append(predicates, entVod.IDEQ(id))
			} else {
				log.Info().Msg("invalid id format in query")
			}
		case "ext_id":
			predicates = append(predicates, entVod.ExtIDContainsFold(qp.Q))
		case "chapter":
			predicates = append(predicates, entVod.HasChaptersWith(entChapter.TitleContainsFold(qp.Q)))
		case "channel_name":
			predicates = append(predicates, entVod.HasChannelWith(entChannel.NameContainsFold(qp.Q)))
		case "channel_id":
			if id, err := uuid.Parse(qp.Q); err == nil {
				predicates = append(predicates, entVod.HasChannelWith(entChannel.IDEQ(id)))
			} else {
				log.Info().Msg("invalid channel id format in query")
			}
		case "channel_ext_id":
			predicates = append(predicates, entVod.HasChannelWith(entChannel.ExtIDContainsFold(qp.Q)))
		default:
		}
	}

	// Default to title if no predicates are provided
	if len(predicates) == 0 {
		predicates = append(predicates, entVod.TitleContainsFold(qp.Q))
	}

	v, err := h.Service.VodService.SearchVods(c.Request().Context(), limit, offset, types, predicates)
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
//		@Summary		Get vods pagination
//		@Description	Get vods pagination
//		@Tags			vods
//		@Accept			json
//		@Produce		json
//		@Param			limit		query		integer	false	"Limit"		default(10)
//		@Param			offset		query		integer	false	"Offset"	default(0)
//		@Param			channel_id	query		string	false	"Channel ID"
//	 @Param			types		query		string	false	"Types"
//		@Param			playlist_id	query		string	false	"Playlist ID"
//		@Param			processing	query		string	false	"Processing. Set to false to exclude videos that are still processing."
//		@Success		200			{object}	vod.Pagination
//		@Failure		400			{object}	utils.ErrorResponse
//		@Failure		500			{object}	utils.ErrorResponse
//		@Router			/vod/pagination [get]
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

	// Default to true to include all videos. Only exclude processing videos is requested.
	isProcessing := c.QueryParam("processing") != "false"

	v, err := h.Service.VodService.GetVodsPagination(c, limit, offset, cUUID, types, playlistUUID, isProcessing)
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

func (h *Handler) GetVodClips(c echo.Context) error {
	id := c.Param("id")
	videoId, err := uuid.Parse(id)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "Invalid video ID")
	}
	clips, err := h.Service.VodService.GetVodClips(c.Request().Context(), videoId)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return SuccessResponse(c, clips, "clips for video")
}

func (h *Handler) GetVodSpriteThumbnails(c echo.Context) error {
	var id string
	var videoUUID uuid.UUID

	id = c.Param("id")

	videoUUID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid UUID provided: "+err.Error())
	}

	video, err := h.Service.VodService.GetVod(c.Request().Context(), videoUUID, false, false, false, false)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	if !video.SpriteThumbnailsEnabled {
		return ErrorResponse(c, http.StatusBadRequest, "Video does not have sprite thumbnails enabled.")
	}

	spriteMetata := SpriteMetadata{
		Duration:       video.Duration,
		SpriteImages:   video.SpriteThumbnailsImages,
		SpriteInterval: video.SpriteThumbnailsInterval,
		SpriteRows:     video.SpriteThumbnailsRows,
		SpriteColumns:  video.SpriteThumbnailsColumns,
		SpriteHeight:   video.SpriteThumbnailsHeight,
		SpriteWidth:    video.SpriteThumbnailsWidth,
	}
	webvtt, err := GenerateThumbnailsVTT(spriteMetata)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return c.String(http.StatusOK, webvtt)
}

type SpriteMetadata struct {
	Duration       int
	SpriteImages   []string
	SpriteInterval int
	SpriteRows     int
	SpriteColumns  int
	SpriteHeight   int
	SpriteWidth    int
}

func GenerateThumbnailsVTT(metadata SpriteMetadata) (string, error) {
	var builder strings.Builder

	// Write VTT header
	builder.WriteString("WEBVTT\n\n")

	// Calculate frame dimensions
	totalFrames := metadata.Duration / metadata.SpriteInterval

	frameIndex := 0

	cdnUrl := config.GetEnvConfig().CDN_URL

	// Generate VTT entries
	for _, imagePath := range metadata.SpriteImages {
		for row := 0; row < metadata.SpriteRows; row++ {
			for col := 0; col < metadata.SpriteColumns; col++ {
				start := frameIndex * metadata.SpriteInterval
				end := start + metadata.SpriteInterval
				if frameIndex >= totalFrames {
					break
				}

				startTime := formatTimestamp(start)
				endTime := formatTimestamp(end)
				x := col * metadata.SpriteWidth
				y := row * metadata.SpriteHeight

				entry := fmt.Sprintf("%s --> %s\n%s%s#xywh=%d,%d,%d,%d\n\n",
					startTime, endTime, cdnUrl, imagePath, x, y, metadata.SpriteWidth, metadata.SpriteHeight)

				builder.WriteString(entry)

				frameIndex++
			}
		}
		if frameIndex >= totalFrames {
			break
		}
	}

	return builder.String(), nil
}

func formatTimestamp(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	milliseconds := 0
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, secs, milliseconds)
}

func (h *Handler) GenerateSpriteThumbnails(c echo.Context) error {
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	job, err := h.Service.VodService.GenerateSpriteThumbnails(c.Request().Context(), vID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, nil, fmt.Sprintf("job created: %d", job.Job.ID))
}

func (h *Handler) GetVodChatHistogram(c echo.Context) error {
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	histogram, err := h.Service.VodService.GetVodChatHistogram(c.Request().Context(), vID, 60)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return SuccessResponse(c, histogram, "chat histogram")
}
