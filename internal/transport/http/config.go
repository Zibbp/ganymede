package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/config"
)

type ConfigService interface {
	GetConfig(c echo.Context) (*config.Conf, error)
	UpdateConfig(c echo.Context, conf *config.Conf) error
	GetNotificationConfig(c echo.Context) (*config.Notification, error)
	UpdateNotificationConfig(c echo.Context, conf *config.Notification) error
}

type UpdateConfigRequest struct {
	RegistrationEnabled bool `json:"registration_enabled"`
	Parameters          struct {
		VideoConvert   string `json:"video_convert" validate:"required"`
		ChatRender     string `json:"chat_render" validate:"required"`
		StreamlinkLive string `json:"streamlink_live"`
	} `json:"parameters"`
	Archive struct {
		SaveAsHls bool `json:"save_as_hls"`
	} `json:"archive"`
}

type UpdateNotificationRequest struct {
	VideoSuccessWebhookUrl string `json:"video_success_webhook_url"`
	VideoSuccessTemplate   string `json:"video_success_template"`
	VideoSuccessEnabled    bool   `json:"video_success_enabled"`
	LiveSuccessWebhookUrl  string `json:"live_success_webhook_url"`
	LiveSuccessTemplate    string `json:"live_success_template"`
	LiveSuccessEnabled     bool   `json:"live_success_enabled"`
	ErrorWebhookUrl        string `json:"error_webhook_url"`
	ErrorTemplate          string `json:"error_template"`
	ErrorEnabled           bool   `json:"error_enabled"`
	IsLiveWebhookUrl       string `json:"is_live_webhook_url"`
	IsLiveTemplate         string `json:"is_live_template"`
	IsLiveEnabled          bool   `json:"is_live_enabled"`
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
		Archive: struct {
			SaveAsHls bool `json:"save_as_hls"`
		}(conf.Archive),
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

func (h *Handler) GetNotificationConfig(c echo.Context) error {
	conf, err := h.Service.ConfigService.GetNotificationConfig(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, conf)
}

func (h *Handler) UpdateNotificationConfig(c echo.Context) error {
	conf := new(UpdateNotificationRequest)
	if err := c.Bind(conf); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	cDto := config.Notification{
		VideoSuccessWebhookUrl: conf.VideoSuccessWebhookUrl,
		VideoSuccessTemplate:   conf.VideoSuccessTemplate,
		VideoSuccessEnabled:    conf.VideoSuccessEnabled,
		LiveSuccessWebhookUrl:  conf.LiveSuccessWebhookUrl,
		LiveSuccessTemplate:    conf.LiveSuccessTemplate,
		LiveSuccessEnabled:     conf.LiveSuccessEnabled,
		ErrorWebhookUrl:        conf.ErrorWebhookUrl,
		ErrorTemplate:          conf.ErrorTemplate,
		ErrorEnabled:           conf.ErrorEnabled,
		IsLiveWebhookUrl:       conf.IsLiveWebhookUrl,
		IsLiveTemplate:         conf.IsLiveTemplate,
		IsLiveEnabled:          conf.IsLiveEnabled,
	}

	if err := h.Service.ConfigService.UpdateNotificationConfig(c, &cDto); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, conf)
}
