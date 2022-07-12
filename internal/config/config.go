package config

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"os"
)

type Conf struct {
	RegistrationEnabled bool
	WebhookURL          string
	DBSeeded            bool
	Parameters          struct {
		VideoConvert string
		ChatRender   string
	}
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
	viper.SetDefault("parameters.video_convert", "'-c:v', 'copy', '-c:a', 'copy'")
	viper.SetDefault("parameters.chat_render", "'-h', '1440', '-w', '340', '--framerate', '30', '--font', 'Inter', '--font-size', '13'")

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
