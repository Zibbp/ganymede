package twitch

import (
	"context"
	"fmt"
	"net/http"
)

// CheckUserAccessToken checks if the access token is valid by sending a GET request to the Twitch API
func CheckUserAccessToken(ctx context.Context, accessToken string) error {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://id.twitch.tv/oauth2/validate", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("OAuth %s", accessToken))

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check access token: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to check access token: %v", resp)
	}

	return nil
}
