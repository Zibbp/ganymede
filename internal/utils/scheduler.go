package utils

import (
	"github.com/go-co-op/gocron"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/twitch"
	"time"
)

func StartScheduler() {
	s := gocron.NewScheduler(time.UTC)

	twitchAuthSchedule(s)
	s.StartAsync()
}

func twitchAuthSchedule(s *gocron.Scheduler) {
	log.Debug().Msg("setup twitch auth schedule")
	s.Every(7).Days().Do(func() {
		log.Debug().Msg("running twitch auth schedule")
		err := twitch.Authenticate()
		if err != nil {
			log.Error().Err(err).Msg("failed to authenticate with twitch")
		}

	})
}
