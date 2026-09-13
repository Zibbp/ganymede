package http

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
)

type BlockedVideoService interface {
	IsVideoBlocked(ctx context.Context, id string) (bool, error)
	CreateBlockedVideo(ctx context.Context, id string) error
	DeleteBlockedVideo(ctx context.Context, id string) error
	GetBlockedVideos(ctx context.Context) ([]*ent.BlockedVideos, error)
}

type ID struct {
	ID string `json:"id" validate:"required"`
}

// IsVideoBlocked godoc
//
//	@Summary		Check if video is blocked
//	@Description	Check if a video is on the blocked list
//	@Tags			blocked-video
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Video ID"
//	@Success		200	{object}	bool
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/blocked-video/{id} [get]
func (h *Handler) IsVideoBlocked(c echo.Context) error {
	id := c.Param("id")

	err := h.Server.Validator.Validate(ID{ID: id})
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	blocked, err := h.Service.BlockedVideoService.IsVideoBlocked(c.Request().Context(), id)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, blocked, "is video blocked")
}

// CreateBlockedVideo godoc
//
//	@Summary		Block a video
//	@Description	Add a video to the blocked list
//	@Tags			blocked-video
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Video ID"
//	@Success		200	{object}	string
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/blocked-video/{id} [post]
//	@Security		ApiKeyCookieAuth
//	@Security		ApiKeyAuth
func (h *Handler) CreateBlockedVideo(c echo.Context) error {
	id := c.Param("id")

	err := h.Server.Validator.Validate(ID{ID: id})
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	err = h.Service.BlockedVideoService.CreateBlockedVideo(c.Request().Context(), id)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, "", "blocked video")
}

// DeleteBlockedVideo godoc
//
//	@Summary		Unblock a video
//	@Description	Remove a video from the blocked list
//	@Tags			blocked-video
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Video ID"
//	@Success		200	{object}	string
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/blocked-video/{id} [delete]
//	@Security		ApiKeyCookieAuth
//	@Security		ApiKeyAuth
func (h *Handler) DeleteBlockedVideo(c echo.Context) error {
	id := c.Param("id")

	err := h.Server.Validator.Validate(ID{ID: id})
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	err = h.Service.BlockedVideoService.DeleteBlockedVideo(c.Request().Context(), id)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, "", "unblocked video")
}

// GetBlockedVideos godoc
//
//	@Summary		Get blocked videos
//	@Description	Get all blocked videos
//	@Tags			blocked-video
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}		ent.BlockedVideos
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/blocked-video [get]
func (h *Handler) GetBlockedVideos(c echo.Context) error {
	videos, err := h.Service.BlockedVideoService.GetBlockedVideos(c.Request().Context())
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, videos, "blocked videos")
}
