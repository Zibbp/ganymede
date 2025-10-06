package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

type KickConnection struct {
	ClientId     string
	ClientSecret string
	AccessToken  string
}

// Authenticate authenticates the Kick connection using client credentials.
func (c *KickConnection) Authenticate(ctx context.Context) (*ConnectionInfo, error) {
	info := ConnectionInfo{
		ClientID:     c.ClientId,
		ClientSecret: c.ClientSecret,
	}

	authResponse, err := kickAuthenticate(c.ClientId, c.ClientSecret)
	if err != nil {
		return nil, err
	}
	info.AccessToken = authResponse.AccessToken
	c.AccessToken = authResponse.AccessToken
	duration := time.Duration(authResponse.ExpiresIn) * time.Second
	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24

	log.Info().Str("expires_in", fmt.Sprintf("%d days and %d hours", days, hours)).Msg("kick connection authenticated")

	return &info, nil
}
