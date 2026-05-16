package live

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/channel"
	"github.com/zibbp/ganymede/ent/live"
	"github.com/zibbp/ganymede/ent/livetitleregex"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/utils"
)

type TwitchVideoResponse struct {
	Data       []Video    `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type Video struct {
	ID            string      `json:"id"`
	StreamID      string      `json:"stream_id"`
	UserID        string      `json:"user_id"`
	UserLogin     UserLogin   `json:"user_login"`
	UserName      UserName    `json:"user_name"`
	Title         string      `json:"title"`
	Description   string      `json:"description"`
	CreatedAt     string      `json:"created_at"`
	PublishedAt   string      `json:"published_at"`
	URL           string      `json:"url"`
	ThumbnailURL  string      `json:"thumbnail_url"`
	Viewable      Viewable    `json:"viewable"`
	ViewCount     int64       `json:"view_count"`
	Language      Language    `json:"language"`
	Type          Type        `json:"type"`
	Duration      string      `json:"duration"`
	MutedSegments interface{} `json:"muted_segments"`
}

type Pagination struct {
	Cursor string `json:"cursor"`
}

type Language string

type Type string

type UserLogin string

type UserName string

type Viewable string

func (s *Service) CheckVodWatchedChannels(ctx context.Context, logger zerolog.Logger) error {
	// Get channels from DB
	channels, err := s.Store.Client.Live.Query().Where(live.WatchVod(true)).WithChannel().WithCategories().WithTitleRegex(func(ltrq *ent.LiveTitleRegexQuery) {
		ltrq.Where(livetitleregex.ApplyToVideosEQ(true))
	}).All(context.Background())
	if err != nil {
		return err
	}

	if len(channels) == 0 {
		logger.Info().Msg("no channels to check")
		return nil
	}

	logger.Info().Msgf("checking %d channels for new videos", len(channels))

	for _, watch := range channels {
		// Check if channel has category restrictions
		var channelVideoCategories []string
		if len(watch.Edges.Categories) > 0 {
			for _, category := range watch.Edges.Categories {
				channelVideoCategories = append(channelVideoCategories, *category.Name)
			}
			logger.Debug().Msgf("channel %s has category restrictions: %s", watch.Edges.Channel.Name, strings.Join(channelVideoCategories, ", "))
		}

		var videos []platform.VideoInfo
		// If archives is enabled, fetch all videos
		if watch.DownloadArchives {
			tmpVideos, err := s.PlatformTwitch.GetVideos(ctx, watch.Edges.Channel.ExtID, platform.VideoTypeArchive, false, false)
			if err != nil {
				logger.Error().Str("channel", watch.Edges.Channel.Name).Err(err).Msg("error getting videos")
				continue
			}
			videos = append(videos, tmpVideos...)
		}
		// If highlights is enabled, fetch all videos
		if watch.DownloadHighlights {
			tmpVideos, err := s.PlatformTwitch.GetVideos(ctx, watch.Edges.Channel.ExtID, platform.VideoTypeHighlight, false, false)
			if err != nil {
				logger.Error().Str("channel", watch.Edges.Channel.Name).Err(err).Msg("error getting videos")
				continue
			}
			videos = append(videos, tmpVideos...)
		}
		// If uploads is enabled, fetch all videos
		if watch.DownloadUploads {
			tmpVideos, err := s.PlatformTwitch.GetVideos(ctx, watch.Edges.Channel.ExtID, platform.VideoTypeUpload, false, false)
			if err != nil {
				logger.Error().Str("channel", watch.Edges.Channel.Name).Err(err).Msg("error getting videos")
				continue
			}
			videos = append(videos, tmpVideos...)
		}

		// Fetch all videos from DB
		dbVideos, err := s.Store.Client.Vod.Query().Where(vod.HasChannelWith(channel.ID(watch.Edges.Channel.ID))).All(context.Background())
		if err != nil {
			logger.Error().Str("channel", watch.Edges.Channel.Name).Err(err).Msg("error getting videos from database")
			continue
		}
		// Check if video is already in DB
	OUTER:
		for _, video := range videos {
			// Video is not in DB
			if !contains(dbVideos, video.ID) {
				platformVideo, err := s.PlatformTwitch.GetVideo(ctx, video.ID, true, true)
				if err != nil {
					logger.Error().Str("channel", watch.Edges.Channel.Name).Err(err).Msg("error getting video")
					continue
				}
				// check if there are any title regexes that need to be tested
				if len(watch.Edges.TitleRegex) > 0 {
					// run regexes against title
					for _, titleRegex := range watch.Edges.TitleRegex {
						regex, err := regexp.Compile(titleRegex.Regex)
						if err != nil {
							logger.Error().Err(err).Msgf("error compiling regex %s", titleRegex.Regex)
							continue
						}
						matches := regex.FindAllString(video.Title, -1)

						if titleRegex.Negative && len(matches) == 0 {
							continue
						}

						if !titleRegex.Negative && len(matches) > 0 {
							continue
						}

						logger.Debug().Str("regex", titleRegex.Regex).Str("title", video.Title).Msgf("no regex matches for video")
						continue OUTER
					}
				}

				// check if video is too old
				if watch.VideoAge > 0 {

					currentTime := time.Now()
					ageDuration := time.Duration(watch.VideoAge) * 24 * time.Hour
					ageCutOff := currentTime.Add(-ageDuration)

					if platformVideo.CreatedAt.Before(ageCutOff) {
						logger.Debug().Str("video_id", video.ID).Msgf("skipping video; video is older than %d days.", watch.VideoAge)
						continue
					}
				}

				// Get video chapters
				var videoChapters []string

				if len(platformVideo.Chapters) > 0 {
					for _, chapter := range platformVideo.Chapters {
						videoChapters = append(videoChapters, chapter.Title)
					}
					logger.Debug().Str("video_id", video.ID).Str("chapters", strings.Join(videoChapters, ", ")).Msg("video has chapters")
				}

				// Append chapters and video category to video categories
				var videoCategories []string
				videoCategories = append(videoCategories, videoChapters...)
				if platformVideo.Category != nil {
					videoCategories = append(videoCategories, *platformVideo.Category)
				}

				// Check if video is sub only restricted
				if video.Restriction != nil && *video.Restriction == string(platform.VideoRestrictionSubscriber) {
					// Skip if sub only is disabled
					if !watch.DownloadSubOnly {
						logger.Info().Str("video_id", video.ID).Msgf("skipping subscriber-only video")
						continue
					}
					// Skip if Twitch token is not set
					if config.Get().Parameters.TwitchToken == "" {
						logger.Info().Str("video_id", video.ID).Msg("skipping sub only video; Twitch token is not set")
						continue
					}
				}

				// Check category restrictions / blacklists
				if len(channelVideoCategories) > 0 {
					var found bool
					for _, category := range videoCategories {
						for _, channelCategory := range channelVideoCategories {
							if strings.EqualFold(category, channelCategory) {
								found = true
								break
							}
						}
						if found {
							break
						}
					}

					if watch.BlacklistCategories {
						// If blacklist mode and a matching category was found, skip
						if found {
							logger.Info().
								Str("video_id", video.ID).
								Str("categories", strings.Join(videoCategories, ", ")).
								Str("blacklisted_categories", strings.Join(channelVideoCategories, ", ")).
								Msg("skipping video; video is in blacklisted categories")
							continue
						}
					} else {
						// If whitelist mode and no matching category was found, skip
						if !found {
							logger.Info().
								Str("video_id", video.ID).
								Str("categories", strings.Join(videoCategories, ", ")).
								Str("expected_categories", strings.Join(channelVideoCategories, ", ")).
								Msg("video does not match category restrictions")
							continue
						}
					}
				}

				// archive the video
				input := archive.ArchiveVideoInput{
					VideoId:     video.ID,
					Quality:     utils.VodQuality(watch.Resolution),
					ArchiveChat: watch.ArchiveChat,
					RenderChat:  watch.RenderChat,
				}
				_, err = s.ArchiveService.ArchiveVideo(ctx, input)
				if err != nil {
					log.Error().Err(err).Str("video_id", video.ID).Msgf("error archiving video")
					continue
				}
				logger.Info().Str("video_id", video.ID).Msgf("archiving video")
			}
		}
	}
	return nil
}

func contains(videos []*ent.Vod, id string) bool {
	for _, video := range videos {
		if video.ExtID == id {
			return true
		}
	}
	return false
}
