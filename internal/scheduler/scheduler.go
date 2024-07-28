package scheduler

import (
	"context"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/live"
)

type Service struct {
	LiveService    *live.Service
	ArchiveService *archive.Service
}

func NewService(liveService *live.Service, archiveService *archive.Service) *Service {
	return &Service{LiveService: liveService, ArchiveService: archiveService}
}

func (s *Service) StartLiveScheduler() {
	time.Sleep(time.Second * 5)
	scheduler := gocron.NewScheduler(time.UTC)

	s.checkLiveStreamSchedule(scheduler)

	scheduler.StartAsync()
}

func (s *Service) checkLiveStreamSchedule(scheduler *gocron.Scheduler) {
	log.Debug().Msg("setting up check live stream schedule")
	config := config.Get()
	log.Debug().Msgf("setting live check interval to run every %d seconds", config.LiveCheckInterval)
	_, err := scheduler.Every(config.LiveCheckInterval).Seconds().Do(func() {
		ctx := context.Background()
		log.Debug().Msg("running check live stream schedule")
		err := s.LiveService.Check(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to check live streams")
		}
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to set up check live stream schedule")
	}
}
