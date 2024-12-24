package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/config"
)

// GetConfig godoc
//
//	@Summary		Get config
//	@Description	Get config
//	@Tags			config
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	config.Conf
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/config [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetConfig(c echo.Context) error {
	config := config.Get()
	return SuccessResponse(c, config, "config")
}

// UpdateConfig godoc
//
//	@Summary		Update config
//	@Description	Update config
//	@Tags			config
//	@Accept			json
//	@Produce		json
//	@Param			body	body		UpdateConfigRequest	true	"Config"
//	@Success		200		{object}	UpdateConfigRequest
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/config [put]
//	@Security		ApiKeyCookieAuth
func (h *Handler) UpdateConfig(c echo.Context) error {
	conf := new(config.Config)
	if err := c.Bind(conf); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(conf); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	err := config.UpdateConfig(conf)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return SuccessResponse(c, conf, "config updated")
}
