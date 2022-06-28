package queue

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/database"
	"time"
)

type Service struct {
	Store *database.Database
}

func NewService(store *database.Database) *Service {
	return &Service{Store: store}
}

type Queue struct {
	ID              uuid.UUID `json:"id"`
	LiveArchive     bool      `json:"live_archive"`
	OnHold          bool      `json:"on_hold"`
	VideoProcessing bool      `json:"video_processing"`
	ChatProcessing  bool      `json:"chat_processing"`
	Processing      bool      `json:"processing"`
	UpdatedAt       time.Time `json:"updated_at"`
	CreatedAt       time.Time `json:"created_at"`
}

func (s *Service) CreateQueueItem(c echo.Context, queueDto Queue, vID uuid.UUID) (*ent.Queue, error) {
	q, err := s.Store.Client.Queue.Create().SetVodID(vID).Save(c.Request().Context())
	if err != nil {
		if _, ok := err.(*ent.ConstraintError); ok {
			return nil, fmt.Errorf("queue item exists for vod or vod does not exist")
		}
		log.Debug().Err(err).Msg("error creating queue")
		return nil, fmt.Errorf("error creating queue: %v", err)
	}
	return q, nil
}

func (s *Service) GetQueueItems(c echo.Context) ([]*ent.Queue, error) {
	q, err := s.Store.Client.Queue.Query().WithVod().All(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error getting queue tasks: %v", err)
	}
	return q, nil
}
