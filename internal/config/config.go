package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/zibbp/ganymede/internal/database"
	"os"
	"strings"
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
	Debug               bool   `json:"debug"`
	LiveCheckInterval   int    `json:"live_check_interval_seconds"`
	ActiveQueueItems    int    `json:"active_queue_items"`
	OAuthEnabled        bool   `json:"oauth_enabled"`
	RegistrationEnabled bool   `json:"registration_enabled"`
	WebhookURL          string `json:"webhook_url"`
	DBSeeded            bool   `json:"db_seeded"`
	Parameters          struct {
		VideoConvert   string `json:"video_convert"`
		ChatRender     string `json:"chat_render"`
		StreamlinkLive string `json:"streamlink_live"`
	} `json:"parameters"`
}

func NewConfig() {
	configLocation := "/data"
	configName := "config"
	configType := "json"
	configPath := fmt.Sprintf("%s/%s.%s", configLocation, configName, configType)

	viper.AddConfigPath(configLocation)
	viper.SetConfigName(configName)
	viper.SetConfigType(configType)

	viper.SetDefault("debug", false)
	viper.SetDefault("live_check_interval_seconds", 300)
	viper.SetDefault("active_queue_items", 2)
	viper.SetDefault("oauth_enabled", false)
	viper.SetDefault("registration_enabled", true)
	viper.SetDefault("webhook_url", "")
	viper.SetDefault("db_seeded", false)
	viper.SetDefault("parameters.video_convert", "-c:v copy -c:a copy")
	viper.SetDefault("parameters.chat_render", "-h 1440 -w 340 --framerate 30 --font Inter --font-size 13")
	viper.SetDefault("parameters.streamlink_live", "--force-progress,--force,--twitch-low-latency,--twitch-disable-hosting")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Info().Msgf("config file not found at %s, creating new one", configPath)
		err := viper.SafeWriteConfigAs(configPath)
		if err != nil {
			log.Panic().Err(err).Msg("error creating config file")
		}
	} else {
		log.Info().Msgf("config file found at %s, loading", configPath)
		err := viper.ReadInConfig()
		// Rewrite config file to apply new variables and remove old values
		refreshConfig(configPath)
		log.Debug().Msgf("config file loaded: %s", viper.ConfigFileUsed())
		if err != nil {
			log.Panic().Err(err).Msg("error reading config file")
		}
	}
}

func (s *Service) GetConfig(c echo.Context) (*Conf, error) {
	return &Conf{
		RegistrationEnabled: viper.GetBool("registration_enabled"),
		WebhookURL:          viper.GetString("webhook_url"),
		DBSeeded:            viper.GetBool("db_seeded"),
		Parameters: struct {
			VideoConvert   string `json:"video_convert"`
			ChatRender     string `json:"chat_render"`
			StreamlinkLive string `json:"streamlink_live"`
		}(struct {
			VideoConvert   string
			ChatRender     string
			StreamlinkLive string
		}{
			VideoConvert:   viper.GetString("parameters.video_convert"),
			ChatRender:     viper.GetString("parameters.chat_render"),
			StreamlinkLive: viper.GetString("parameters.streamlink_live"),
		}),
	}, nil
}

func (s *Service) UpdateConfig(c echo.Context, cDto *Conf) error {
	viper.Set("registration_enabled", cDto.RegistrationEnabled)
	viper.Set("webhook_url", cDto.WebhookURL)
	viper.Set("db_seeded", cDto.DBSeeded)
	viper.Set("parameters.video_convert", cDto.Parameters.VideoConvert)
	viper.Set("parameters.chat_render", cDto.Parameters.ChatRender)
	viper.Set("parameters.streamlink_live", cDto.Parameters.StreamlinkLive)
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
		viper.Set("parameters.streamlink_live", "--force-progress,--force,--twitch-low-latency,--twitch-disable-hosting")
	}
	err = viper.WriteConfigAs(configPath)
	if err != nil {
		log.Panic().Err(err).Msg("error writing config file")
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
