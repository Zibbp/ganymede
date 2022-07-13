package config

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/zibbp/ganymede/internal/database"
	"os"
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
	RegistrationEnabled bool   `json:"registration_enabled"`
	WebhookURL          string `json:"webhook_url"`
	DBSeeded            bool   `json:"db_seeded"`
	Parameters          struct {
		VideoConvert string `json:"video_convert"`
		ChatRender   string `json:"chat_render"`
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

	viper.SetDefault("registration_enabled", true)
	viper.SetDefault("webhook_url", "")
	viper.SetDefault("db_seeded", false)
	viper.SetDefault("parameters.video_convert", "-c:v copy -c:a copy")
	viper.SetDefault("parameters.chat_render", "-h 1440 -w 340 --framerate 30 --font Inter --font-size 13")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Info().Msgf("config file not found at %s, creating new one", configPath)
		err := viper.SafeWriteConfigAs(configPath)
		if err != nil {
			log.Panic().Err(err).Msg("error creating config file")
		}
	} else {
		err := viper.ReadInConfig()
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
			VideoConvert string `json:"video_convert"`
			ChatRender   string `json:"chat_render"`
		}(struct {
			VideoConvert string
			ChatRender   string
		}{
			VideoConvert: viper.GetString("parameters.video_convert"),
			ChatRender:   viper.GetString("parameters.chat_render"),
		}),
	}, nil
}

func (s *Service) UpdateConfig(c echo.Context, cDto *Conf) error {
	viper.Set("registration_enabled", cDto.RegistrationEnabled)
	viper.Set("webhook_url", cDto.WebhookURL)
	viper.Set("db_seeded", cDto.DBSeeded)
	viper.Set("parameters.video_convert", cDto.Parameters.VideoConvert)
	viper.Set("parameters.chat_render", cDto.Parameters.ChatRender)
	err := viper.WriteConfig()
	if err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}
	return nil
}
