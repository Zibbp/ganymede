package vod

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	entChannel "github.com/zibbp/ganymede/ent/channel"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/database"
)

func PruneVideos(ctx context.Context, store *database.Database) error {
	vodService := &Service{Store: database.DB()}
	req := &http.Request{}
	echoCtx := echo.New().NewContext(req, nil)
	echoCtx.SetRequest(req.WithContext(ctx))

	// fetch all channels that have retention enable
	channels, err := store.Client.Channel.Query().Where(entChannel.Retention(true)).All(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error fetching channels")
		return err
	}
	log.Debug().Msgf("found %d channels with retention enabled", len(channels))

	// loop over channels
	for _, channel := range channels {
		log.Debug().Msgf("Processing channel %s", channel.ID)
		// fetch all videos for channel
		videos, err := store.Client.Vod.Query().Where(entVod.HasChannelWith(entChannel.ID(channel.ID))).All(context.Background())
		if err != nil {
			log.Error().Err(err).Msgf("Error fetching videos for channel %s", channel.ID)
			continue
		}

		// loop over videos
		for _, video := range videos {
			// check if video is locked
			if video.Locked {
				log.Debug().Str("video_id", video.ID.String()).Msg("skipping locked video")
				continue
			}
			// check if video is older than retention
			if video.CreatedAt.Add(time.Duration(channel.RetentionDays) * 24 * time.Hour).Before(time.Now()) {
				// delete video
				log.Info().Str("video_id", video.ID.String()).Msg("deleting video as it is older than retention")
				err := vodService.DeleteVod(ctx, video.ID, true)
				if err != nil {
					log.Error().Err(err).Msgf("Error deleting video %s", video.ID)
					continue
				}
			}
		}

	}

	return nil
}
