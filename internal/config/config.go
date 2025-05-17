package config

import (
	"encoding/json"
	"os"
	"reflect"
	"sync"
)

// Config is the main application configuration saved to disk.
type Config struct {
	LiveCheckInterval   int  `json:"live_check_interval_seconds"`  // How often in seconds watched channels are checked for live streams. Recommended to not be less than 30 seconds.
	VideoCheckInterval  int  `json:"video_check_interval_minutes"` // How often in minutes watched channels are checked for new videos.
	RegistrationEnabled bool `json:"registration_enabled"`         // Enable registration.
	Parameters          struct {
		TwitchToken    string `json:"twitch_token"`    // Twitch token for ad-free live streams or subscriber-only videos/
		VideoConvert   string `json:"video_convert"`   // Video convert FFmpeg arguments.
		ChatRender     string `json:"chat_render"`     // Chater render TwitchDownloaderCLI arguments.
		StreamlinkLive string `json:"streamlink_live"` // Streamlink live stream download arguments.
	} `json:"parameters"`
	Archive struct {
		SaveAsHls                bool `json:"save_as_hls"`                // Save as HLS rather than mp4.
		GenerateSpriteThumbnails bool `json:"generate_sprite_thumbnails"` // Generate sprite thumbnails (seen when scrubbing the video).
	} `json:"archive"`
	Notification     Notification    `json:"notifications"`     // Notification templates.
	StorageTemplates StorageTemplate `json:"storage_templates"` // Storage templates.
	Livestream       struct {
		Proxies         []ProxyListItem `json:"proxies"`          // List of proxies for download live streams.
		ProxyEnabled    bool            `json:"proxy_enabled"`    // Enable downloading live stream through proxy.
		ProxyParameters string          `json:"proxy_parameters"` // Proxy parameters.
		ProxyWhitelist  []string        `json:"proxy_whitelist"`  // Whitelist channels from proxy.
	} `json:"livestream"`
}

// UnmarshalJSON implements custom JSON unmarshaling for Config
func (c *Config) UnmarshalJSON(data []byte) error {
	type ConfigAlias Config

	c.setDefaults()

	// Create a map to check which fields actually exist in JSON
	var jsonMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		return err
	}

	alias := (*ConfigAlias)(c)

	if err := customUnmarshal(data, alias, jsonMap); err != nil {
		return err
	}

	return nil
}

// customUnmarshal handles the recursive unmarshaling of structs
func customUnmarshal(data []byte, v interface{}, existingFields map[string]json.RawMessage) error {
	// Regular unmarshal for fields that exist in JSON
	if err := json.Unmarshal(data, v); err != nil {
		return err
	}

	// Use reflection to check and preserve default values for missing fields
	val := reflect.ValueOf(v).Elem()
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			continue
		}

		// Handle nested structs recursively
		if field.Type.Kind() == reflect.Struct {
			if rawData, exists := existingFields[jsonTag]; exists {
				var nestedFields map[string]json.RawMessage
				if err := json.Unmarshal(rawData, &nestedFields); err != nil {
					continue
				}

				fieldVal := val.Field(i).Addr().Interface()
				if err := customUnmarshal(rawData, fieldVal, nestedFields); err != nil {
					return err
				}
			}
			continue
		}

		// For non-struct fields, check if they exist in JSON
		//nolint:all
		if _, exists := existingFields[jsonTag]; !exists {
			// Field doesn't exist in JSON, keep the default value
			continue
		}

	}

	return nil
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

// Init initializes the config, creating if necessary, and returning.
func Init() (*Config, error) {
	env := GetEnvConfig()
	configFile = env.ConfigDir + "/config.json"
	instance = &Config{}
	initErr = instance.loadConfig()

	return instance, initErr
}

// loadConfig loads the config from disk
func (c *Config) loadConfig() error {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
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

	return nil
}

// setDefaults sets default config values.
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

// UpdateConfig updates the config on disk.
func UpdateConfig(newConfig *Config) error {
	mutex.Lock()
	defer mutex.Unlock()

	*instance = *newConfig
	return saveConfigUnsafe()
}

// UpdateConfig updates the config on disk.
func SaveConfig() error {
	mutex.Lock()
	defer mutex.Unlock()
	return saveConfigUnsafe()
}

func saveConfigUnsafe() error {
	file, err := json.MarshalIndent(instance, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, file, 0644)
}

// Get returns the config
func Get() *Config {
	mutex.RLock()
	defer mutex.RUnlock()

	instance := &Config{}
	err := instance.loadConfig()
	if err != nil {
		return nil
	}

	return instance
}
