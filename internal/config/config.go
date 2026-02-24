// Package config provides application configuration loading, caching, and explicit updates.
package config

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/zibbp/ganymede/internal/utils"
)

// Config is the main application configuration saved to disk and used in memory.
type Config struct {
	LiveCheckInterval   int  `json:"live_check_interval_seconds"`  // How often in seconds watched channels are checked for live streams.
	VideoCheckInterval  int  `json:"video_check_interval_minutes"` // How often in minutes watched channels are checked for new videos.
	RegistrationEnabled bool `json:"registration_enabled"`         // Enable registration.
	Parameters          struct {
		TwitchToken  string `json:"twitch_token"`  // Twitch token for ad-free live streams or subscriber-only videos.
		VideoConvert string `json:"video_convert"` // FFmpeg arguments for video conversion.
		ChatRender   string `json:"chat_render"`   // TwitchDownloaderCLI arguments for chat rendering.
		YtDlpVideo   string `json:"yt_dlp_video"`  // yt-dlp arguments for video downloads.
	} `json:"parameters"`
	Archive struct {
		SaveAsHls                bool `json:"save_as_hls"`                // Save as HLS rather than MP4.
		GenerateSpriteThumbnails bool `json:"generate_sprite_thumbnails"` // Generate sprite thumbnails for scrubbing.
	} `json:"archive"`
	StorageTemplates StorageTemplate `json:"storage_templates"` // Storage folder/file templates.
	Livestream       struct {
		Proxies             []ProxyListItem `json:"proxies" validate:"dive"` // List of proxies for live stream download.
		ProxyEnabled        bool            `json:"proxy_enabled"`           // Enable proxy usage.
		ProxyParameters     string          `json:"proxy_parameters"`        // Query parameters for proxy URL.
		ProxyWhitelist      []string        `json:"proxy_whitelist"`         // Channels exempt from proxy.
		WatchWhileArchiving bool            `json:"watch_while_archiving"`   // Allow watching live streams while archiving them by downloading a temporary HLS stream.
	} `json:"livestream"`
	Experimental struct {
		BetterLiveStreamDetectionAndCleanup bool `json:"better_live_stream_detection_and_cleanup"` // [EXPERIMENTAL] Enable enhanced detection and cleanup.
	} `json:"experimental"`
}

// LegacyNotification is the old config.json notification format used before
// the database-backed notification system. It is only used for migration.
type LegacyNotification struct {
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

// LegacyConfig is a minimal struct for reading the old config.json notifications field during migration.
type LegacyConfig struct {
	Notification LegacyNotification `json:"notifications"`
}

// ReadLegacyNotifications reads the old config.json and returns any legacy notification settings.
// Returns nil if the config file doesn't exist or has no configured webhook URLs.
func ReadLegacyNotifications() *LegacyNotification {
	env := GetEnvConfig()
	path := env.ConfigDir + "/config.json"

	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var legacy LegacyConfig
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil
	}

	n := &legacy.Notification
	// Only return if at least one webhook URL was configured
	if n.VideoSuccessWebhookUrl == "" && n.LiveSuccessWebhookUrl == "" && n.ErrorWebhookUrl == "" && n.IsLiveWebhookUrl == "" {
		return nil
	}

	return n
}

// StorageTemplate defines folder and file naming patterns.
type StorageTemplate struct {
	FolderTemplate string `json:"folder_template"`
	FileTemplate   string `json:"file_template"`
}

// ProxyListItem defines a single proxy and optional header.
type ProxyListItem struct {
	URL       string          `json:"url" validate:"required,min=1"`        // URL of the proxy server.
	Header    string          `json:"header"`                               // Optional header to send with the proxy request.
	ProxyType utils.ProxyType `json:"proxy_type" validate:"required,min=1"` // Type of proxy to use.
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
		cfg.SetDefaults()

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
	cfg.SetDefaults()
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

// SetDefaults initializes all fields of Config to their default values.
func (c *Config) SetDefaults() {
	c.LiveCheckInterval = 300
	c.VideoCheckInterval = 180
	c.RegistrationEnabled = true
	c.Parameters.TwitchToken = ""
	c.Parameters.VideoConvert = "-c:v copy -c:a copy"
	c.Parameters.ChatRender = "-h 1440 -w 340 --framerate 30 --font Inter --font-size 13"
	c.Parameters.YtDlpVideo = ""

	c.Archive.SaveAsHls = false
	c.Archive.GenerateSpriteThumbnails = true

	// storage templates
	c.StorageTemplates.FolderTemplate = "{{date}}-{{id}}-{{type}}-{{uuid}}"
	c.StorageTemplates.FileTemplate = "{{id}}"

	// livestream proxies
	c.Livestream.Proxies = []ProxyListItem{
		{URL: "https://eu.luminous.dev", Header: "", ProxyType: utils.ProxyTypeTwitchHLS},
	}
	c.Livestream.ProxyEnabled = false
	c.Livestream.ProxyParameters = "%3Fplayer%3Dtwitchweb%26type%3Dany%26allow_source%3Dtrue%26allow_audio_only%3Dtrue%26allow_spectre%3Dfalse%26fast_bread%3Dtrue"
	c.Livestream.ProxyWhitelist = []string{}
	c.Livestream.WatchWhileArchiving = false

	// experimental features
	c.Experimental.BetterLiveStreamDetectionAndCleanup = false
}
