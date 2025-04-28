package platform

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/chapter"
)

var (
	TwitchApiUrl = "https://api.twitch.tv/helix"
)

// authentication response
type AuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type TwitchGetVideosResponse struct {
	Data       []TwitchVideoInfo `json:"data"`
	Pagination TwitchPagination  `json:"pagination"`
}

type TwitchVideoInfo struct {
	ID            string            `json:"id"`
	StreamID      string            `json:"stream_id"`
	UserID        string            `json:"user_id"`
	UserLogin     string            `json:"user_login"`
	UserName      string            `json:"user_name"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	CreatedAt     string            `json:"created_at"`
	PublishedAt   string            `json:"published_at"`
	URL           string            `json:"url"`
	ThumbnailURL  string            `json:"thumbnail_url"`
	Viewable      string            `json:"viewable"`
	ViewCount     int64             `json:"view_count"`
	Language      string            `json:"language"`
	Type          string            `json:"type"`
	Duration      string            `json:"duration"`
	MutedSegments interface{}       `json:"muted_segments"`
	Chapters      []chapter.Chapter `json:"chapters"`
}

type TwitchLivestreamInfo struct {
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

type TwitchChannelResponse struct {
	Data []TwitchChannel `json:"data"`
}

type TwitchChannel struct {
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

type TwitchLiveStreamsRepsponse struct {
	Data       []TwitchLivestreamInfo `json:"data"`
	Pagination TwitchPagination       `json:"pagination"`
}

type TwitchCategoryResponse struct {
	Data       []TwitchCategory `json:"data"`
	Pagination TwitchPagination `json:"pagination"`
}

type TwitchCategory struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	BoxArtURL string `json:"box_art_url"`
	IgdbID    string `json:"igdb_id"`
}

type TwitchClip struct {
	ID              string  `json:"id"`
	URL             string  `json:"url"`
	EmbedURL        string  `json:"embed_url"`
	BroadcasterID   string  `json:"broadcaster_id"`
	BroadcasterName string  `json:"broadcaster_name"`
	CreatorID       string  `json:"creator_id"`
	CreatorName     string  `json:"creator_name"`
	VideoID         string  `json:"video_id"`
	GameID          string  `json:"game_id"`
	Language        string  `json:"language"`
	Title           string  `json:"title"`
	ViewCount       int     `json:"view_count"`
	CreatedAt       string  `json:"created_at"`
	ThumbnailURL    string  `json:"thumbnail_url"`
	Duration        float64 `json:"duration"`
	VodOffset       any     `json:"vod_offset"`
	IsFeatured      bool    `json:"is_featured"`
}

type TwitchGetClipsResponse struct {
	Data       []TwitchClip     `json:"data"`
	Pagination TwitchPagination `json:"pagination"`
}

type TwitchPagination struct {
	Cursor string `json:"cursor"`
}

type TwitchGlobalBadgeResponse struct {
	Data []struct {
		SetID    string `json:"set_id"`
		Versions []struct {
			ID          string `json:"id"`
			ImageURL1X  string `json:"image_url_1x"`
			ImageURL2X  string `json:"image_url_2x"`
			ImageURL4X  string `json:"image_url_4x"`
			Title       string `json:"title"`
			Description string `json:"description"`
			ClickAction string `json:"click_action"`
			ClickURL    string `json:"click_url"`
		} `json:"versions"`
	} `json:"data"`
}

type TwitchGlobalEmoteResponse struct {
	Data []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Images struct {
			URL1X string `json:"url_1x"`
			URL2X string `json:"url_2x"`
			URL4X string `json:"url_4x"`
		} `json:"images"`
		Format    []string `json:"format"`
		Scale     []string `json:"scale"`
		ThemeMode []string `json:"theme_mode"`
		EmoteType string   `json:"emote_type"`
	} `json:"data"`
	Template string `json:"template"`
}

type TwitchSpriteManifest []struct {
	Count    int      `json:"count"`
	Width    int      `json:"width"`
	Rows     int      `json:"rows"`
	Images   []string `json:"images"`
	Interval int      `json:"interval"`
	Quality  string   `json:"quality"`
	Cols     int      `json:"cols"`
	Height   int      `json:"height"`
}

// authenticate sends a POST request to Twitch for authentication using client credentials. An AuthenTokenResponse is returned on success containing the access token.
func twitchAuthenticate(clientId string, clientSecret string) (*AuthTokenResponse, error) {
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

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing response body")
		}
	}()
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

func (c *TwitchConnection) twitchMakeHTTPRequest(method, url string, queryParams url.Values, headers map[string]string) ([]byte, error) {
	client := &http.Client{}

	for attempt := 0; attempt < maxRetryAttempts; attempt++ {
		req, err := http.NewRequest(method, fmt.Sprintf("%s/%s", TwitchApiUrl, url), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}

		// Set headers
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		// Set auth headers
		req.Header.Set("Client-ID", c.ClientId)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))

		// Set query parameters
		req.URL.RawQuery = queryParams.Encode()

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to make request: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Debug().Err(err).Msg("error closing response body")
			}
		}()

		// Log rate limit usage if over threshold
		rateLimit := 0
		rateLimitRemaining := 0
		rateLimitReset := time.Time{}

		// Parse Ratelimit-Limit
		if rateLimitStr := resp.Header.Get("Ratelimit-Limit"); rateLimitStr != "" {
			value, err := strconv.Atoi(rateLimitStr)
			if err != nil {
				fmt.Printf("Error parsing Ratelimit-Limit: %v\n", err)
			} else {
				rateLimit = value
			}
		}

		// Parse Ratelimit-Remaining
		if rateLimitRemainingStr := resp.Header.Get("Ratelimit-Remaining"); rateLimitRemainingStr != "" {
			value, err := strconv.Atoi(rateLimitRemainingStr)
			if err != nil {
				fmt.Printf("Error parsing Ratelimit-Remaining: %v\n", err)
			} else {
				rateLimitRemaining = value
			}
		}

		// Parse Ratelimit-Reset
		if rateLimitResetStr := resp.Header.Get("Ratelimit-Reset"); rateLimitResetStr != "" {
			unixTime, err := strconv.ParseInt(rateLimitResetStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse Ratelimit-Reset: %v", err)
			}
			rateLimitReset = time.Unix(unixTime, 0)
		}

		// Check rate limit usage
		if rateLimit > 0 && rateLimitRemaining > 0 {
			usagePercentage := float64(rateLimit-rateLimitRemaining) / float64(rateLimit) * 100
			if usagePercentage > 75 {
				log.Warn().
					Int("rate_limit", rateLimit).
					Int("rate_limit_remaining", rateLimitRemaining).
					Str("rate_limit_expires", rateLimitReset.String()).
					Msgf("rate limit usage is over 75%% - %d/%d remaining", rateLimitRemaining, rateLimit)
			}
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %v", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			if attempt < maxRetryAttempts-1 {
				time.Sleep(retryDelay)
				continue
			}
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
		}

		return body, nil
	}

	return nil, fmt.Errorf("max retry attempts reached")
}
