package category

import (
	"context"
	"fmt"

	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/database"
)

type Service struct {
	Store *database.Database
}

func NewService(store *database.Database) *Service {
	return &Service{Store: store}
}

func (s *Service) GetCategories(ctx context.Context) ([]*ent.TwitchCategory, error) {
	categories, err := database.DB().Client.TwitchCategory.Query().All(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %v", err)
	}

	return categories, nil
}
