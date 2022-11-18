package http

import (
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/config"
	"net/http"
)

type ConfigService interface {
	GetConfig(c echo.Context) (*config.Conf, error)
	UpdateConfig(c echo.Context, conf *config.Conf) error
}

type UpdateConfigRequest struct {
	RegistrationEnabled bool   `json:"registration_enabled"`
	WebhookURL          string `json:"webhook_url"`
	DBSeeded            bool   `json:"db_seeded"`
	Parameters          struct {
		VideoConvert   string `json:"video_convert" validate:"required"`
		ChatRender     string `json:"chat_render" validate:"required"`
		StreamlinkLive string `json:"streamlink_live"`
	} `json:"parameters"`
}

func (h *Handler) GetConfig(c echo.Context) error {
	conf, err := h.Service.ConfigService.GetConfig(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, conf)
}

func (h *Handler) UpdateConfig(c echo.Context) error {
	conf := new(UpdateConfigRequest)
	if err := c.Bind(conf); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(conf); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	cDto := config.Conf{
		RegistrationEnabled: conf.RegistrationEnabled,
		WebhookURL:          conf.WebhookURL,
		DBSeeded:            conf.DBSeeded,
		Parameters: struct {
			VideoConvert   string `json:"video_convert"`
			ChatRender     string `json:"chat_render"`
			StreamlinkLive string `json:"streamlink_live"`
		}(conf.Parameters),
	}
	if err := h.Service.ConfigService.UpdateConfig(c, &cDto); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, conf)
}
