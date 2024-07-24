package blocked

import (
	"context"

	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/blockedvods"
	"github.com/zibbp/ganymede/internal/database"
)

type Service struct {
	Store *database.Database
}

func NewService(store *database.Database) *Service {
	return &Service{Store: store}
}

func (s *Service) IsVodBlocked(ctx context.Context, id string) (bool, error) {
	return s.Store.Client.BlockedVods.Query().Where(blockedvods.ID(id)).Exist(ctx)
}

func (s *Service) CreateBlockedVod(ctx context.Context, id string) error {
	_, err := s.Store.Client.BlockedVods.Create().SetID(id).Save(ctx)
	return err
}

func (s *Service) DeleteBlockedVod(ctx context.Context, id string) error {
	return s.Store.Client.BlockedVods.DeleteOneID(id).Exec(ctx)
}

func (s *Service) GetBlockedVods(ctx context.Context) ([]string, error) {
	vods, err := s.Store.Client.BlockedVods.Query().Order(ent.Asc(blockedvods.FieldID)).IDs(ctx)
	if err != nil {
		return nil, err
	}
	return vods, nil
}
