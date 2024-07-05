package live

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/channel"
	"github.com/zibbp/ganymede/ent/live"
	"github.com/zibbp/ganymede/ent/livetitleregex"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/twitch"
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

func (s *Service) CheckVodWatchedChannels() error {
	// Get channels from DB
	channels, err := s.Store.Client.Live.Query().Where(live.WatchVod(true)).WithChannel().WithCategories().WithTitleRegex(func(ltrq *ent.LiveTitleRegexQuery) {
		ltrq.Where(livetitleregex.ApplyToVideosEQ(true))
	}).All(context.Background())
	if err != nil {
		log.Debug().Err(err).Msg("error getting channels")
		return err
	}
	if len(channels) == 0 {
		log.Debug().Msg("No channels to check")
		return nil
	}
	log.Info().Msgf("Checking %d channels for new videos", len(channels))
	for _, watch := range channels {
		// Check if channel has category restrictions
		var channelVideoCategories []string
		if len(watch.Edges.Categories) > 0 {
			for _, category := range watch.Edges.Categories {
				channelVideoCategories = append(channelVideoCategories, category.Name)
			}
			log.Debug().Msgf("Channel %s has category restrictions: %s", watch.Edges.Channel.Name, strings.Join(channelVideoCategories, ", "))
		}

		var videos []twitch.Video
		// If archives is enabled, fetch all videos
		if watch.DownloadArchives {
			tmpVideos, err := twitch.GetVideosByUser(watch.Edges.Channel.ExtID, "archive")
			if err != nil {
				log.Error().Err(err).Msg("error getting videos")
				continue
			}
			videos = append(videos, tmpVideos...)
		}
		// If highlights is enabled, fetch all videos
		if watch.DownloadHighlights {
			tmpVideos, err := twitch.GetVideosByUser(watch.Edges.Channel.ExtID, "highlight")
			if err != nil {
				log.Error().Err(err).Msg("error getting videos")
				continue
			}
			videos = append(videos, tmpVideos...)
		}
		// If uploads is enabled, fetch all videos
		if watch.DownloadUploads {
			tmpVideos, err := twitch.GetVideosByUser(watch.Edges.Channel.ExtID, "upload")
			if err != nil {
				log.Error().Err(err).Msg("error getting videos")
				continue
			}
			videos = append(videos, tmpVideos...)
		}

		// Fetch all videos from DB
		dbVideos, err := s.Store.Client.Vod.Query().Where(vod.HasChannelWith(channel.ID(watch.Edges.Channel.ID))).All(context.Background())
		if err != nil {
			log.Error().Err(err).Msg("error getting videos from DB")
			continue
		}
		// Check if video is already in DB
	OUTER:
		for _, video := range videos {
			// Video is not in DB
			if !contains(dbVideos, video.ID) {
				// check if there are any title regexes that need to be tested
				if watch.Edges.TitleRegex != nil && len(watch.Edges.TitleRegex) > 0 {
					// run regexes against title
					for _, titleRegex := range watch.Edges.TitleRegex {
						regex, err := regexp.Compile(titleRegex.Regex)
						if err != nil {
							log.Error().Err(err).Msg("error compiling regex for watched channel check, skipping this regex")
							continue
						}
						matches := regex.FindAllString(video.Title, -1)

						if titleRegex.Negative && len(matches) == 0 {
							continue
						}

						if !titleRegex.Negative && len(matches) > 0 {
							continue
						}

						log.Debug().Str("regex", titleRegex.Regex).Str("title", video.Title).Msgf("no regex matches for video")
						continue OUTER
					}
				}

				// Query the video using Twitch's GraphQL API to check for restrictions
				gqlVideo, err := twitch.GQLGetVideo(video.ID)
				if err != nil {
					log.Error().Err(err).Msgf("error getting video %s from GraphQL API", video.ID)
					continue
				}

				// check if video is too old
				if watch.VideoAge > 0 {
					parsedTime, err := time.Parse(time.RFC3339, video.CreatedAt)
					if err != nil {
						log.Error().Err(err).Msgf("error parsing video %s created_at", video.ID)
						continue
					}

					currentTime := time.Now()
					ageDuration := time.Duration(watch.VideoAge) * 24 * time.Hour
					ageCutOff := currentTime.Add(-ageDuration)

					if parsedTime.Before(ageCutOff) {
						log.Debug().Msgf("skipping video %s. video is older than %d days.", video.ID, watch.VideoAge)
						continue
					}
				}

				// Get video chapters
				gqlVideoChapters, err := twitch.GQLGetChapters(video.ID)
				if err != nil {
					log.Error().Err(err).Msgf("error getting video %s chapters from GraphQL API", video.ID)
					continue
				}
				var videoChapters []string

				if len(gqlVideoChapters.Data.Video.Moments.Edges) > 0 {
					for _, chapter := range gqlVideoChapters.Data.Video.Moments.Edges {
						videoChapters = append(videoChapters, chapter.Node.Details.Game.DisplayName)
					}
					log.Debug().Msgf("Video %s has chapters: %s", video.ID, strings.Join(videoChapters, ", "))
				}

				// Append chapters and video category to video categories
				var videoCategories []string
				videoCategories = append(videoCategories, videoChapters...)
				videoCategories = append(videoCategories, gqlVideo.Data.Video.Game.Name)

				// Check if video is sub only restricted
				if strings.Contains(gqlVideo.Data.Video.ResourceRestriction.Type, "SUB") {
					// Skip if sub only is disabled
					if !watch.DownloadSubOnly {
						log.Info().Msgf("skipping sub only video %s.", video.ID)
						continue
					}
					// Skip if Twitch token is not set
					if viper.GetString("parameters.twitch_token") == "" {
						log.Info().Msgf("skipping sub only video %s. Twitch token is not set.", video.ID)
						continue
					}
				}

				// Check if video is in category restrictions, continue if not
				if len(channelVideoCategories) > 0 {
					var found bool
					for _, category := range videoCategories {
						for _, channelCategory := range channelVideoCategories {
							if strings.EqualFold(category, channelCategory) {
								found = true
								break
							}
						}
					}
					if !found {
						log.Info().Msgf("skipping video %s. video has categories of %s when the restriction requires %s.", video.ID, strings.Join(videoCategories, ", "), strings.Join(channelVideoCategories, ", "))
						continue
					}
				}

				// archive the video
				// _, err = s.ArchiveService.ArchiveTwitchVod(video.ID, watch.Resolution, watch.ArchiveChat, watch.RenderChat)
				// if err != nil {
				// 	log.Error().Err(err).Msgf("Error archiving video %s", video.ID)
				// 	continue
				// }
				// log.Info().Msgf("[Channel Watch] starting archive for video %s", video.ID)
			}
		}
	}
	log.Info().Msg("Finished checking channels for new videos")

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
