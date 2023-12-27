package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/zibbp/ganymede/internal/database"
)

type Service struct {
	Store *database.Database
}

func NewService(store *database.Database) *Service {
	return &Service{
		Store: store,
	}
}

type Conf struct {
	Debug               bool `json:"debug"`
	LiveCheckInterval   int  `json:"live_check_interval_seconds"`
	VideoCheckInterval  int  `json:"video_check_interval_minutes"`
	OAuthEnabled        bool `json:"oauth_enabled"`
	RegistrationEnabled bool `json:"registration_enabled"`
	Parameters          struct {
		TwitchToken    string `json:"twitch_token"`
		VideoConvert   string `json:"video_convert"`
		ChatRender     string `json:"chat_render"`
		StreamlinkLive string `json:"streamlink_live"`
	} `json:"parameters"`
	Archive struct {
		SaveAsHls bool `json:"save_as_hls"`
	} `json:"archive"`
	Notifications    Notification    `json:"notifications"`
	StorageTemplates StorageTemplate `json:"storage_templates"`
	Livestream       struct {
		Proxies         []ProxyListItem `json:"proxies"`
		ProxyEnabled    bool            `json:"proxy_enabled"`
		ProxyParameters string          `json:"proxy_parameters"`
		ProxyWhitelist  []string        `json:"proxy_whitelist"`
	} `json:"livestream"`
}

type Notification struct {
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

type StorageTemplate struct {
	FolderTemplate string `json:"folder_template"`
	FileTemplate   string `json:"file_template"`
}

type ProxyListItem struct {
	URL    string `json:"url"`
	Header string `json:"header"`
}

func NewConfig(refresh bool) {
	configLocation := "/data"
	configName := "config"
	configType := "json"
	configPath := fmt.Sprintf("%s/%s.%s", configLocation, configName, configType)

	viper.AddConfigPath(configLocation)
	viper.SetConfigName(configName)
	viper.SetConfigType(configType)

	viper.SetDefault("debug", false)
	viper.SetDefault("live_check_interval_seconds", 300)
	viper.SetDefault("video_check_interval_minutes", 180)
	viper.SetDefault("oauth_enabled", false)
	viper.SetDefault("registration_enabled", true)
	viper.SetDefault("parameters.video_convert", "-c:v copy -c:a copy")
	viper.SetDefault("parameters.chat_render", "-h 1440 -w 340 --framerate 30 --font Inter --font-size 13")
	viper.SetDefault("parameters.streamlink_live", "--twitch-low-latency,--twitch-disable-hosting")
	viper.SetDefault("archive.save_as_hls", false)
	viper.SetDefault("parameters.twitch_token", "")
	// Notifications
	viper.SetDefault("notifications.video_success_webhook_url", "")
	viper.SetDefault("notifications.video_success_template", "✅ Video Archived: {{vod_title}} by {{channel_display_name}}.")
	viper.SetDefault("notifications.video_success_enabled", true)
	viper.SetDefault("notifications.live_success_webhook_url", "")
	viper.SetDefault("notifications.live_success_template", "✅ Live Stream Archived: {{vod_title}} by {{channel_display_name}}.")
	viper.SetDefault("notifications.live_success_enabled", true)
	viper.SetDefault("notifications.error_webhook_url", "")
	viper.SetDefault("notifications.error_template", "⚠️ Error: Queue ID {{queue_id}} for {{channel_display_name}} failed at task {{failed_task}}.")
	viper.SetDefault("notifications.error_enabled", true)
	viper.SetDefault("notifications.is_live_webhook_url", "")
	viper.SetDefault("notifications.is_live_template", "🔴 {{channel_display_name}} is live!")
	viper.SetDefault("notifications.is_live_enabled", true)

	// Storage Templates
	viper.SetDefault("storage_templates.folder_template", "{{date}}-{{id}}-{{type}}-{{uuid}}")
	viper.SetDefault("storage_templates.file_template", "{{id}}")

	// Livestream
	viper.SetDefault("livestream.proxies", []ProxyListItem{
		{
			URL:    "https://eu.luminous.dev",
			Header: "",
		},
		{
			URL:    "https://api.ttv.lol",
			Header: "x-donate-to:https://ttv.lol/donate",
		},
	})
	viper.SetDefault("livestream.proxy_enabled", false)
	viper.SetDefault("livestream.proxy_parameters", "%3Fplayer%3Dtwitchweb%26type%3Dany%26allow_source%3Dtrue%26allow_audio_only%3Dtrue%26allow_spectre%3Dfalse%26fast_bread%3Dtrue")
	viper.SetDefault("livestream.proxy_whitelist", []string{
		"",
	})

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Info().Msgf("config file not found at %s, creating new one", configPath)
		retries := 10
		for i := 0; i < retries; i++ {
			err := viper.SafeWriteConfigAs(configPath)
			if err == nil {
				log.Info().Msgf("config file created")
				break
			}
			log.Error().Err(err).Msgf("error creating config file (attempt %d/%d)", i+1, retries)
			if i < retries-1 {
				log.Info().Msgf("retrying in 1 second")
				time.Sleep(1 * time.Second)
			} else {
				log.Panic().Err(err).Msg("error creating config file")
			}
		}
	} else {
		log.Info().Msgf("config file found at %s, loading", configPath)
		retries := 10
		for i := 0; i < retries; i++ {
			err := viper.ReadInConfig()
			if err == nil {
				log.Info().Msgf("config file loaded: %s", viper.ConfigFileUsed())
				break
			}
			log.Error().Err(err).Msgf("error loading config (attempt %d/%d)", i+1, retries)
			if i < retries-1 {
				log.Info().Msgf("retrying in 1 second")
				time.Sleep(1 * time.Second)
			} else {
				log.Panic().Err(err).Msg("error loading config")
			}
		}
		// Rewrite config file to apply new variables and remove old values
		if refresh {
			refreshConfig(configPath)
		}
		log.Debug().Msgf("config file loaded: %s", viper.ConfigFileUsed())
	}
}

func (s *Service) GetConfig(c echo.Context) (*Conf, error) {
	proxies := viper.Get("livestream.proxies")
	var proxyListItems []ProxyListItem
	for _, proxy := range proxies.([]interface{}) {
		proxyListItem := ProxyListItem{
			URL:    proxy.(map[string]interface{})["url"].(string),
			Header: proxy.(map[string]interface{})["header"].(string),
		}
		proxyListItems = append(proxyListItems, proxyListItem)
	}

	return &Conf{
		RegistrationEnabled: viper.GetBool("registration_enabled"),
		Archive: struct {
			SaveAsHls bool `json:"save_as_hls"`
		}(struct {
			SaveAsHls bool
		}{
			SaveAsHls: viper.GetBool("archive.save_as_hls"),
		}),
		Parameters: struct {
			TwitchToken    string `json:"twitch_token"`
			VideoConvert   string `json:"video_convert"`
			ChatRender     string `json:"chat_render"`
			StreamlinkLive string `json:"streamlink_live"`
		}(struct {
			TwitchToken    string
			VideoConvert   string
			ChatRender     string
			StreamlinkLive string
		}{
			TwitchToken:    viper.GetString("parameters.twitch_token"),
			VideoConvert:   viper.GetString("parameters.video_convert"),
			ChatRender:     viper.GetString("parameters.chat_render"),
			StreamlinkLive: viper.GetString("parameters.streamlink_live"),
		}),
		StorageTemplates: struct {
			FolderTemplate string `json:"folder_template"`
			FileTemplate   string `json:"file_template"`
		}(struct {
			FolderTemplate string
			FileTemplate   string
		}{
			FolderTemplate: viper.GetString("storage_templates.folder_template"),
			FileTemplate:   viper.GetString("storage_templates.file_template"),
		}),
		Livestream: struct {
			Proxies         []ProxyListItem `json:"proxies"`
			ProxyEnabled    bool            `json:"proxy_enabled"`
			ProxyParameters string          `json:"proxy_parameters"`
			ProxyWhitelist  []string        `json:"proxy_whitelist"`
		}(struct {
			Proxies         []ProxyListItem
			ProxyEnabled    bool
			ProxyParameters string
			ProxyWhitelist  []string
		}{
			Proxies:         proxyListItems,
			ProxyEnabled:    viper.GetBool("livestream.proxy_enabled"),
			ProxyParameters: viper.GetString("livestream.proxy_parameters"),
			ProxyWhitelist:  viper.GetStringSlice("livestream.proxy_whitelist"),
		}),
	}, nil
}

func (s *Service) UpdateConfig(c echo.Context, cDto *Conf) error {
	viper.Set("registration_enabled", cDto.RegistrationEnabled)
	viper.Set("parameters.video_convert", cDto.Parameters.VideoConvert)
	viper.Set("parameters.chat_render", cDto.Parameters.ChatRender)
	viper.Set("parameters.streamlink_live", cDto.Parameters.StreamlinkLive)
	viper.Set("parameters.twitch_token", cDto.Parameters.TwitchToken)
	viper.Set("archive.save_as_hls", cDto.Archive.SaveAsHls)
	// proxies
	var proxyListItems []interface{}
	for _, proxy := range cDto.Livestream.Proxies {
		proxyListItem := map[string]interface{}{
			"url":    proxy.URL,
			"header": proxy.Header,
		}
		proxyListItems = append(proxyListItems, proxyListItem)
	}
	viper.Set("livestream.proxies", proxyListItems)
	viper.Set("livestream.proxy_enabled", cDto.Livestream.ProxyEnabled)
	viper.Set("livestream.proxy_whitelist", cDto.Livestream.ProxyWhitelist)

	err := viper.WriteConfig()
	if err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}
	return nil
}

func (s *Service) GetNotificationConfig(c echo.Context) (*Notification, error) {
	return &Notification{
		VideoSuccessWebhookUrl: viper.GetString("notifications.video_success_webhook_url"),
		VideoSuccessTemplate:   viper.GetString("notifications.video_success_template"),
		VideoSuccessEnabled:    viper.GetBool("notifications.video_success_enabled"),
		LiveSuccessWebhookUrl:  viper.GetString("notifications.live_success_webhook_url"),
		LiveSuccessTemplate:    viper.GetString("notifications.live_success_template"),
		LiveSuccessEnabled:     viper.GetBool("notifications.live_success_enabled"),
		ErrorWebhookUrl:        viper.GetString("notifications.error_webhook_url"),
		ErrorTemplate:          viper.GetString("notifications.error_template"),
		ErrorEnabled:           viper.GetBool("notifications.error_enabled"),
		IsLiveWebhookUrl:       viper.GetString("notifications.is_live_webhook_url"),
		IsLiveTemplate:         viper.GetString("notifications.is_live_template"),
		IsLiveEnabled:          viper.GetBool("notifications.is_live_enabled"),
	}, nil
}

func (s *Service) GetStorageTemplateConfig(c echo.Context) (*StorageTemplate, error) {
	return &StorageTemplate{
		FolderTemplate: viper.GetString("storage_templates.folder_template"),
		FileTemplate:   viper.GetString("storage_templates.file_template"),
	}, nil
}

func (s *Service) UpdateNotificationConfig(c echo.Context, nDto *Notification) error {
	viper.Set("notifications.video_success_webhook_url", nDto.VideoSuccessWebhookUrl)
	viper.Set("notifications.video_success_template", nDto.VideoSuccessTemplate)
	viper.Set("notifications.video_success_enabled", nDto.VideoSuccessEnabled)
	viper.Set("notifications.live_success_webhook_url", nDto.LiveSuccessWebhookUrl)
	viper.Set("notifications.live_success_template", nDto.LiveSuccessTemplate)
	viper.Set("notifications.live_success_enabled", nDto.LiveSuccessEnabled)
	viper.Set("notifications.error_webhook_url", nDto.ErrorWebhookUrl)
	viper.Set("notifications.error_template", nDto.ErrorTemplate)
	viper.Set("notifications.error_enabled", nDto.ErrorEnabled)
	viper.Set("notifications.is_live_webhook_url", nDto.IsLiveWebhookUrl)
	viper.Set("notifications.is_live_template", nDto.IsLiveTemplate)
	viper.Set("notifications.is_live_enabled", nDto.IsLiveEnabled)
	err := viper.WriteConfig()
	if err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}
	return nil
}

func (s *Service) UpdateStorageTemplateConfig(c echo.Context, stDto *StorageTemplate) error {
	viper.Set("storage_templates.folder_template", stDto.FolderTemplate)
	viper.Set("storage_templates.file_template", stDto.FileTemplate)
	err := viper.WriteConfig()
	if err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}
	return nil
}

// refreshConfig: rewrites config file applying variable changes and removing old ones
func refreshConfig(configPath string) {
	err := unset("live_check_interval")
	if err != nil {
		log.Error().Err(err).Msg("error unsetting config value")
	}
	// Add authentication method
	if !viper.IsSet("oauth_enabled") {
		viper.Set("oauth_enabled", false)
	}
	// streamlink params
	if !viper.IsSet("parameters.streamlink_live") {
		viper.Set("parameters.streamlink_live", "--twitch-low-latency,--twitch-disable-hosting")
	}
	err = viper.WriteConfigAs(configPath)
	if err != nil {
		log.Panic().Err(err).Msg("error writing config file")
	}
	if viper.IsSet("webhook_url") && viper.GetString("webhook_url") != "" {
		oldWebhookUrl := viper.GetString("webhook_url")
		viper.Set("notifications.video_success_webhook_url", oldWebhookUrl)
		viper.Set("notifications.live_success_webhook_url", oldWebhookUrl)
		viper.Set("notifications.error_webhook_url", oldWebhookUrl)
		viper.Set("notifications.is_live_webhook_url", oldWebhookUrl)
		err = viper.WriteConfigAs(configPath)
		if err != nil {
			log.Panic().Err(err).Msg("error writing config file")
		}
		err = unset("webhook_url")
		if err != nil {
			log.Error().Err(err).Msg("error unsetting config value")
		}
	} else {
		err = unset("webhook_url")
		if err != nil {
			log.Error().Err(err).Msg("error unsetting config value")
		}
	}
	// Archive
	if !viper.IsSet("archive.save_as_hls") {
		viper.Set("archive.save_as_hls", false)
	}
	// Storage template
	if !viper.IsSet("storage_templates.folder_template") {
		viper.Set("storage_templates.folder_template", "{{date}}-{{id}}-{{type}}-{{uuid}}")
	}
	if !viper.IsSet("storage_templates.file_template") {
		viper.Set("storage_templates.file_template", "{{id}}")
	}
	// Twitch Token
	if !viper.IsSet("parameters.twitch_token") {
		viper.Set("parameters.twitch_token", "")
	}
	// Livestream
	if !viper.IsSet("livestream.proxies") {
		viper.Set("livestream.proxies", []ProxyListItem{
			{
				URL:    "https://eu.luminous.dev",
				Header: "",
			},
			{
				URL:    "https://api.ttv.lol",
				Header: "x-donate-to:https://ttv.lol/donate",
			},
		})
	}
	if !viper.IsSet("livestream.proxy_enabled") {
		viper.Set("livestream.proxy_enabled", false)
	}
	if !viper.IsSet("livestream.proxy_parameters") {
		viper.Set("livestream.proxy_parameters", "%3Fplayer%3Dtwitchweb%26type%3Dany%26allow_source%3Dtrue%26allow_audio_only%3Dtrue%26allow_spectre%3Dfalse%26fast_bread%3Dtrue")
	}
	if !viper.IsSet("livestream.proxy_whitelist") {
		viper.Set("livestream.proxy_whitelist", []string{
			"",
		})
	}
	if !viper.IsSet("video_check_interval_minutes") {
		viper.Set("video_check_interval_minutes", 180)
	}
	err = unset("db_seeded")
	if err != nil {
		log.Error().Err(err).Msg("error unsetting config value")
	}
	err = unset("active_queue_items")
	if err != nil {
		log.Error().Err(err).Msg("error unsetting config value")
	}

}

// unset: removes variable from config file
// https://github.com/spf13/viper/issues/632#issuecomment-869668629
func unset(vars ...string) error {
	cfg := viper.AllSettings()
	vals := cfg

	for _, v := range vars {
		parts := strings.Split(v, ".")
		for i, k := range parts {
			v, ok := vals[k]
			if !ok {
				// Doesn't exist no action needed
				break
			}

			switch len(parts) {
			case i + 1:
				// Last part so delete.
				delete(vals, k)
			default:
				m, ok := v.(map[string]interface{})
				if !ok {
					return fmt.Errorf("unsupported type: %T for %q", v, strings.Join(parts[0:i], "."))
				}
				vals = m
			}
		}
	}

	b, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		return err
	}

	if err = viper.ReadConfig(bytes.NewReader(b)); err != nil {
		return err
	}

	return viper.WriteConfig()
}
