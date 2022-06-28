package channel

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/channel"
	"github.com/zibbp/ganymede/internal/database"
	"time"
)

type Service struct {
	Store *database.Database
}

func NewService(store *database.Database) *Service {
	return &Service{Store: store}
}

type Channel struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	ImagePath   string    `json:"image_path"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Service) CreateChannel(c echo.Context, channelDto Channel) (*ent.Channel, error) {

	cha, err := s.Store.Client.Channel.Create().SetName(channelDto.Name).SetDisplayName(channelDto.DisplayName).SetImagePath(channelDto.ImagePath).Save(c.Request().Context())
	if err != nil {
		if _, ok := err.(*ent.ConstraintError); ok {
			return nil, fmt.Errorf("channel already exists")
		}
		log.Debug().Err(err).Msg("error creating channel")
		return nil, fmt.Errorf("error creating channel: %v", err)
	}

	return cha, nil
}

func (s *Service) GetChannels(c echo.Context) ([]*ent.Channel, error) {
	channels, err := s.Store.Client.Channel.Query().All(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error getting channels")
		return nil, fmt.Errorf("error getting channels: %v", err)
	}

	return channels, nil
}

func (s *Service) GetChannel(c echo.Context, channelID uuid.UUID) (*ent.Channel, error) {
	cha, err := s.Store.Client.Channel.Query().Where(channel.ID(channelID)).Only(c.Request().Context())
	if err != nil {
		// if channel not found
		if _, ok := err.(*ent.NotFoundError); ok {
			return nil, fmt.Errorf("channel not found")
		}
		log.Debug().Err(err).Msg("error getting channel")
		return nil, fmt.Errorf("error getting channel: %v", err)
	}

	return cha, nil
}

func (s *Service) GetChannelByName(c echo.Context, cName string) (*ent.Channel, error) {
	cha, err := s.Store.Client.Channel.Query().Where(channel.Name(cName)).Only(c.Request().Context())
	if err != nil {
		// if channel not found
		if _, ok := err.(*ent.NotFoundError); ok {
			return nil, fmt.Errorf("channel not found")
		}
		log.Debug().Err(err).Msg("error getting channel")
		return nil, fmt.Errorf("error getting channel: %v", err)
	}

	return cha, nil
}

func (s *Service) DeleteChannel(c echo.Context, channelID uuid.UUID) error {
	err := s.Store.Client.Channel.DeleteOneID(channelID).Exec(c.Request().Context())
	if err != nil {
		// if channel not found
		if _, ok := err.(*ent.NotFoundError); ok {
			return fmt.Errorf("channel not found")
		}
		log.Debug().Err(err).Msg("error deleting channel")
		return fmt.Errorf("error deleting channel: %v", err)
	}

	return nil
}

func (s *Service) UpdateChannel(c echo.Context, cId uuid.UUID, channelDto Channel) (*ent.Channel, error) {
	cha, err := s.Store.Client.Channel.UpdateOneID(cId).SetName(channelDto.Name).SetDisplayName(channelDto.DisplayName).SetImagePath(channelDto.ImagePath).Save(c.Request().Context())
	if err != nil {
		// if channel not found
		if _, ok := err.(*ent.NotFoundError); ok {
			return nil, fmt.Errorf("channel not found")
		}
		log.Debug().Err(err).Msg("error updating channel")
		return nil, fmt.Errorf("error updating channel: %v", err)
	}

	return cha, nil
}

func (s *Service) CheckChannelExists(c echo.Context, cName string) bool {
	_, err := s.Store.Client.Channel.Query().Where(channel.Name(cName)).Only(c.Request().Context())
	if err != nil {
		// if channel not found
		if _, ok := err.(*ent.NotFoundError); ok {
			return false
		}
		log.Error().Err(err).Msg("error checking channel exists")
		return false
	}

	return true
}
