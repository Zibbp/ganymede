package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/admin"
)

type AdminService interface {
	GetVideoStatistics(ctx context.Context) (admin.GetVideoStatisticsResponse, error)
	GetSystemOverview(ctx context.Context) (admin.GetSystemOverviewResponse, error)
	GetStorageDistribution(ctx context.Context) (admin.GetStorageDistributionResponse, error)
	GetInfo(ctx context.Context) (admin.InfoResp, error)
}

// GetVideoStatistics godoc
//
//	@Summary		Get Ganymede video statistics
//	@Description	Get Ganymede video statistics
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	admin.GetVideoStatisticsResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/admin/video/statistics [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetVideoStatistics(c echo.Context) error {
	resp, err := h.Service.AdminService.GetVideoStatistics(c.Request().Context())
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("Error retrieving video statistics: %v", err))
	}
	return SuccessResponse(c, resp, "Video Statistics")
}

// GetSystemOverview godoc
//
//	@Summary		Get system overview
//	@Description	Get system overview
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	admin.GetSystemOverviewResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/admin/system/overview [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetSystemOverview(c echo.Context) error {
	resp, err := h.Service.AdminService.GetSystemOverview(c.Request().Context())
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("Error retrieving system overview: %v", err))
	}
	return SuccessResponse(c, resp, "System Overview")
}

// GetStorageDistribution godoc
//
//	@Summary		Get storage distribution
//	@Description	Get storage distribution
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	admin.GetStorageDistributionResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/admin/storage-distribution [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetStorageDistribution(c echo.Context) error {
	resp, err := h.Service.AdminService.GetStorageDistribution(c.Request().Context())
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("Error retrieving storage distribution: %v", err))
	}
	return SuccessResponse(c, resp, "Storage Distribution")
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
