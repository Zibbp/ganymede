package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/config"
)

// type ConfigService interface {
// 	GetConfig(ctx context.Context) (*config.Conf, error)
// 	UpdateConfig(c echo.Context, conf *config.Conf) error
// 	GetNotificationConfig(c echo.Context) (*config.Notification, error)
// 	UpdateNotificationConfig(c echo.Context, conf *config.Notification) error
// 	GetStorageTemplateConfig(c echo.Context) (*config.StorageTemplate, error)
// 	UpdateStorageTemplateConfig(c echo.Context, conf *config.StorageTemplate) error
// }

// type UpdateConfigRequest struct {
// 	RegistrationEnabled bool `json:"registration_enabled"`
// 	Parameters          struct {
// 		TwitchToken    string `json:"twitch_token"`
// 		VideoConvert   string `json:"video_convert" validate:"required"`
// 		ChatRender     string `json:"chat_render" validate:"required"`
// 		StreamlinkLive string `json:"streamlink_live"`
// 	} `json:"parameters"`
// 	Archive struct {
// 		SaveAsHls bool `json:"save_as_hls"`
// 	} `json:"archive"`
// 	Livestream struct {
// 		Proxies         []config.ProxyListItem `json:"proxies"`
// 		ProxyEnabled    bool                   `json:"proxy_enabled"`
// 		ProxyParameters string                 `json:"proxy_parameters"`
// 		ProxyWhitelist  []string               `json:"proxy_whitelist"`
// 	} `json:"livestream"`
// }

// type UpdateNotificationRequest struct {
// 	VideoSuccessWebhookUrl string `json:"video_success_webhook_url"`
// 	VideoSuccessTemplate   string `json:"video_success_template"`
// 	VideoSuccessEnabled    bool   `json:"video_success_enabled"`
// 	LiveSuccessWebhookUrl  string `json:"live_success_webhook_url"`
// 	LiveSuccessTemplate    string `json:"live_success_template"`
// 	LiveSuccessEnabled     bool   `json:"live_success_enabled"`
// 	ErrorWebhookUrl        string `json:"error_webhook_url"`
// 	ErrorTemplate          string `json:"error_template"`
// 	ErrorEnabled           bool   `json:"error_enabled"`
// 	IsLiveWebhookUrl       string `json:"is_live_webhook_url"`
// 	IsLiveTemplate         string `json:"is_live_template"`
// 	IsLiveEnabled          bool   `json:"is_live_enabled"`
// }

// type UpdateStorageTemplateRequest struct {
// 	FolderTemplate string `json:"folder_template" validate:"required"`
// 	FileTemplate   string `json:"file_template" validate:"required"`
// }

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
	return c.JSON(http.StatusOK, config)
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
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(conf); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	err := config.UpdateConfig(conf)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, conf)
}
