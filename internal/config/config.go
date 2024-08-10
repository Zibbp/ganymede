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
		SaveAsHls bool `json:"save_as_hls"`
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
	once     sync.Once
	mutex    sync.RWMutex
	initErr  error
)

var configFile string

func init() {
	env := GetEnvConfig()
	configDir := os.Getenv("CONFIG_DIR")
	if configDir == "" {
		configDir = env.ConfigDir
	}
	configFile = configDir + "/config.json"
}

// Init initializes and returns the configuration
func Init() (*Config, error) {
	once.Do(func() {
		instance = &Config{}
		initErr = instance.loadConfig()
	})
	return instance, initErr
}

// LoadConfig loads the configuration from the JSON file or creates a default one
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

func (c *Config) setDefaults() {
	c.LiveCheckInterval = 300
	c.VideoCheckInterval = 180
	c.RegistrationEnabled = true
	c.Parameters.TwitchToken = ""
	c.Parameters.VideoConvert = "-c:v copy -c:a copy"
	c.Parameters.ChatRender = "-h 1440 -w 340 --framerate 30 --font Inter --font-size 13"
	c.Parameters.StreamlinkLive = "--twitch-low-latency,--twitch-disable-hosting"
	c.Archive.SaveAsHls = false

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

// Get returns the configuration
func Get() *Config {
	return instance
}
