package platform_twitch

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zibbp/ganymede/internal/chapter"
	"github.com/zibbp/ganymede/internal/kv"
	"github.com/zibbp/ganymede/internal/platform"
)

type TwitchPlatformService struct {
	ClientId     string
	ClientSecret string
	AccessToken  string
}

type PlatformTwitch struct{}

type TwitchGetVideosResponse struct {
	Data       []TwitchVideoInfo `json:"data"`
	Pagination Pagination        `json:"pagination"`
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

type TwitchLivestreams struct {
	Data       []TwitchLivestreamInfo `json:"data"`
	Pagination Pagination             `json:"pagination"`
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

type TwitchCategoryResponse struct {
	Data       []TwitchCategory `json:"data"`
	Pagination Pagination       `json:"pagination"`
}

type TwitchCategory struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	BoxArtURL string `json:"box_art_url"`
	IgdbID    string `json:"igdb_id"`
}

func NewTwitchPlatformService(clientId string, clientSercret string) (platform.PlatformService[TwitchVideoInfo, TwitchLivestreamInfo, TwitchChannel, TwitchCategory], error) {

	accessToken := kv.DB().Get("TWITCH_ACCESS_TOKEN")

	if accessToken == "" {
		tokenResponse, err := authenticate(clientId, clientSercret)
		if err != nil {
			return nil, err
		}
		accessToken = tokenResponse.AccessToken

		kv.DB().Set("TWITCH_ACCESS_TOKEN", accessToken)
	}

	return &TwitchPlatformService{
		ClientId:     clientId,
		ClientSecret: clientSercret,
		AccessToken:  accessToken,
	}, nil
}

func (tp *TwitchPlatformService) Authenticate(ctx context.Context) error {

	tokenResponse, err := authenticate(tp.ClientId, tp.ClientSecret)
	if err != nil {
		return err
	}
	tp.AccessToken = tokenResponse.AccessToken

	kv.DB().Set("TWITCH_ACCESS_TOKEN", tp.AccessToken)

	return nil
}

func (tp *TwitchPlatformService) GetVideoInfo(ctx context.Context, id string) (TwitchVideoInfo, error) {

	info, err := tp.GetVideoById(ctx, id)
	if err != nil {
		return TwitchVideoInfo{}, err
	}

	return info, nil
}

func (tp *TwitchPlatformService) GetVideoById(ctx context.Context, videoId string) (TwitchVideoInfo, error) {
	queryParams := map[string]string{"id": videoId}
	body, err := makeHTTPRequest("GET", "videos", queryParams, nil)
	if err != nil {
		return TwitchVideoInfo{}, err
	}

	var videoResponse GetVideoResponse
	err = json.Unmarshal(body, &videoResponse)
	if err != nil {
		return TwitchVideoInfo{}, err
	}

	if len(videoResponse.Data) == 0 {
		return TwitchVideoInfo{}, fmt.Errorf("video not found")
	}

	return videoResponse.Data[0], nil
}

func (tp *TwitchPlatformService) GetLivestreamInfo(ctx context.Context, channelName string) (TwitchLivestreamInfo, error) {
	queryParams := map[string]string{"user_login": channelName}
	body, err := makeHTTPRequest("GET", "streams", queryParams, nil)
	if err != nil {
		return TwitchLivestreamInfo{}, err
	}

	var resp TwitchLivestreams
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return TwitchLivestreamInfo{}, err
	}

	if len(resp.Data) == 0 {
		return TwitchLivestreamInfo{}, fmt.Errorf("no streams found")
	}

	return resp.Data[0], nil
}

func (tp *TwitchPlatformService) GetChannelByName(ctx context.Context, name string) (TwitchChannel, error) {
	queryParams := map[string]string{"login": name}
	body, err := makeHTTPRequest("GET", "users", queryParams, nil)
	if err != nil {
		return TwitchChannel{}, err
	}

	var resp TwitchChannelResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return TwitchChannel{}, err
	}

	if len(resp.Data) == 0 {
		return TwitchChannel{}, fmt.Errorf("channel not found")
	}

	return resp.Data[0], nil
}

func (tp *TwitchPlatformService) GetVideosByUser(ctx context.Context, userId string, videoType string) ([]TwitchVideoInfo, error) {
	queryParams := map[string]string{"user_id": userId, "first": "100", "type": videoType}
	body, err := makeHTTPRequest("GET", "videos", queryParams, nil)
	if err != nil {
		return nil, err
	}

	var resp TwitchGetVideosResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	var videos []TwitchVideoInfo
	videos = append(videos, resp.Data...)

	// pagination
	cursor := resp.Pagination.Cursor
	for cursor != "" {
		queryParams["after"] = cursor
		body, err = makeHTTPRequest("GET", "videos", queryParams, nil)
		if err != nil {
			return nil, err
		}
		var resp TwitchGetVideosResponse
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return nil, err
		}
		videos = append(videos, resp.Data...)
		cursor = resp.Pagination.Cursor
	}

	return videos, nil
}

func (tp *TwitchPlatformService) GetCategories(ctx context.Context) ([]TwitchCategory, error) {
	queryParams := map[string]string{}
	body, err := makeHTTPRequest("GET", "games/top", queryParams, nil)
	if err != nil {
		return nil, err
	}

	var resp TwitchCategoryResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	var categories []TwitchCategory
	categories = append(categories, resp.Data...)

	// pagination
	cursor := resp.Pagination.Cursor
	for cursor != "" {
		queryParams["after"] = cursor
		body, err = makeHTTPRequest("GET", "games/top", queryParams, nil)
		if err != nil {
			return nil, err
		}
		var resp TwitchCategoryResponse
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return nil, err
		}
		categories = append(categories, resp.Data...)
		cursor = resp.Pagination.Cursor
	}

	return categories, nil
}
