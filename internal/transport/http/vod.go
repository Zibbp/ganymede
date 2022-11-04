package http

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
	"net/http"
	"strconv"
	"time"
)

type VodService interface {
	CreateVod(vod vod.Vod, cID uuid.UUID) (*ent.Vod, error)
	GetVods(c echo.Context) ([]*ent.Vod, error)
	GetVodsByChannel(c echo.Context, cUUID uuid.UUID) ([]*ent.Vod, error)
	GetVod(vID uuid.UUID) (*ent.Vod, error)
	GetVodWithChannel(vID uuid.UUID) (*ent.Vod, error)
	DeleteVod(c echo.Context, vID uuid.UUID) error
	UpdateVod(c echo.Context, vID uuid.UUID, vod vod.Vod, cID uuid.UUID) (*ent.Vod, error)
	SearchVods(c echo.Context, query string) ([]*ent.Vod, error)
	GetVodPlaylists(c echo.Context, vID uuid.UUID) ([]*ent.Playlist, error)
	GetVodsPagination(c echo.Context, limit int, offset int, channelId uuid.UUID) (vod.Pagination, error)
}

type CreateVodRequest struct {
	ID               string            `json:"id"`
	ChannelID        string            `json:"channel_id" validate:"required"`
	ExtID            string            `json:"ext_id" validate:"min=1"`
	Platform         utils.VodPlatform `json:"platform" validate:"required,oneof=twitch youtube"`
	Type             utils.VodType     `json:"type" validate:"required,oneof=archive live highlight upload clip"`
	Title            string            `json:"title" validate:"required,min=1"`
	Duration         int               `json:"duration" validate:"required"`
	Views            int               `json:"views" validate:"required"`
	Resolution       string            `json:"resolution"`
	Processing       bool              `json:"processing"`
	ThumbnailPath    string            `json:"thumbnail_path"`
	WebThumbnailPath string            `json:"web_thumbnail_path" validate:"required,min=1"`
	VideoPath        string            `json:"video_path" validate:"required,min=1"`
	ChatPath         string            `json:"chat_path"`
	ChatVideoPath    string            `json:"chat_video_path"`
	InfoPath         string            `json:"info_path"`
	StreamedAt       string            `json:"streamed_at" validate:"required"`
}

func (h *Handler) CreateVod(c echo.Context) error {
	var req CreateVodRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	cUUID, err := uuid.Parse(req.ChannelID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	// Parse streamed at time
	streamedAt, err := time.Parse(time.RFC3339, req.StreamedAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	var vodID uuid.UUID
	if req.ID == "" {
		vodID = uuid.New()
	} else {
		vID, err := uuid.Parse(req.ID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		_, err = h.Service.VodService.GetVod(vID)
		if err == nil {
			return echo.NewHTTPError(http.StatusConflict, "vod already exists")
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
		StreamedAt:       streamedAt,
	}

	v, err := h.Service.VodService.CreateVod(cvrDto, cUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, v)
}

func (h *Handler) GetVods(c echo.Context) error {
	cID := c.QueryParam("channel_id")
	if cID == "" {
		v, err := h.Service.VodService.GetVods(c)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, v)
	}
	cUUID, err := uuid.Parse(cID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid channel id")
	}
	v, err := h.Service.VodService.GetVodsByChannel(c, cUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, v)
}

func (h *Handler) GetVod(c echo.Context) error {
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	wC := c.QueryParam("with_channel")
	if wC == "true" {
		v, err := h.Service.VodService.GetVodWithChannel(vID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, v)
	}
	v, err := h.Service.VodService.GetVod(vID)
	if err != nil {
		if err.Error() == "vod not found" {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, v)
}

func (h *Handler) DeleteVod(c echo.Context) error {
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	err = h.Service.VodService.DeleteVod(c, vID)
	if err != nil {
		if err.Error() == "vod not found" {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusOK)
}

func (h *Handler) UpdateVod(c echo.Context) error {
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	var req CreateVodRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	cUUID, err := uuid.Parse(req.ChannelID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	// Parse streamed at time
	streamedAt, err := time.Parse(time.RFC3339, req.StreamedAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
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
		StreamedAt:       streamedAt,
	}

	v, err := h.Service.VodService.UpdateVod(c, vID, cvrDto, cUUID)
	if err != nil {
		if err.Error() == "vod not found" {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, v)
}

func (h *Handler) SearchVods(c echo.Context) error {
	q := c.QueryParam("q")
	if q == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "q is required")
	}
	v, err := h.Service.VodService.SearchVods(c, q)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, v)
}

func (h *Handler) GetVodPlaylists(c echo.Context) error {
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	v, err := h.Service.VodService.GetVodPlaylists(c, vID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, v)
}

func (h *Handler) GetVodsPagination(c echo.Context) error {
	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("invalid limit: %w", err).Error())
	}
	offset, err := strconv.Atoi(c.QueryParam("offset"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("invalid offset: %w", err).Error())
	}

	cID := c.QueryParam("channel_id")
	cUUID := uuid.Nil

	if cID != "" {
		cUUID, err = uuid.Parse(cID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
	}

	v, err := h.Service.VodService.GetVodsPagination(c, limit, offset, cUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, v)
}
