package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

type TwitchConnection struct {
	ClientId     string
	ClientSecret string
	AccessToken  string
}

func (c *TwitchConnection) Authenticate(ctx context.Context) (*ConnectionInfo, error) {
	info := ConnectionInfo{
		ClientId:     c.ClientId,
		ClientSecret: c.ClientSecret,
	}

	authResponse, err := twitchAuthenticate(c.ClientId, c.ClientSecret)
	if err != nil {
		return nil, err
	}
	info.AccessToken = authResponse.AccessToken
	c.AccessToken = authResponse.AccessToken
	duration := time.Duration(authResponse.ExpiresIn) * time.Second
	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24

	log.Info().Str("expires_in", fmt.Sprintf("%d days and %d hours", days, hours)).Msg("twitch connection authenticated")

	return &info, nil
}
