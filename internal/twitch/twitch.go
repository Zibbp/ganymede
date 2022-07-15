package twitch

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

type Service struct {
}

type AuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type ChannelResponse struct {
	Data []Channel `json:"data"`
}

type Channel struct {
	ID              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	Type            string `json:"type"`
	BroadcasterType string `json:"broadcaster_type"`
	Description     string `json:"description"`
	ProfileImageURL string `json:"profile_image_url"`
	OfflineImageURL string `json:"offline_image_url"`
	ViewCount       int64  `json:"view_count"`
	CreatedAt       string `json:"created_at"`
}

type VodResponse struct {
	Data       []Vod      `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type Vod struct {
	ID            string      `json:"id"`
	StreamID      string      `json:"stream_id"`
	UserID        string      `json:"user_id"`
	UserLogin     string      `json:"user_login"`
	UserName      string      `json:"user_name"`
	Title         string      `json:"title"`
	Description   string      `json:"description"`
	CreatedAt     string      `json:"created_at"`
	PublishedAt   string      `json:"published_at"`
	URL           string      `json:"url"`
	ThumbnailURL  string      `json:"thumbnail_url"`
	Viewable      string      `json:"viewable"`
	ViewCount     int64       `json:"view_count"`
	Language      string      `json:"language"`
	Type          string      `json:"type"`
	Duration      string      `json:"duration"`
	MutedSegments interface{} `json:"muted_segments"`
}

type Pagination struct {
}

type Stream struct {
	Data       []Live     `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type Live struct {
	ID           string   `json:"id"`
	UserID       string   `json:"user_id"`
	UserLogin    string   `json:"user_login"`
	UserName     string   `json:"user_name"`
	GameID       string   `json:"game_id"`
	GameName     string   `json:"game_name"`
	Type         string   `json:"type"`
	Title        string   `json:"title"`
	ViewerCount  int64    `json:"viewer_count"`
	StartedAt    string   `json:"started_at"`
	Language     string   `json:"language"`
	ThumbnailURL string   `json:"thumbnail_url"`
	TagIDS       []string `json:"tag_ids"`
	IsMature     bool     `json:"is_mature"`
}

func NewService() *Service {
	return &Service{}
}

func Authenticate() error {
	twitchClientID := os.Getenv("TWITCH_CLIENT_ID")
	twitchClientSecret := os.Getenv("TWITCH_CLIENT_SECRET")
	if twitchClientID == "" || twitchClientSecret == "" {
		return fmt.Errorf("twitch client id or secret not set")
	}
	log.Debug().Msg("authenticating with twitch")

	client := &http.Client{}

	req, err := http.NewRequest("POST", "https://id.twitch.tv/oauth2/token", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	q := url.Values{}
	q.Set("client_id", twitchClientID)
	q.Set("client_secret", twitchClientSecret)
	q.Set("grant_type", "client_credentials")
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to authenticate: %v", resp)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	var authTokenResponse AuthTokenResponse
	err = json.Unmarshal(body, &authTokenResponse)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// Set access token as env var
	err = os.Setenv("TWITCH_ACCESS_TOKEN", authTokenResponse.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to set env var: %v", err)
	}

	log.Info().Msg("authenticated with twitch")

	return nil
}

func (s *Service) GetUserByLogin(cName string) (Channel, error) {
	log.Debug().Msgf("getting user by login: %s", cName)
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.twitch.tv/helix/users?login=%s", cName), nil)
	if err != nil {
		return Channel{}, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Client-ID", os.Getenv("TWITCH_CLIENT_ID"))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("TWITCH_ACCESS_TOKEN")))

	resp, err := client.Do(req)
	if err != nil {
		return Channel{}, fmt.Errorf("failed to get user: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Channel{}, fmt.Errorf("failed to get user: %v", resp)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Channel{}, fmt.Errorf("failed to read response body: %v", err)
	}

	var channelResponse ChannelResponse
	err = json.Unmarshal(body, &channelResponse)
	if err != nil {
		return Channel{}, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// Check if channel is populated
	if len(channelResponse.Data) == 0 {
		return Channel{}, fmt.Errorf("channel not found")
	}

	return channelResponse.Data[0], nil
}

func (s *Service) GetVodByID(vID string) (Vod, error) {
	log.Debug().Msgf("getting twitch vod by id: %s", vID)
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.twitch.tv/helix/videos", nil)
	if err != nil {
		return Vod{}, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Client-ID", os.Getenv("TWITCH_CLIENT_ID"))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("TWITCH_ACCESS_TOKEN")))

	q := req.URL.Query()
	q.Add("id", vID)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return Vod{}, fmt.Errorf("failed to get vod: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Vod{}, fmt.Errorf("vod not found")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Vod{}, fmt.Errorf("failed to read response body: %v", err)
	}

	var vodResponse VodResponse
	err = json.Unmarshal(body, &vodResponse)
	if err != nil {
		return Vod{}, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// Check if vod is populated
	if len(vodResponse.Data) == 0 {
		return Vod{}, fmt.Errorf("vod not found")
	}

	return vodResponse.Data[0], nil
}

func (s *Service) GetStreams(queryParams string) (Stream, error) {
	log.Debug().Msgf("getting live streams using the following query param: %s", queryParams)
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.twitch.tv/helix/streams%s", queryParams), nil)
	if err != nil {
		return Stream{}, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Client-ID", os.Getenv("TWITCH_CLIENT_ID"))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("TWITCH_ACCESS_TOKEN")))

	resp, err := client.Do(req)
	if err != nil {
		return Stream{}, fmt.Errorf("failed to get twitch streams: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Stream{}, fmt.Errorf("failed to get twitch streams: %v", resp)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Stream{}, fmt.Errorf("failed to read response body: %v", err)
	}

	var streamResponse Stream
	err = json.Unmarshal(body, &streamResponse)
	if err != nil {
		return Stream{}, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return streamResponse, nil
}
