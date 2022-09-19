package scheduler

import (
	"github.com/go-co-op/gocron"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/twitch"
	"time"
)

type Service struct {
	LiveService    *live.Service
	ArchiveService *archive.Service
}

func NewService(liveService *live.Service, archiveService *archive.Service) *Service {
	return &Service{LiveService: liveService, ArchiveService: archiveService}
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

func (s *Service) StartQueueItemScheduler() {
	time.Sleep(time.Second * 5)
	scheduler := gocron.NewScheduler(time.UTC)

	s.checkHeldQueueItems(scheduler)

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
	configLiveCheckInterval := viper.GetInt("live_check_interval_seconds")
	log.Debug().Msgf("setting live check interval to run every %d seconds", configLiveCheckInterval)
	scheduler.Every(configLiveCheckInterval).Seconds().Do(func() {
		log.Debug().Msg("running check live stream schedule")
		err := s.LiveService.Check()
		if err != nil {
			log.Error().Err(err).Msg("failed to check live streams")
		}

	})
}

func (s *Service) checkHeldQueueItems(scheduler *gocron.Scheduler) {
	log.Debug().Msg("setting up queue item schedule")
	scheduler.Every(1).Hours().Do(func() {
		log.Debug().Msg("running queue item schedule")
		go s.ArchiveService.CheckOnHold()
	})
}
