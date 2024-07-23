package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/config"
)

type ConfigService interface {
	GetConfig(ctx context.Context) (*config.Conf, error)
	UpdateConfig(c echo.Context, conf *config.Conf) error
	GetNotificationConfig(c echo.Context) (*config.Notification, error)
	UpdateNotificationConfig(c echo.Context, conf *config.Notification) error
	GetStorageTemplateConfig(c echo.Context) (*config.StorageTemplate, error)
	UpdateStorageTemplateConfig(c echo.Context, conf *config.StorageTemplate) error
}

type UpdateConfigRequest struct {
	RegistrationEnabled bool `json:"registration_enabled"`
	Parameters          struct {
		TwitchToken    string `json:"twitch_token"`
		VideoConvert   string `json:"video_convert" validate:"required"`
		ChatRender     string `json:"chat_render" validate:"required"`
		StreamlinkLive string `json:"streamlink_live"`
	} `json:"parameters"`
	Archive struct {
		SaveAsHls bool `json:"save_as_hls"`
	} `json:"archive"`
	Livestream struct {
		Proxies         []config.ProxyListItem `json:"proxies"`
		ProxyEnabled    bool                   `json:"proxy_enabled"`
		ProxyParameters string                 `json:"proxy_parameters"`
		ProxyWhitelist  []string               `json:"proxy_whitelist"`
	} `json:"livestream"`
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

type UpdateStorageTemplateRequest struct {
	FolderTemplate string `json:"folder_template" validate:"required"`
	FileTemplate   string `json:"file_template" validate:"required"`
}

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
	conf, err := h.Service.ConfigService.GetConfig(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, conf)
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
			TwitchToken    string `json:"twitch_token"`
			VideoConvert   string `json:"video_convert"`
			ChatRender     string `json:"chat_render"`
			StreamlinkLive string `json:"streamlink_live"`
		}(conf.Parameters),
		Livestream: struct {
			Proxies         []config.ProxyListItem `json:"proxies"`
			ProxyEnabled    bool                   `json:"proxy_enabled"`
			ProxyParameters string                 `json:"proxy_parameters"`
			ProxyWhitelist  []string               `json:"proxy_whitelist"`
		}(conf.Livestream),
	}
	if err := h.Service.ConfigService.UpdateConfig(c, &cDto); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, conf)
}

// GetNotificationConfig godoc
//
//	@Summary		Get notification config
//	@Description	Get notification config
//	@Tags			config
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	config.Notification
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/config/notification [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetNotificationConfig(c echo.Context) error {
	conf, err := h.Service.ConfigService.GetNotificationConfig(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, conf)
}

// UpdateNotificationConfig godoc
//
//	@Summary		Update notification config
//	@Description	Update notification config
//	@Tags			config
//	@Accept			json
//	@Produce		json
//	@Param			body	body		UpdateNotificationRequest	true	"Config"
//	@Success		200		{object}	UpdateNotificationRequest
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/config/notification [put]
//	@Security		ApiKeyCookieAuth
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

// GetStorageTemplateConfig godoc
//
//	@Summary		Get storage template config
//	@Description	Get storage template config
//	@Tags			config
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	config.StorageTemplate
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/config/storage [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetStorageTemplateConfig(c echo.Context) error {
	conf, err := h.Service.ConfigService.GetStorageTemplateConfig(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, conf)
}

// UpdateStorageTemplateConfig godoc
//
//	@Summary		Update storage template config
//	@Description	Update storage template config
//	@Tags			config
//	@Accept			json
//	@Produce		json
//	@Param			body	body		UpdateStorageTemplateRequest	true	"Config"
//	@Success		200		{object}	UpdateStorageTemplateRequest
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/config/storage [put]
//	@Security		ApiKeyCookieAuth
func (h *Handler) UpdateStorageTemplateConfig(c echo.Context) error {
	conf := new(UpdateStorageTemplateRequest)
	if err := c.Bind(conf); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(conf); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if len(conf.FolderTemplate) == 0 || len(conf.FileTemplate) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Folder template and file template can't be empty")
	}

	// Check if folder template contains {{uuid}}

	if !strings.Contains(conf.FolderTemplate, "{{uuid}}") {
		return echo.NewHTTPError(http.StatusBadRequest, "Folder template must contain {{uuid}}")
	}

	cDto := config.StorageTemplate{
		FolderTemplate: conf.FolderTemplate,
		FileTemplate:   conf.FileTemplate,
	}

	if err := h.Service.ConfigService.UpdateStorageTemplateConfig(c, &cDto); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, conf)
}
