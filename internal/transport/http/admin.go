package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/admin"
)

type AdminService interface {
	GetStats(c echo.Context) (admin.GetStatsResp, error)
	GetInfo(c echo.Context) (admin.InfoResp, error)
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
	resp, err := h.Service.AdminService.GetStats(c)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
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
	resp, err := h.Service.AdminService.GetInfo(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, resp)
}
