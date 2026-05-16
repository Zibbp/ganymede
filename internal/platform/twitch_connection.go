package platform

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type TwitchConnection struct {
	ClientId     string
	ClientSecret string
	AccessToken  string

	mu             sync.RWMutex
	tokenExpiresAt time.Time
}

func (c *TwitchConnection) Authenticate(ctx context.Context) (*ConnectionInfo, error) {
	info := ConnectionInfo{
		ClientId:     c.ClientId,
		ClientSecret: c.ClientSecret,
	}

	authResponse, err := twitchAuthenticate(ctx, c.ClientId, c.ClientSecret)
	if err != nil {
		return nil, err
	}

	info.AccessToken = authResponse.AccessToken
	c.setAuthToken(authResponse)
	duration := time.Duration(authResponse.ExpiresIn) * time.Second
	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24

	log.Info().Str("expires_in", fmt.Sprintf("%d days and %d hours", days, hours)).Msg("twitch connection authenticated")

	return &info, nil
}

func (c *TwitchConnection) currentAccessToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.AccessToken
}

func (c *TwitchConnection) setAuthToken(authResponse *AuthTokenResponse) {
	expiresAt := time.Now().Add(time.Duration(authResponse.ExpiresIn) * time.Second)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.AccessToken = authResponse.AccessToken
	c.tokenExpiresAt = expiresAt
}

func (c *TwitchConnection) tokenNeedsRefresh(now time.Time) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.accessTokenNeedsRefreshLocked(now)
}

func (c *TwitchConnection) accessTokenNeedsRefreshLocked(now time.Time) bool {
	return c.AccessToken == "" || c.tokenExpiresAt.IsZero() || !now.Add(twitchTokenRefreshBuffer).Before(c.tokenExpiresAt)
}

func (c *TwitchConnection) ensureValidAccessToken(ctx context.Context) error {
	if !c.tokenNeedsRefresh(time.Now()) {
		return nil
	}

	return c.refreshAccessToken(ctx)
}

func (c *TwitchConnection) refreshAccessToken(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.accessTokenNeedsRefreshLocked(time.Now()) {
		return nil
	}

	authResponse, err := twitchAuthenticate(ctx, c.ClientId, c.ClientSecret)
	if err != nil {
		return err
	}

	c.AccessToken = authResponse.AccessToken
	c.tokenExpiresAt = time.Now().Add(time.Duration(authResponse.ExpiresIn) * time.Second)

	duration := time.Duration(authResponse.ExpiresIn) * time.Second
	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24

	log.Info().Str("expires_in", fmt.Sprintf("%d days and %d hours", days, hours)).Msg("twitch connection authenticated")

	return nil
}

func (c *TwitchConnection) forceRefreshAccessToken(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	authResponse, err := twitchAuthenticate(ctx, c.ClientId, c.ClientSecret)
	if err != nil {
		return err
	}

	c.AccessToken = authResponse.AccessToken
	c.tokenExpiresAt = time.Now().Add(time.Duration(authResponse.ExpiresIn) * time.Second)

	duration := time.Duration(authResponse.ExpiresIn) * time.Second
	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24

	log.Info().Str("expires_in", fmt.Sprintf("%d days and %d hours", days, hours)).Msg("twitch connection authenticated")

	return nil
}
