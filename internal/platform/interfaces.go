package platform

import (
	"context"
	"time"

	"github.com/zibbp/ganymede/internal/chapter"
)

type VideoInfo struct {
	ID                          string            `json:"id"`
	StreamID                    string            `json:"stream_id"`
	UserID                      string            `json:"user_id"`
	UserLogin                   string            `json:"user_login"`
	UserName                    string            `json:"user_name"`
	Title                       string            `json:"title"`
	Description                 string            `json:"description"`
	CreatedAt                   time.Time         `json:"created_at"`
	PublishedAt                 time.Time         `json:"published_at"`
	URL                         string            `json:"url"`
	ThumbnailURL                string            `json:"thumbnail_url"`
	Viewable                    string            `json:"viewable"`
	ViewCount                   int64             `json:"view_count"`
	Language                    string            `json:"language"`
	Type                        string            `json:"type"`
	Duration                    time.Duration     `json:"duration"`
	Category                    *string           `json:"category"`    // the default/main category of the video
	Restriction                 *string           `json:"restriction"` // video restriction
	Chapters                    []chapter.Chapter `json:"chapters"`
	MutedSegments               []MutedSegment    `json:"muted_segments"`
	SpriteThumbnailsManifestUrl *string           `json:"sprite_thumbnails_manifest_url"`
}

type VideoRestriction string

const (
	VideoRestrictionSubscriber VideoRestriction = "subscriber"
)

type LiveStreamInfo struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	UserLogin    string    `json:"user_login"`
	UserName     string    `json:"user_name"`
	GameID       string    `json:"game_id"`
	GameName     string    `json:"game_name"`
	Type         string    `json:"type"`
	Title        string    `json:"title"`
	ViewerCount  int64     `json:"viewer_count"`
	StartedAt    time.Time `json:"started_at"`
	Language     string    `json:"language"`
	ThumbnailURL string    `json:"thumbnail_url"`
}

type ChannelInfo struct {
	ID              string    `json:"id"`
	Login           string    `json:"login"`
	DisplayName     string    `json:"display_name"`
	Type            string    `json:"type"`
	BroadcasterType string    `json:"broadcaster_type"`
	Description     string    `json:"description"`
	ProfileImageURL string    `json:"profile_image_url"`
	OfflineImageURL string    `json:"offline_image_url"`
	ViewCount       int64     `json:"view_count"`
	CreatedAt       time.Time `json:"created_at"`
}

type ClipInfo struct {
	ID           string    `json:"id"`
	URL          string    `json:"url"`
	ChannelID    string    `json:"channel_id"`
	ChannelName  *string   `json:"channel_name"`
	CreatorID    *string   `json:"creator_id"`
	CreatorName  *string   `json:"creator_name"`
	VideoID      string    `json:"video_id"`
	GameID       *string   `json:"game_id"`
	Language     *string   `json:"language"`
	Title        string    `json:"title"`
	ViewCount    int       `json:"view_count"`
	CreatedAt    time.Time `json:"created_at"`
	ThumbnailURL string    `json:"thumbnail_url"`
	Duration     int       `json:"duration"`
	VodOffset    *int      `json:"vod_offset"`
}

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ConnectionInfo struct {
	ClientId     string
	ClientSecret string
	AccessToken  string
}

// ClipsFilter a filter used when fetching clips from the platform
type ClipsFilter struct {
	StartedAt time.Time // start date
	EndedAt   time.Time // end date
	Limit     int       // number of clips to return
}

type VideoType string

const (
	VideoTypeArchive   VideoType = "archive"
	VideoTypeHighlight VideoType = "highlight"
	VideoTypeUpload    VideoType = "upload"
)

type MutedSegment struct {
	Duration int `json:"duration"`
	Offset   int `json:"offset"`
}

const (
	maxRetryAttempts = 3
	retryDelay       = 5 * time.Second
)

type Platform interface {
	Authenticate(ctx context.Context) (*ConnectionInfo, error)
	GetVideo(ctx context.Context, id string, withChapters bool, withMutedSegments bool) (*VideoInfo, error)
	GetLiveStream(ctx context.Context, channelName string) (*LiveStreamInfo, error)
	GetLiveStreams(ctx context.Context, channelNames []string) ([]LiveStreamInfo, error)
	GetChannel(ctx context.Context, channelName string) (*ChannelInfo, error)
	GetVideos(ctx context.Context, channelId string, videoType VideoType, withChapters bool, withMutedSegments bool) ([]VideoInfo, error)
	GetCategories(ctx context.Context) ([]Category, error)
	GetGlobalBadges(ctx context.Context) ([]Badge, error)
	GetChannelBadges(ctx context.Context, channelId string) ([]Badge, error)
	GetGlobalEmotes(ctx context.Context) ([]Emote, error)
	GetChannelEmotes(ctx context.Context, channelId string) ([]Emote, error)
	GetChannelClips(ctx context.Context, channelId string, filter ClipsFilter) ([]ClipInfo, error)
	GetClip(ctx context.Context, id string) (*ClipInfo, error)
	CheckIfStreamIsLive(ctx context.Context, channelName string) (bool, error)
}
