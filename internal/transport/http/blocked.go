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

func (h *Handler) GetBlockedVideos(c echo.Context) error {
	videos, err := h.Service.BlockedVideoService.GetBlockedVideos(c.Request().Context())
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, videos, "blocked videos")
}
