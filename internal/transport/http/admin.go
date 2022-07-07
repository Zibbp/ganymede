package http

import (
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/admin"
	"net/http"
)

type AdminService interface {
	GetStats(c echo.Context) (admin.GetStatsResp, error)
}

func (h *Handler) GetStats(c echo.Context) error {
	resp, err := h.Service.AdminService.GetStats(c)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}
