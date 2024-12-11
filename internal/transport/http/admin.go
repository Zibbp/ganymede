package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/admin"
)

type AdminService interface {
	GetStats(ctx context.Context) (admin.GetStatsResp, error)
	GetInfo(ctx context.Context) (admin.InfoResp, error)
}

// GetStats godoc
//
//	@Summary		Get ganymede stats
//	@Description	Get ganymede stats
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	admin.GetStatsResp
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/admin/stats [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetStats(c echo.Context) error {
	resp, err := h.Service.AdminService.GetStats(c.Request().Context())
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("Error retrieving stats: %v", err))
	}
	return SuccessResponse(c, resp, "Statistics")
}

// GetInfo godoc
//
//	@Summary		Get ganymede info
//	@Description	Get ganymede info
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	admin.InfoResp
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/admin/info [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetInfo(c echo.Context) error {
	resp, err := h.Service.AdminService.GetInfo(c.Request().Context())
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("Error retrieving Ganymede information: %v", err))
	}
	return SuccessResponse(c, resp, "Ganymede information")
}
