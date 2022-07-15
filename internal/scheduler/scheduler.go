package scheduler

import (
	"github.com/go-co-op/gocron"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/twitch"
	"time"
)

type Service struct {
	LiveService *live.Service
}

func NewService(liveService *live.Service) *Service {
	return &Service{LiveService: liveService}
}

func (s *Service) StartAppScheduler() {
	scheduler := gocron.NewScheduler(time.UTC)

	s.twitchAuthSchedule(scheduler)

	scheduler.StartAsync()
}

func (s *Service) StartLiveScheduler() {
	time.Sleep(time.Second * 5)
	scheduler := gocron.NewScheduler(time.UTC)

	s.checkLiveStreamSchedule(scheduler)

	scheduler.StartAsync()
}

func (s *Service) twitchAuthSchedule(scheduler *gocron.Scheduler) {
	log.Debug().Msg("setting up twitch auth schedule")
	scheduler.Every(7).Days().Do(func() {
		log.Debug().Msg("running twitch auth schedule")
		err := twitch.Authenticate()
		if err != nil {
			log.Error().Err(err).Msg("failed to authenticate with twitch")
		}

	})
}

func (s *Service) checkLiveStreamSchedule(scheduler *gocron.Scheduler) {
	log.Debug().Msg("setting up check live stream schedule")
	scheduler.Every(5).Minutes().Do(func() {
		log.Debug().Msg("running check live stream schedule")
		err := s.LiveService.Check()
		if err != nil {
			log.Error().Err(err).Msg("failed to check live streams")
		}

	})
}
