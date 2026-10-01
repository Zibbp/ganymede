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

// GetCategories godoc
//
//	@Summary		Get twitch categories
//	@Description	Get cached twitch categories
//	@Tags			category
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}		ent.TwitchCategory
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/category [get]
func (h *Handler) GetCategories(c echo.Context) error {
	categories, err := h.Service.CategoryService.GetCategories(c.Request().Context())
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, categories, "twitch categories")
}
