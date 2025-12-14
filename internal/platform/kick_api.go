package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zibbp/ganymede/internal/utils"
)

var (
	KickApiUrl          = "https://api.kick.com/public/v1"
	KickPrivateApiUrlv2 = "https://kick.com/api/v2"
	KickPrivateApiUrl   = "https://kick.com/api/v1"
	KickAuthUrl         = "https://id.kick.com"
)

type KickAPIResponse[T any] struct {
	Data []T `json:"data"`
}

type KickOldAPIResponse[T any] struct {
	Data T `json:"data"`
}

type KickChannel struct {
	BannerPicture     string `json:"banner_picture"`
	BroadcasterUserID int    `json:"broadcaster_user_id"`
	Category          struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Thumbnail string `json:"thumbnail"`
	} `json:"category"`
	ChannelDescription string `json:"channel_description"`
	Slug               string `json:"slug"`
	Stream             struct {
		IsLive      bool   `json:"is_live"`
		IsMature    bool   `json:"is_mature"`
		Key         string `json:"key"`
		Language    string `json:"language"`
		StartTime   string `json:"start_time"`
		Thumbnail   string `json:"thumbnail"`
		URL         string `json:"url"`
		ViewerCount int    `json:"viewer_count"`
	} `json:"stream"`
	StreamTitle string `json:"stream_title"`
}

type KickLiveStream struct {
	BroadcasterUserID int `json:"broadcaster_user_id"`
	Category          struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Thumbnail string `json:"thumbnail"`
	} `json:"category"`
	ChannelID        int    `json:"channel_id"`
	HasMatureContent bool   `json:"has_mature_content"`
	Language         string `json:"language"`
	Slug             string `json:"slug"`
	StartedAt        string `json:"started_at"`
	StreamTitle      string `json:"stream_title"`
	Thumbnail        string `json:"thumbnail"`
	ViewerCount      int    `json:"viewer_count"`
}

type KickUser struct {
	Email          string `json:"email"`
	Name           string `json:"name"`
	ProfilePicture string `json:"profile_picture"`
	UserID         int    `json:"user_id"`
}

type KickChatRoom struct {
	ID int `json:"id"`
	// Has other fields, but not used
}

type KickVodChatMesssageResponse struct {
	Messages []KickVodChatMessage `json:"messages"`
	Cursor   string               `json:"cursor"`
}

type KickVodChatMessage struct {
	ID       string `json:"id"`
	ChatID   int    `json:"chat_id"`
	UserID   int    `json:"user_id"`
	Content  string `json:"content"`
	Type     string `json:"type"`
	Metadata string `json:"metadata"`
	Sender   struct {
		ID       int    `json:"id"`
		Slug     string `json:"slug"`
		Username string `json:"username"`
		Identity struct {
			Color  string `json:"color"`
			Badges []struct {
				Type  string `json:"type"`
				Text  string `json:"text"`
				Count int    `json:"count"`
			} `json:"badges"`
		} `json:"identity"`
	} `json:"sender"`
	CreatedAt time.Time `json:"created_at"`
}

type KickVideo struct {
	ID                int       `json:"id"`
	LiveStreamID      int       `json:"live_stream_id"`
	Slug              any       `json:"slug"`
	Thumb             any       `json:"thumb"`
	S3                any       `json:"s3"`
	TradingPlatformID any       `json:"trading_platform_id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	UUID              string    `json:"uuid"`
	Views             int       `json:"views"`
	DeletedAt         any       `json:"deleted_at"`
	IsPruned          bool      `json:"is_pruned"`
	IsPrivate         bool      `json:"is_private"`
	Status            string    `json:"status"`
	Source            string    `json:"source"`
	Livestream        struct {
		ID            int       `json:"id"`
		Slug          string    `json:"slug"`
		ChannelID     int       `json:"channel_id"`
		CreatedAt     string    `json:"created_at"`
		SessionTitle  string    `json:"session_title"`
		IsLive        bool      `json:"is_live"`
		RiskLevelID   any       `json:"risk_level_id"`
		StartTime     time.Time `json:"start_time"`
		Source        any       `json:"source"`
		TwitchChannel any       `json:"twitch_channel"`
		Duration      int       `json:"duration"`
		Language      string    `json:"language"`
		IsMature      bool      `json:"is_mature"`
		ViewerCount   int       `json:"viewer_count"`
		Tags          []any     `json:"tags"`
		Thumbnail     string    `json:"thumbnail"`
		Channel       struct {
			ID                  int    `json:"id"`
			UserID              int    `json:"user_id"`
			Slug                string `json:"slug"`
			IsBanned            bool   `json:"is_banned"`
			PlaybackURL         string `json:"playback_url"`
			NameUpdatedAt       any    `json:"name_updated_at"`
			VodEnabled          bool   `json:"vod_enabled"`
			SubscriptionEnabled bool   `json:"subscription_enabled"`
			IsAffiliate         bool   `json:"is_affiliate"`
			FollowersCount      int    `json:"followersCount"`
			User                struct {
				Profilepic string `json:"profilepic"`
				Bio        string `json:"bio"`
				Twitter    string `json:"twitter"`
				Facebook   string `json:"facebook"`
				Instagram  string `json:"instagram"`
				Youtube    string `json:"youtube"`
				Discord    string `json:"discord"`
				Tiktok     string `json:"tiktok"`
				Username   string `json:"username"`
			} `json:"user"`
			CanHost  bool `json:"can_host"`
			Verified struct {
				ID        int       `json:"id"`
				ChannelID int       `json:"channel_id"`
				CreatedAt time.Time `json:"created_at"`
				UpdatedAt time.Time `json:"updated_at"`
			} `json:"verified"`
		} `json:"channel"`
		Categories []struct {
			ID          int      `json:"id"`
			CategoryID  int      `json:"category_id"`
			Name        string   `json:"name"`
			Slug        string   `json:"slug"`
			Tags        []string `json:"tags"`
			Description any      `json:"description"`
			DeletedAt   any      `json:"deleted_at"`
			IsMature    bool     `json:"is_mature"`
			IsPromoted  bool     `json:"is_promoted"`
			Viewers     int      `json:"viewers"`
			Category    struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
				Slug string `json:"slug"`
				Icon string `json:"icon"`
			} `json:"category"`
		} `json:"categories"`
	} `json:"livestream"`
}

// kickAuthenticate is used to authenticate with the Kick API using client credentials.
func kickAuthenticate(clientId, clientSecret string) (*AuthTokenResponse, error) {
	authUrl := KickAuthUrl + "/oauth/token"

	client := &http.Client{}
	req, err := http.NewRequest("POST", authUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	body := fmt.Sprintf("client_id=%s&client_secret=%s&grant_type=client_credentials", clientId, clientSecret)
	req.Body = io.NopCloser(strings.NewReader(body))
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to authenticate: %v", resp)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	var authTokenResponse AuthTokenResponse
	err = json.Unmarshal(bodyBytes, &authTokenResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	return &authTokenResponse, nil
}

func (c *KickConnection) kickMakeHTTPRequest(baseUrl string, method, url string, queryParams url.Values, headers map[string]string) ([]byte, error) {
	client := &http.Client{}

	for attempt := 0; attempt < maxRetryAttempts; attempt++ {
		req, err := http.NewRequest(method, fmt.Sprintf("%s/%s", baseUrl, url), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}

		// Set headers
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		// Set auth headers
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.AccessToken))
		req.Header.Set("User-Agent", utils.ChromeUserAgent)

		// Set query parameters
		req.URL.RawQuery = queryParams.Encode()

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			continue // Retry on rate limit
		} else if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %v", err)
		}

		return bodyBytes, nil
	}

	return nil, fmt.Errorf("max retry attempts reached")
}

// GetChatRoom retrieves the chat room information for a given channel name.
// This uses the private Kick API endpoint to get the chat room details.
func (c *KickConnection) GetChatRoom(ctx context.Context, channelName string) (*KickChatRoom, error) {
	body, err := c.kickMakeHTTPRequest(KickPrivateApiUrlv2, "GET", fmt.Sprintf("channels/%s/chatroom", channelName), nil, nil)
	if err != nil {
		return nil, err
	}

	var resp KickChatRoom
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &resp, nil
}
