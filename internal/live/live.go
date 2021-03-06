package live

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/channel"
	"github.com/zibbp/ganymede/ent/live"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/twitch"
	"time"
)

type Service struct {
	Store          *database.Database
	TwitchService  *twitch.Service
	ArchiveService *archive.Service
}

type Live struct {
	ID          uuid.UUID `json:"id"`
	IsLive      bool      `json:"is_live"`
	ArchiveChat bool      `json:"archive_chat"`
	Resolution  string    `json:"resolution"`
	LastLive    time.Time `json:"last_live"`
}

func NewService(store *database.Database, twitchService *twitch.Service, archiveService *archive.Service) *Service {
	return &Service{Store: store, TwitchService: twitchService, ArchiveService: archiveService}
}

func (s *Service) GetLiveWatchedChannels(c echo.Context) ([]*ent.Live, error) {
	watchedChannels, err := s.Store.Client.Live.Query().WithChannel().All(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error getting watched channels: %v", err)
	}
	return watchedChannels, nil
}

func (s *Service) AddLiveWatchedChannel(c echo.Context, liveDto Live) (*ent.Live, error) {
	// Check if channel is already in database
	liveWatchedChannel, err := s.Store.Client.Live.Query().WithChannel().Where(live.HasChannelWith(channel.ID(liveDto.ID))).All(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error getting live watched channel")
	}
	if len(liveWatchedChannel) > 0 {
		return nil, fmt.Errorf("channel already watched")
	}
	l, err := s.Store.Client.Live.Create().SetChannelID(liveDto.ID).SetResolution(liveDto.Resolution).SetArchiveChat(liveDto.ArchiveChat).Save(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error adding watched channel: %v", err)
	}
	return l, nil
}

func (s *Service) UpdateLiveWatchedChannel(c echo.Context, liveDto Live) (*ent.Live, error) {
	l, err := s.Store.Client.Live.UpdateOneID(liveDto.ID).SetResolution(liveDto.Resolution).SetArchiveChat(liveDto.ArchiveChat).Save(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error updating watched channel: %v", err)
	}
	return l, nil
}

func (s *Service) DeleteLiveWatchedChannel(c echo.Context, lID uuid.UUID) error {
	err := s.Store.Client.Live.DeleteOneID(lID).Exec(c.Request().Context())
	if err != nil {
		return fmt.Errorf("error deleting watched channel: %v", err)
	}
	return nil
}

//func  StartScheduler() {
//	s := gocron.NewScheduler(time.UTC)
//
//	twitchAuthSchedule(s)
//	s.StartAsync()
//}
//
//func liveCheckSchedule(s *gocron.Scheduler) {
//	log.Debug().Msg("setting up live check schedule")
//	s.Every(5).Minutes().Do(Check)
//}

func (s *Service) Check() error {
	log.Debug().Msg("checking live channels")
	// get live watched channels from database
	liveWatchedChannels, err := s.Store.Client.Live.Query().WithChannel().All(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error getting live watched channels")
	}
	if len(liveWatchedChannels) == 0 {
		log.Debug().Msg("no live watched channels")
		return nil
	}
	// Generate query string for Twitch API
	var queryString string

	for i, lwc := range liveWatchedChannels {
		if i == 0 {
			queryString += "?user_login=" + lwc.Edges.Channel.Name
		} else {
			queryString += "&user_login=" + lwc.Edges.Channel.Name
		}
	}

	twitchStreams, err := s.TwitchService.GetStreams(queryString)
	if err != nil {
		log.Error().Err(err).Msg("error getting twitch streams")
	}

	// check if live stream is online
	for _, lwc := range liveWatchedChannels {
		// Check if LWC is in twitchStreams.Data
		stream := stringInSlice(lwc.Edges.Channel.Name, twitchStreams.Data)
		if len(stream.ID) > 0 {
			if lwc.IsLive == false {
				log.Debug().Msgf("%s is now live", lwc.Edges.Channel.Name)
				// Stream is online, update database
				_, err := s.Store.Client.Live.UpdateOneID(lwc.ID).SetIsLive(true).Save(context.Background())
				if err != nil {
					log.Error().Err(err).Msg("error updating live watched channel")
				}
				// Archive stream
				_, err = s.ArchiveService.ArchiveTwitchLive(lwc, stream)
				if err != nil {
					log.Error().Err(err).Msg("error archiving twitch live")
				}

			}
		} else {
			if lwc.IsLive == true {
				log.Debug().Msgf("%s is now offline", lwc.Edges.Channel.Name)
				// Stream is offline, update database
				_, err := s.Store.Client.Live.UpdateOneID(lwc.ID).SetIsLive(false).SetLastLive(time.Now()).Save(context.Background())
				if err != nil {
					log.Error().Err(err).Msg("error updating live watched channel")
				}
			}
		}
	}

	return nil
}

func stringInSlice(a string, list []twitch.Live) twitch.Live {
	for _, b := range list {
		if b.UserLogin == a {
			return b
		}
	}
	return twitch.Live{}
}
