package http

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
)

type BlockedVodService interface {
	IsVodBlocked(ctx context.Context, id string) (bool, error)
	CreateBlockedVod(ctx context.Context, id string) error
	DeleteBlockedVod(ctx context.Context, id string) error
	GetBlockedVods(ctx context.Context) ([]string, error)
}

func (h *Handler) IsVodBlocked(c echo.Context) error {
	id := c.Param("id")

	blocked, err := h.Service.BlockedVodService.IsVodBlocked(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, blocked)
}

func (h *Handler) CreateBlockedVod(c echo.Context) error {
	id := c.Param("id")

	err := h.Service.BlockedVodService.CreateBlockedVod(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, nil)
}

func (h *Handler) DeleteBlockedVod(c echo.Context) error {
	id := c.Param("id")

	err := h.Service.BlockedVodService.DeleteBlockedVod(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, nil)
}

func (h *Handler) GetBlockedVods(c echo.Context) error {
	vods, err := h.Service.BlockedVodService.GetBlockedVods(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, vods)
}
