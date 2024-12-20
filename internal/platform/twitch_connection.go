package platform

import (
	"context"
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
	return &info, nil
}
