package task

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/auth"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/twitch"
)

type Service struct {
	Store          *database.Database
	LiveService    *live.Service
	ArchiveService *archive.Service
}

func NewService(store *database.Database, liveService *live.Service, archiveService *archive.Service) *Service {
	return &Service{Store: store, LiveService: liveService, ArchiveService: archiveService}
}

func (s *Service) StartTask(c echo.Context, task string) error {
	log.Info().Msgf("Manually starting task %s", task)

	switch task {
	case "check_live":
		err := s.LiveService.Check()
		if err != nil {
			return fmt.Errorf("error checking live: %v", err)
		}

	case "check_vod":
		go s.LiveService.CheckVodWatchedChannels()

	case "get_jwks":
		err := auth.FetchJWKS()
		if err != nil {
			return fmt.Errorf("error fetching jwks: %v", err)
		}

	case "twitch_auth":
		err := twitch.Authenticate()
		if err != nil {
			return fmt.Errorf("error authenticating twitch: %v", err)
		}

	case "queue_hold_check":
		go s.ArchiveService.CheckOnHold()
	}

	return nil
}
