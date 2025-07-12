// Package config provides application configuration loading, caching, and explicit updates.
package config

import (
	"encoding/json"
	"os"
	"sync"
)

// Config is the main application configuration saved to disk and used in memory.
type Config struct {
	LiveCheckInterval   int  `json:"live_check_interval_seconds"`  // How often in seconds watched channels are checked for live streams.
	VideoCheckInterval  int  `json:"video_check_interval_minutes"` // How often in minutes watched channels are checked for new videos.
	RegistrationEnabled bool `json:"registration_enabled"`         // Enable registration.
	Parameters          struct {
		TwitchToken    string `json:"twitch_token"`    // Twitch token for ad-free live streams or subscriber-only videos.
		VideoConvert   string `json:"video_convert"`   // FFmpeg arguments for video conversion.
		ChatRender     string `json:"chat_render"`     // TwitchDownloaderCLI arguments for chat rendering.
		StreamlinkLive string `json:"streamlink_live"` // Streamlink arguments for live streams.
	} `json:"parameters"`
	Archive struct {
		SaveAsHls                bool `json:"save_as_hls"`                // Save as HLS rather than MP4.
		GenerateSpriteThumbnails bool `json:"generate_sprite_thumbnails"` // Generate sprite thumbnails for scrubbing.
	} `json:"archive"`
	Notification     Notification    `json:"notifications"`     // Notification templates and settings.
	StorageTemplates StorageTemplate `json:"storage_templates"` // Storage folder/file templates.
	Livestream       struct {
		Proxies         []ProxyListItem `json:"proxies"`          // List of proxies for live stream download.
		ProxyEnabled    bool            `json:"proxy_enabled"`    // Enable proxy usage.
		ProxyParameters string          `json:"proxy_parameters"` // Query parameters for proxy URL.
		ProxyWhitelist  []string        `json:"proxy_whitelist"`  // Channels exempt from proxy.
	} `json:"livestream"`
	Experimental struct {
		BetterLiveStreamDetectionAndCleanup bool `json:"better_live_stream_detection_and_cleanup"` // [EXPERIMENTAL] Enable enhanced detection and cleanup.
	} `json:"experimental"`
}

// Notification defines webhook URLs and templates for various events.
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

// StorageTemplate defines folder and file naming patterns.
type StorageTemplate struct {
	FolderTemplate string `json:"folder_template"`
	FileTemplate   string `json:"file_template"`
}

// ProxyListItem defines a single proxy and optional header.
type ProxyListItem struct {
	URL    string `json:"url"`
	Header string `json:"header"`
}

var (
	instance     *Config      // in-memory singleton
	configFile   string       // path to JSON file
	onceInit     sync.Once    // ensures Init runs only once
	configMutex  sync.RWMutex // guards instance
	initialError error        // error encountered during Init
)

// Init loads the configuration from the given file path exactly once.
// If the file does not exist, it will be created with default values.
// If new fields are added, the file will be rewritten to include them.
func Init() (*Config, error) {
	env := GetEnvConfig()
	configFile = env.ConfigDir + "/config.json"
	onceInit.Do(func() {
		cfg := &Config{}
		cfg.setDefaults()

		// Attempt to read existing file
		data, err := os.ReadFile(configFile)
		if err == nil {
			// Unmarshal existing values over defaults
			if err = json.Unmarshal(data, cfg); err != nil {
				initialError = err
				return
			}
			// Rewrite to include any new defaults
			if err = saveConfigUnsafe(cfg); err != nil {
				initialError = err
				return
			}
		} else if os.IsNotExist(err) {
			// Create new file with defaults
			if err = saveConfigUnsafe(cfg); err != nil {
				initialError = err
				return
			}
		} else {
			initialError = err
			return
		}

		instance = cfg
	})
	return instance, initialError
}

// Get reads the latest configuration from disk each time it is called.
// Init must be called beforehand to ensure the config file path is set.
func Get() *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()

	data, err := os.ReadFile(configFile)
	if err != nil {
		return instance
	}

	cfg := &Config{}
	cfg.setDefaults()
	if err := json.Unmarshal(data, cfg); err != nil {
		return instance
	}

	return cfg
}

// UpdateConfig replaces the in-memory config and persists it to disk.
func UpdateConfig(newCfg *Config) error {
	configMutex.Lock()
	defer configMutex.Unlock()
	if err := saveConfigUnsafe(newCfg); err != nil {
		return err
	}
	instance = newCfg
	return nil
}

// saveConfigUnsafe writes the given config struct to disk in JSON format.
func saveConfigUnsafe(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}

// setDefaults initializes all fields of Config to their default values.
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

	// livestream proxies
	c.Livestream.Proxies = []ProxyListItem{
		{URL: "https://eu.luminous.dev", Header: ""},
		{URL: "https://api.ttv.lol", Header: "x-donate-to:https://ttv.lol/donate"},
	}
	c.Livestream.ProxyEnabled = false
	c.Livestream.ProxyParameters = "%3Fplayer%3Dtwitchweb%26type%3Dany%26allow_source%3Dtrue%26allow_audio_only%3Dtrue%26allow_spectre%3Dfalse%26fast_bread%3Dtrue"
	c.Livestream.ProxyWhitelist = []string{}

	// experimental features
	c.Experimental.BetterLiveStreamDetectionAndCleanup = false
}
