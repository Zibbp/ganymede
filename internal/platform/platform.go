package platform

import "context"

type PlatformService[V any, L any, C any, Category any] interface {
	Authenticate(ctx context.Context) error
	GetVideoInfo(ctx context.Context, id string) (V, error)
	GetLivestreamInfo(ctx context.Context, channelName string) (L, error)
	GetVideoById(ctx context.Context, videoId string) (V, error)
	GetChannelByName(ctx context.Context, name string) (C, error)
	GetVideosByUser(ctx context.Context, userId string, videoType string) ([]V, error)
	GetCategories(ctx context.Context) ([]Category, error)
}
