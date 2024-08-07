package admin

import (
	"context"
	"fmt"

	"github.com/zibbp/ganymede/internal/database"
)

type Service struct {
	Store *database.Database
}

func NewService(store *database.Database) *Service {
	return &Service{Store: store}
}

type GetStatsResp struct {
	VodCount     int `json:"vod_count"`
	ChannelCount int `json:"channel_count"`
}

func (s *Service) GetStats(ctx context.Context) (GetStatsResp, error) {

	vC, err := s.Store.Client.Vod.Query().Count(ctx)
	if err != nil {
		return GetStatsResp{}, fmt.Errorf("error getting vod count: %v", err)
	}
	cC, err := s.Store.Client.Channel.Query().Count(ctx)
	if err != nil {
		return GetStatsResp{}, fmt.Errorf("error getting channel count: %v", err)
	}

	return GetStatsResp{
		VodCount:     vC,
		ChannelCount: cC,
	}, nil
}
