package config

import (
	"encoding/json"
	"os"
	"sync"
)

type Config struct {
	LiveCheckInterval   int  `json:"live_check_interval_seconds"`
	VideoCheckInterval  int  `json:"video_check_interval_minutes"`
	RegistrationEnabled bool `json:"registration_enabled"`
	Parameters          struct {
		TwitchToken    string `json:"twitch_token"`
		VideoConvert   string `json:"video_convert"`
		ChatRender     string `json:"chat_render"`
		StreamlinkLive string `json:"streamlink_live"`
	} `json:"parameters"`
	Archive struct {
		GenerateSpriteThumbnails bool `json:"generate_sprite_thumbnails"`
		SaveAsHls                bool `json:"save_as_hls"`
	} `json:"archive"`
	Notification     Notification    `json:"notifications"`
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

var (
	instance *Config
	mutex    sync.RWMutex
	initErr  error
)

var configFile string

func (c *Config) loadConfig() error {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// If config file does not exist, set defaults and save the config
		c.setDefaults()
		return SaveConfig()
	}

	file, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(file, c)
	if err != nil {
		return err
	}

	// Merge the defaults
	mergeDefaults(&Config{}, c)

	return nil
}

// Get returns the configuration.
func Get() *Config {
	mutex.RLock()
	defer mutex.RUnlock()

	if instance == nil {
		instance = &Config{}
		err := instance.loadConfig()
		if err != nil {
			return nil
		}
	}

	return instance
}

// Init initializes and returns the configuration
func Init() (*Config, error) {
	env := GetEnvConfig()
	configFile = env.ConfigDir + "/config.json"
	instance = &Config{}
	initErr = instance.loadConfig()

	// Merge defaults
	mergeDefaults(&Config{}, instance)

	return instance, initErr
}

func mergeDefaults(defaults, current *Config) {
	if current.LiveCheckInterval == 0 {
		current.LiveCheckInterval = defaults.LiveCheckInterval
	}
	if current.VideoCheckInterval == 0 {
		current.VideoCheckInterval = defaults.VideoCheckInterval
	}
	if current.RegistrationEnabled == false {
		current.RegistrationEnabled = defaults.RegistrationEnabled
	}

	if current.Parameters.TwitchToken == "" {
		current.Parameters.TwitchToken = defaults.Parameters.TwitchToken
	}
	if current.Parameters.VideoConvert == "" {
		current.Parameters.VideoConvert = defaults.Parameters.VideoConvert
	}
	if current.Parameters.ChatRender == "" {
		current.Parameters.ChatRender = defaults.Parameters.ChatRender
	}
	if current.Parameters.StreamlinkLive == "" {
		current.Parameters.StreamlinkLive = defaults.Parameters.StreamlinkLive
	}

	if !current.Archive.GenerateSpriteThumbnails {
		current.Archive.GenerateSpriteThumbnails = defaults.Archive.GenerateSpriteThumbnails
	}
	if !current.Archive.SaveAsHls {
		current.Archive.SaveAsHls = defaults.Archive.SaveAsHls
	}

	if current.Notification.VideoSuccessWebhookUrl == "" {
		current.Notification.VideoSuccessWebhookUrl = defaults.Notification.VideoSuccessWebhookUrl
	}
	if current.Notification.VideoSuccessTemplate == "" {
		current.Notification.VideoSuccessTemplate = defaults.Notification.VideoSuccessTemplate
	}
	if !current.Notification.VideoSuccessEnabled {
		current.Notification.VideoSuccessEnabled = defaults.Notification.VideoSuccessEnabled
	}
	if current.Notification.LiveSuccessWebhookUrl == "" {
		current.Notification.LiveSuccessWebhookUrl = defaults.Notification.LiveSuccessWebhookUrl
	}
	if current.Notification.LiveSuccessTemplate == "" {
		current.Notification.LiveSuccessTemplate = defaults.Notification.LiveSuccessTemplate
	}
	if !current.Notification.LiveSuccessEnabled {
		current.Notification.LiveSuccessEnabled = defaults.Notification.LiveSuccessEnabled
	}
	if current.Notification.ErrorWebhookUrl == "" {
		current.Notification.ErrorWebhookUrl = defaults.Notification.ErrorWebhookUrl
	}
	if current.Notification.ErrorTemplate == "" {
		current.Notification.ErrorTemplate = defaults.Notification.ErrorTemplate
	}
	if !current.Notification.ErrorEnabled {
		current.Notification.ErrorEnabled = defaults.Notification.ErrorEnabled
	}
	if current.Notification.IsLiveWebhookUrl == "" {
		current.Notification.IsLiveWebhookUrl = defaults.Notification.IsLiveWebhookUrl
	}
	if current.Notification.IsLiveTemplate == "" {
		current.Notification.IsLiveTemplate = defaults.Notification.IsLiveTemplate
	}
	if !current.Notification.IsLiveEnabled {
		current.Notification.IsLiveEnabled = defaults.Notification.IsLiveEnabled
	}

	if current.StorageTemplates.FolderTemplate == "" {
		current.StorageTemplates.FolderTemplate = defaults.StorageTemplates.FolderTemplate
	}
	if current.StorageTemplates.FileTemplate == "" {
		current.StorageTemplates.FileTemplate = defaults.StorageTemplates.FileTemplate
	}

	if len(current.Livestream.Proxies) == 0 {
		current.Livestream.Proxies = defaults.Livestream.Proxies
	}
	if !current.Livestream.ProxyEnabled {
		current.Livestream.ProxyEnabled = defaults.Livestream.ProxyEnabled
	}
	if current.Livestream.ProxyParameters == "" {
		current.Livestream.ProxyParameters = defaults.Livestream.ProxyParameters
	}
}

func (c *Config) setDefaults() {
	c.LiveCheckInterval = 300
	c.VideoCheckInterval = 180
	c.RegistrationEnabled = true
	c.Parameters.TwitchToken = ""
	c.Parameters.VideoConvert = "-c:v copy -c:a copy"
	c.Parameters.ChatRender = "-h 1440 -w 340 --framerate 30 --font Inter --font-size 13"
	c.Parameters.StreamlinkLive = "--twitch-low-latency,--twitch-disable-hosting"
	c.Archive.SaveAsHls = false
	c.Archive.GenerateSpriteThumbnails = true

	// notifications
	c.Notification.VideoSuccessWebhookUrl = ""
	c.Notification.VideoSuccessTemplate = "✅ Video Archived: {{vod_title}} by {{channel_display_name}}."
	c.Notification.VideoSuccessEnabled = true
	c.Notification.LiveSuccessWebhookUrl = ""
	c.Notification.LiveSuccessTemplate = "✅ Live Stream Archived: {{vod_title}} by {{channel_display_name}}."
	c.Notification.LiveSuccessEnabled = true
	c.Notification.ErrorWebhookUrl = ""
	c.Notification.ErrorTemplate = "⚠️ Error: Queue {{queue_id}} failed at task {{failed_task}}."
	c.Notification.ErrorEnabled = true
	c.Notification.IsLiveWebhookUrl = ""
	c.Notification.IsLiveTemplate = "🔴 {{channel_display_name}} is live!"
	c.Notification.IsLiveEnabled = true

	// storage templates
	c.StorageTemplates.FolderTemplate = "{{date}}-{{id}}-{{type}}-{{uuid}}"
	c.StorageTemplates.FileTemplate = "{{id}}"

	// livestream
	c.Livestream.Proxies = []ProxyListItem{
		{
			URL:    "https://eu.luminous.dev",
			Header: "",
		},
		{
			URL:    "https://api.ttv.lol",
			Header: "x-donate-to:https://ttv.lol/donate",
		},
	}
	c.Livestream.ProxyEnabled = false
	c.Livestream.ProxyParameters = "%3Fplayer%3Dtwitchweb%26type%3Dany%26allow_source%3Dtrue%26allow_audio_only%3Dtrue%26allow_spectre%3Dfalse%26fast_bread%3Dtrue"
	c.Livestream.ProxyWhitelist = []string{}
}

func UpdateConfig(newConfig *Config) error {
	mutex.Lock()
	defer mutex.Unlock()

	// Make a deep copy of the new config
	*instance = *newConfig

	// Call SaveConfig without the mutex
	return saveConfigUnsafe()
}

// SaveConfig saves the current configuration to the JSON file
func SaveConfig() error {
	mutex.Lock()
	defer mutex.Unlock()
	return saveConfigUnsafe()
}

// saveConfigUnsafe saves the config without locking the mutex
func saveConfigUnsafe() error {
	file, err := json.MarshalIndent(instance, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, file, 0644)
}
