package live

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/channel"
	"github.com/zibbp/ganymede/ent/live"
	entLive "github.com/zibbp/ganymede/ent/live"
	"github.com/zibbp/ganymede/ent/livecategory"
	"github.com/zibbp/ganymede/ent/livetitleregex"
	"github.com/zibbp/ganymede/ent/queue"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/notification"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/utils"
)

type Service struct {
	Store          *database.Database
	ArchiveService *archive.Service
	PlatformTwitch platform.Platform
}

type Live struct {
	ID                    uuid.UUID            `json:"id"`
	WatchLive             bool                 `json:"watch_live"`
	WatchVod              bool                 `json:"watch_vod"`
	DownloadArchives      bool                 `json:"download_archives"`
	DownloadHighlights    bool                 `json:"download_highlights"`
	DownloadUploads       bool                 `json:"download_uploads"`
	IsLive                bool                 `json:"is_live"`
	ArchiveChat           bool                 `json:"archive_chat"`
	Resolution            string               `json:"resolution"`
	LastLive              time.Time            `json:"last_live"`
	RenderChat            bool                 `json:"render_chat"`
	DownloadSubOnly       bool                 `json:"download_sub_only"`
	Categories            []string             `json:"categories"`
	ApplyCategoriesToLive bool                 `json:"apply_categories_to_live"`
	VideoAge              int64                `json:"video_age"` // Restrict fetching videos to a certain age.
	TitleRegex            []ent.LiveTitleRegex `json:"title_regex"`
	WatchClips            bool                 `json:"watch_clips"`
	ClipsLimit            int                  `json:"clips_limit"`
	ClipsIntervalDays     int                  `json:"clips_interval_days"`
}

type ConvertChat struct {
	FileName      string `json:"file_name"`
	ChannelName   string `json:"channel_name"`
	VodID         string `json:"vod_id"`
	ChannelID     int    `json:"channel_id"`
	VodExternalID string `json:"vod_external_id"`
	ChatStart     string `json:"chat_start"`
}

type ArchiveLive struct {
	ChannelID   uuid.UUID `json:"channel_id"`
	Resolution  string    `json:"resolution"`
	ArchiveChat bool      `json:"archive_chat"`
	RenderChat  bool      `json:"render_chat"`
}

func NewService(store *database.Database, archiveService *archive.Service, platformTwitch platform.Platform) *Service {
	return &Service{Store: store, ArchiveService: archiveService, PlatformTwitch: platformTwitch}
}

func (s *Service) GetLiveWatchedChannels(c echo.Context) ([]*ent.Live, error) {
	watchedChannels, err := s.Store.Client.Live.Query().WithChannel().WithCategories().WithTitleRegex().All(c.Request().Context())
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

	l, err := s.Store.Client.Live.Create().SetChannelID(liveDto.ID).SetWatchLive(liveDto.WatchLive).SetWatchVod(liveDto.WatchVod).SetDownloadArchives(liveDto.DownloadArchives).SetDownloadHighlights(liveDto.DownloadHighlights).SetDownloadUploads(liveDto.DownloadUploads).SetResolution(liveDto.Resolution).SetArchiveChat(liveDto.ArchiveChat).SetRenderChat(liveDto.RenderChat).SetDownloadSubOnly(liveDto.DownloadSubOnly).SetVideoAge(liveDto.VideoAge).SetApplyCategoriesToLive(liveDto.ApplyCategoriesToLive).SetWatchClips(liveDto.WatchClips).SetClipsLimit(liveDto.ClipsLimit).SetClipsIntervalDays(liveDto.ClipsIntervalDays).Save(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error adding watched channel: %v", err)
	}
	// If category is set, add to database
	if len(liveDto.Categories) > 0 {
		for _, category := range liveDto.Categories {
			_, err := s.Store.Client.LiveCategory.Create().SetName(category).SetLive(l).Save(c.Request().Context())
			if err != nil {
				return nil, fmt.Errorf("error adding category: %v", err)
			}
		}
	}
	// add title regexes
	if len(liveDto.TitleRegex) > 0 {
		for _, regex := range liveDto.TitleRegex {
			_, err := s.Store.Client.LiveTitleRegex.Create().SetNegative(regex.Negative).SetApplyToVideos(regex.ApplyToVideos).SetRegex(regex.Regex).SetLive(l).Save(c.Request().Context())
			if err != nil {
				return nil, fmt.Errorf("error adding title regex: %v", err)
			}
		}
	}
	return l, nil
}

func (s *Service) UpdateLiveWatchedChannel(c echo.Context, liveDto Live) (*ent.Live, error) {
	l, err := s.Store.Client.Live.UpdateOneID(liveDto.ID).SetWatchLive(liveDto.WatchLive).SetWatchVod(liveDto.WatchVod).SetDownloadArchives(liveDto.DownloadArchives).SetDownloadHighlights(liveDto.DownloadHighlights).SetDownloadUploads(liveDto.DownloadUploads).SetResolution(liveDto.Resolution).SetArchiveChat(liveDto.ArchiveChat).SetRenderChat(liveDto.RenderChat).SetDownloadSubOnly(liveDto.DownloadSubOnly).SetVideoAge(liveDto.VideoAge).SetApplyCategoriesToLive(liveDto.ApplyCategoriesToLive).SetClipsLimit(liveDto.ClipsLimit).SetClipsIntervalDays(liveDto.ClipsIntervalDays).SetWatchClips(liveDto.WatchClips).Save(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error updating watched channel: %v", err)
	}

	// Delete all categories
	_, err = s.Store.Client.LiveCategory.Delete().Where(livecategory.HasLiveWith(live.ID(liveDto.ID))).Exec(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error deleting categories: %v", err)
	}

	// Update categories
	if len(liveDto.Categories) > 0 {
		// Add new categories
		for _, category := range liveDto.Categories {
			_, err := s.Store.Client.LiveCategory.Create().SetName(category).SetLive(l).Save(c.Request().Context())
			if err != nil {
				return nil, fmt.Errorf("error adding category: %v", err)
			}
		}
	}

	// delete all title regexes
	_, err = s.Store.Client.LiveTitleRegex.Delete().Where(livetitleregex.HasLiveWith(live.ID(liveDto.ID))).Exec(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error deleting title regexes: %v", err)
	}

	// update title regexes
	if len(liveDto.TitleRegex) > 0 {
		for _, regex := range liveDto.TitleRegex {
			_, err := s.Store.Client.LiveTitleRegex.Create().SetNegative(regex.Negative).SetApplyToVideos(regex.ApplyToVideos).SetRegex(regex.Regex).SetLive(l).Save(c.Request().Context())
			if err != nil {
				return nil, fmt.Errorf("error adding title regex: %v", err)
			}
		}
	}

	return l, nil
}

func (s *Service) DeleteLiveWatchedChannel(c echo.Context, lID uuid.UUID) error {
	// delete watched channel and categories
	v, err := s.Store.Client.Live.Query().Where(entLive.ID(lID)).WithCategories().Only(c.Request().Context())
	if err != nil {
		if _, ok := err.(*ent.NotFoundError); ok {
			return fmt.Errorf("watched channel not found")
		}
		return fmt.Errorf("error deleting watched channel: %v", err)
	}
	if v.Edges.Categories != nil {
		for _, category := range v.Edges.Categories {
			err := s.Store.Client.LiveCategory.DeleteOneID(category.ID).Exec(c.Request().Context())
			if err != nil {
				return fmt.Errorf("error deleting watched channel: %v", err)
			}
		}
	}

	err = s.Store.Client.Live.DeleteOneID(lID).Exec(c.Request().Context())
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

func (s *Service) Check(ctx context.Context) error {
	log.Debug().Msg("checking live channels")
	// get live watched channels from database
	liveWatchedChannels, err := s.Store.Client.Live.Query().Where(live.WatchLive(true)).WithChannel().WithCategories().WithTitleRegex(func(ltrq *ent.LiveTitleRegexQuery) {
		ltrq.Where(livetitleregex.ApplyToVideosEQ(false))
	}).All(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error getting live watched channels")
	}
	if len(liveWatchedChannels) == 0 {
		log.Debug().Msg("no live watched channels")
		return nil
	}

	// split into 99 channels per requests to avoid 100 channel limit
	var liveWatchedChannelsSplit [][]*ent.Live
	for i := 0; i < len(liveWatchedChannels); i += 99 {
		end := i + 99
		if end > len(liveWatchedChannels) {
			end = len(liveWatchedChannels)
		}
		liveWatchedChannelsSplit = append(liveWatchedChannelsSplit, liveWatchedChannels[i:end])
	}

	var streams []platform.LiveStreamInfo
	channels := make([]string, 0)
	// generate query string for twitch api
	for _, lwc := range liveWatchedChannelsSplit {
		for _, lwc := range lwc {
			channels = append(channels, lwc.Edges.Channel.Name)
		}
		log.Debug().Str("channels", strings.Join(channels, ", ")).Msg("checking live streams")

		twitchStreams, err := s.PlatformTwitch.GetLiveStreams(ctx, channels)
		if err != nil {
			if errors.Is(err, &platform.ErrorNoStreamsFound{}) {
				log.Debug().Msg("no streams found")
				continue
			} else {
				return fmt.Errorf("error getting live streams: %v", err)
			}
		}

		streams = append(streams, twitchStreams...)
	}

	// check if live stream is online
OUTER:
	for _, lwc := range liveWatchedChannels {
		// Check if LWC is in twitchStreams.Data
		stream := channelInLiveStreamInfo(lwc.Edges.Channel.Name, streams)
		if len(stream.ID) > 0 {
			if !lwc.IsLive {
				// stream is live
				log.Debug().Str("channel", lwc.Edges.Channel.Name).Msg("stream is live; checking for restrictions before archiving")
				// check for any user-constraints before archiving
				if len(lwc.Edges.TitleRegex) > 0 {
					// run regexes against title
					for _, titleRegex := range lwc.Edges.TitleRegex {
						regex, err := regexp.Compile(titleRegex.Regex)
						if err != nil {
							log.Error().Err(err).Msg("error compiling regex for watched channel check, skipping this regex")
							continue
						}
						matches := regex.FindAllString(stream.Title, -1)

						if titleRegex.Negative && len(matches) == 0 {
							continue
						}

						if !titleRegex.Negative && len(matches) > 0 {
							continue
						}

						log.Debug().Str("regex", titleRegex.Regex).Str("title", stream.Title).Msgf("no regex matches for stream")
						continue OUTER
					}
				}

				tmpCategoryNames := make([]string, 0)
				for _, category := range lwc.Edges.Categories {
					tmpCategoryNames = append(tmpCategoryNames, category.Name)
				}

				// check for category restrictions
				if lwc.ApplyCategoriesToLive && len(lwc.Edges.Categories) > 0 {
					found := false
					for _, category := range lwc.Edges.Categories {
						if strings.EqualFold(category.Name, stream.GameName) {
							log.Debug().Str("category", stream.GameName).Str("category_restrictions", strings.Join(tmpCategoryNames, ", ")).Msgf("%s matches category restrictions", lwc.Edges.Channel.Name)
							found = true
							break
						}
					}

					if !found {
						log.Debug().Str("category", stream.GameName).Str("category_restrictions", strings.Join(tmpCategoryNames, ", ")).Msgf("%s does not match category restrictions", lwc.Edges.Channel.Name)
						continue
					}
				}

				log.Debug().Msgf("%s is now live", lwc.Edges.Channel.Name)
				// Stream is online, update database
				_, err := s.Store.Client.Live.UpdateOneID(lwc.ID).SetIsLive(true).Save(context.Background())
				if err != nil {
					log.Error().Err(err).Msg("error updating live watched channel")
				}
				// check if stream is already being archived
				queueItems, err := database.DB().Client.Queue.Query().Where(queue.Processing(true)).WithVod().All(context.Background())
				if err != nil {
					log.Error().Err(err).Msg("error getting queue items")
				}
				for _, queueItem := range queueItems {
					if queueItem.Edges.Vod.ExtID == stream.ID && queueItem.TaskVideoDownload == utils.Running {
						log.Debug().Msgf("%s is already being archived", lwc.Edges.Channel.Name)
						return nil
					}
				}
				// Archive stream
				err = s.ArchiveService.ArchiveLivestream(ctx, archive.ArchiveVideoInput{
					ChannelId:   lwc.Edges.Channel.ID,
					Quality:     utils.VodQuality(lwc.Resolution),
					ArchiveChat: lwc.ArchiveChat,
					RenderChat:  lwc.RenderChat,
				})
				if err != nil {
					log.Error().Err(err).Msg("error archiving twitch livestream")
				}
				// Notification
				// Fetch channel for notification
				vod, err := s.Store.Client.Vod.Query().Where(entVod.ExtStreamID(stream.ID)).WithChannel().WithQueue().Order(entVod.ByCreatedAt()).Limit(1).First(ctx)
				if err != nil {
					log.Error().Err(err).Msg("error getting vod")
					continue
				}
				go notification.SendLiveNotification(lwc.Edges.Channel, vod, vod.Edges.Queue)
			}
		} else {
			if lwc.IsLive {
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

// channelInLiveStreamInfo searches for a string in a slice of LiveStreamInfo and returns the first match.
func channelInLiveStreamInfo(a string, list []platform.LiveStreamInfo) platform.LiveStreamInfo {
	for _, b := range list {
		if b.UserLogin == a {
			return b
		}
	}
	return platform.LiveStreamInfo{}
}
