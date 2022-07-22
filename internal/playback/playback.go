package playback

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/playback"
	"github.com/zibbp/ganymede/internal/auth"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/utils"
)

type Service struct {
	Store *database.Database
}

func NewService(store *database.Database) *Service {
	return &Service{Store: store}
}

func (s *Service) UpdateProgress(c *auth.CustomContext, vID uuid.UUID, time int) error {
	uID := c.UserClaims.UserID

	check, err := s.Store.Client.Playback.Query().Where(playback.UserID(uID)).Where(playback.VodID(vID)).Only(c.Request().Context())
	if err != nil {
		if _, ok := err.(*ent.NotFoundError); ok {

			_, err = s.Store.Client.Playback.Create().SetUserID(uID).SetVodID(vID).SetTime(time).Save(c.Request().Context())
			if err != nil {
				return fmt.Errorf("error creating playback: %v", err)
			}

			return nil
		}
		return fmt.Errorf("error checking playback: %v", err)
	}
	if check != nil {
		_, err = s.Store.Client.Playback.Update().Where(playback.UserID(uID)).Where(playback.VodID(vID)).SetTime(time).Save(c.Request().Context())
		if err != nil {
			return fmt.Errorf("error updating playback: %v", err)
		}
	}

	return nil
}

func (s *Service) GetProgress(c *auth.CustomContext, vID uuid.UUID) (*ent.Playback, error) {
	uID := c.UserClaims.UserID

	playbackEntry, err := s.Store.Client.Playback.Query().Where(playback.UserID(uID)).Where(playback.VodID(vID)).Only(c.Request().Context())
	if err != nil {
		if _, ok := err.(*ent.NotFoundError); ok {
			return nil, fmt.Errorf("playback not found")
		}
		return nil, fmt.Errorf("error getting playback: %v", err)
	}

	return playbackEntry, nil
}

func (s *Service) GetAllProgress(c *auth.CustomContext) ([]*ent.Playback, error) {
	uID := c.UserClaims.UserID

	playbackEntries, err := s.Store.Client.Playback.Query().Where(playback.UserID(uID)).All(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error getting all playback: %v", err)
	}

	return playbackEntries, nil
}

func (s *Service) UpdateStatus(c *auth.CustomContext, vID uuid.UUID, status string) error {
	uID := c.UserClaims.UserID

	_, err := s.Store.Client.Playback.Query().Where(playback.UserID(uID)).Where(playback.VodID(vID)).Only(c.Request().Context())
	if err != nil {
		if _, ok := err.(*ent.NotFoundError); ok {

			_, err = s.Store.Client.Playback.Create().SetUserID(uID).SetVodID(vID).SetStatus(utils.PlaybackStatus(status)).Save(c.Request().Context())
			if err != nil {
				return fmt.Errorf("error creating playback: %v", err)
			}

			return nil
		}
		return fmt.Errorf("error checking playback: %v", err)
	}

	_, err = s.Store.Client.Playback.Update().Where(playback.UserID(uID)).Where(playback.VodID(vID)).SetStatus(utils.PlaybackStatus(status)).Save(c.Request().Context())
	if err != nil {
		if _, ok := err.(*ent.NotFoundError); ok {
			return fmt.Errorf("playback not found")
		}
		return fmt.Errorf("error updating playback: %v", err)
	}

	return nil
}

func (s *Service) DeleteProgress(c *auth.CustomContext, vID uuid.UUID) error {
	uID := c.UserClaims.UserID

	_, err := s.Store.Client.Playback.Delete().Where(playback.UserID(uID)).Where(playback.VodID(vID)).Exec(c.Request().Context())
	if err != nil {
		if _, ok := err.(*ent.NotFoundError); ok {
			return fmt.Errorf("playback not found")
		}
		return fmt.Errorf("error deleting playback: %v", err)
	}

	return nil
}
