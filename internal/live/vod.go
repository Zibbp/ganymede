package live

import (
	"context"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/channel"
	"github.com/zibbp/ganymede/ent/live"
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

func (s *Service) CheckVodWatchedChannels() {
	// Get channels from DB
	channels, err := s.Store.Client.Live.Query().Where(live.WatchVod(true)).WithChannel().All(context.Background())
	if err != nil {
		log.Debug().Err(err).Msg("error getting channels")
		return
	}
	if len(channels) == 0 {
		log.Debug().Msg("No channels to check")
		return
	}
	log.Info().Msgf("Checking %d channels for new videos", len(channels))
	for _, watch := range channels {
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
		for _, video := range videos {
			if !contains(dbVideos, video.ID) {
				// Video is not in DB

				// Query the video using Twitch's GraphQL API to check for restrictions
				gqlVideo, err := twitch.GQLGetVideo(video.ID)
				if err != nil {
					log.Error().Err(err).Msgf("error getting video %s from GraphQL API", video.ID)
					continue
				}
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

				// archive the video
				_, err = s.ArchiveService.ArchiveTwitchVod(video.ID, watch.Resolution, watch.ArchiveChat, true)
				if err != nil {
					log.Error().Err(err).Msgf("Error archiving video %s", video.ID)
					continue
				}
				log.Info().Msgf("[Channel Watch] starting archive for video %s", video.ID)
			}
		}
	}
	log.Info().Msg("Finished checking channels for new videos")
}

func contains(videos []*ent.Vod, id string) bool {
	for _, video := range videos {
		if video.ExtID == id {
			return true
		}
	}
	return false
}
