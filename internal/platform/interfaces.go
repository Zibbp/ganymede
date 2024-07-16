package platform

import (
	"context"

	"github.com/zibbp/ganymede/internal/chapter"
)

type VideoInfo struct {
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
	Chapters      []chapter.Chapter `json:"chapters"`
	MutedSegments []MutedSegment    `json:"muted_segments"`
}

type LiveStreamInfo struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	UserLogin    string `json:"user_login"`
	UserName     string `json:"user_name"`
	GameID       string `json:"game_id"`
	GameName     string `json:"game_name"`
	Type         string `json:"type"`
	Title        string `json:"title"`
	ViewerCount  int64  `json:"viewer_count"`
	StartedAt    string `json:"started_at"`
	Language     string `json:"language"`
	ThumbnailURL string `json:"thumbnail_url"`
}

type ChannelInfo struct {
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

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ConnectionInfo struct {
	ClientId     string
	ClientSecret string
	AccessToken  string
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

type Platform interface {
	Authenticate(ctx context.Context) (*ConnectionInfo, error)
	GetVideo(ctx context.Context, id string, withChapters bool, withMutedSegments bool) (*VideoInfo, error)
	GetLiveStream(ctx context.Context, channelName string) (*LiveStreamInfo, error)
	GetLiveStreams(ctx context.Context, channelNames []string) ([]LiveStreamInfo, error)
	GetChannel(ctx context.Context, channelName string) (*ChannelInfo, error)
	GetVideos(ctx context.Context, channelId string, videoType VideoType) ([]VideoInfo, error)
	GetCategories(ctx context.Context) ([]Category, error)
	GetGlobalBadges(ctx context.Context) ([]Badge, error)
	GetChannelBadges(ctx context.Context, channelId string) ([]Badge, error)
	GetGlobalEmotes(ctx context.Context) ([]Emote, error)
	GetChannelEmotes(ctx context.Context, channelId string) ([]Emote, error)
}
