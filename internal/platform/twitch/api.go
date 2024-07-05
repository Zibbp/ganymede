package platform_twitch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/kv"
)

type AuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type Pagination struct {
	Cursor string `json:"cursor"`
}

type GetVideoResponse struct {
	Data       []TwitchVideoInfo `json:"data"`
	Pagination Pagination        `json:"pagination"`
}

var (
	TwitchApiUrl = "https://api.twitch.tv/helix"
)

func authenticate(clientId string, clientSecret string) (*AuthTokenResponse, error) {
	client := &http.Client{}

	req, err := http.NewRequest("POST", "https://id.twitch.tv/oauth2/token", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	q := url.Values{}
	q.Set("client_id", clientId)
	q.Set("client_secret", clientSecret)
	q.Set("grant_type", "client_credentials")
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate: %v", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to authenticate: %v", resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var authTokenResponse AuthTokenResponse
	err = json.Unmarshal(body, &authTokenResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &authTokenResponse, nil
}

func makeHTTPRequest(method, url string, queryParams map[string]string, headers map[string]string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, fmt.Sprintf("%s/%s", TwitchApiUrl, url), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	envConfig := config.GetEnvConfig()

	// Set auth headers
	req.Header.Set("Client-ID", envConfig.TwitchClientId)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", kv.DB().Get("TWITCH_ACCESS_TOKEN")))

	// Set query parameters
	q := req.URL.Query()
	for key, value := range queryParams {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}

	return body, nil
}
