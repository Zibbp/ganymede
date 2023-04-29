package channel

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/channel"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/twitch"
	"github.com/zibbp/ganymede/internal/utils"
)

type Service struct {
	Store *database.Database
}

func NewService(store *database.Database) *Service {
	return &Service{Store: store}
}

type Channel struct {
	ID            uuid.UUID `json:"id"`
	ExtID         string    `json:"ext_id"`
	Name          string    `json:"name"`
	DisplayName   string    `json:"display_name"`
	ImagePath     string    `json:"image_path"`
	Retention     bool      `json:"retention"`
	RetentionDays int64     `json:"retention_days"`
	UpdatedAt     time.Time `json:"updated_at"`
	CreatedAt     time.Time `json:"created_at"`
}

func (s *Service) CreateChannel(channelDto Channel) (*ent.Channel, error) {

	cha, err := s.Store.Client.Channel.Create().SetExtID(channelDto.ExtID).SetName(channelDto.Name).SetDisplayName(channelDto.DisplayName).SetImagePath(channelDto.ImagePath).Save(context.Background())
	if err != nil {
		if _, ok := err.(*ent.ConstraintError); ok {
			return nil, fmt.Errorf("channel already exists")
		}
		log.Debug().Err(err).Msg("error creating channel")
		return nil, fmt.Errorf("error creating channel: %v", err)
	}

	return cha, nil
}

func (s *Service) GetChannels() ([]*ent.Channel, error) {
	channels, err := s.Store.Client.Channel.Query().Order(ent.Desc(channel.FieldCreatedAt)).All(context.Background())
	if err != nil {
		log.Debug().Err(err).Msg("error getting channels")
		return nil, fmt.Errorf("error getting channels: %v", err)
	}

	return channels, nil
}

func (s *Service) GetChannel(channelID uuid.UUID) (*ent.Channel, error) {
	cha, err := s.Store.Client.Channel.Query().Where(channel.ID(channelID)).WithVods().Only(context.Background())
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

func (s *Service) GetChannelByName(cName string) (*ent.Channel, error) {
	cha, err := s.Store.Client.Channel.Query().Where(channel.Name(cName)).Only(context.Background())
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

func (s *Service) DeleteChannel(channelID uuid.UUID) error {
	err := s.Store.Client.Channel.DeleteOneID(channelID).Exec(context.Background())
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

func (s *Service) UpdateChannel(cId uuid.UUID, channelDto Channel) (*ent.Channel, error) {
	cha, err := s.Store.Client.Channel.UpdateOneID(cId).SetName(channelDto.Name).SetDisplayName(channelDto.DisplayName).SetImagePath(channelDto.ImagePath).SetRetention(channelDto.Retention).SetRetentionDays(channelDto.RetentionDays).Save(context.Background())
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

func (s *Service) CheckChannelExists(cName string) bool {
	_, err := s.Store.Client.Channel.Query().Where(channel.Name(cName)).Only(context.Background())
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

func (s *Service) CheckChannelExistsNoContext(cName string) bool {
	_, err := s.Store.Client.Channel.Query().Where(channel.Name(cName)).Only(context.Background())
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

func PopulateExternalChannelID() {
	channels, err := database.DB().Client.Channel.Query().All(context.Background())
	if err != nil {
		log.Debug().Err(err).Msg("error getting channels")
	}

	for _, c := range channels {
		if c.ExtID != "" {
			continue
		}
		twitchC, err := twitch.API.GetUserByLogin(c.Name)
		if err != nil {
			log.Error().Msg("error getting twitch channel")
			continue
		}
		_, err = database.DB().Client.Channel.UpdateOneID(c.ID).SetExtID(twitchC.ID).Save(context.Background())
		if err != nil {
			log.Error().Err(err).Msg("error updating channel")
			continue
		}
		log.Debug().Msgf("updated channel %s", c.Name)
	}
}

func (s *Service) UpdateChannelImage(c echo.Context, channelID uuid.UUID) error {
	channel, err := s.GetChannel(channelID)
	if err != nil {
		return fmt.Errorf("error getting channel: %v", err)
	}

	// Fetch channel from Twitch API
	tChannel, err := twitch.API.GetUserByLogin(channel.Name)
	if err != nil {
		return fmt.Errorf("error fetching twitch channel: %v", err)
	}

	// Download channel profile image
	err = utils.DownloadFile(tChannel.ProfileImageURL, tChannel.Login, "profile.png")
	if err != nil {
		return fmt.Errorf("error downloading channel profile image: %v", err)
	}

	return nil
}
