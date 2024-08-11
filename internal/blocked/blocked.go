package blocked

import (
	"context"

	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/blockedvideos"
	"github.com/zibbp/ganymede/internal/database"
)

type Service struct {
	Store *database.Database
}

func NewService(store *database.Database) *Service {
	return &Service{Store: store}
}

func (s *Service) IsVideoBlocked(ctx context.Context, id string) (bool, error) {
	return s.Store.Client.BlockedVideos.Query().Where(blockedvideos.ID(id)).Exist(ctx)
}

func (s *Service) CreateBlockedVideo(ctx context.Context, id string) error {
	_, err := s.Store.Client.BlockedVideos.Create().SetID(id).Save(ctx)
	return err
}

func (s *Service) DeleteBlockedVideo(ctx context.Context, id string) error {
	return s.Store.Client.BlockedVideos.DeleteOneID(id).Exec(ctx)
}

func (s *Service) GetBlockedVideos(ctx context.Context) ([]*ent.BlockedVideos, error) {
	videos, err := s.Store.Client.BlockedVideos.Query().Order(ent.Asc(blockedvideos.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}
	return videos, nil
}
