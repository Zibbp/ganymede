package http

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
)

type CategoryService interface {
	GetCategories(ctx context.Context) ([]*ent.TwitchCategory, error)
}

func (h *Handler) GetCategories(c echo.Context) error {
	categories, err := h.Service.CategoryService.GetCategories(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, categories)
}
