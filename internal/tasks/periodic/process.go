package tasks_periodic

import (
	"context"
	"fmt"
	"time"

	"github.com/riverqueue/river"
	"github.com/rs/zerolog/log"
	entChannel "github.com/zibbp/ganymede/ent/channel"
	"github.com/zibbp/ganymede/ent/mutedsegment"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/chapter"
	platformPkg "github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/tasks"
	"github.com/zibbp/ganymede/internal/utils"
)

// Save chapters for all archived videos. Going forward this is done as part of the archive task, it's here to backfill old data.
type SaveVideoChaptersArgs struct{}

func (SaveVideoChaptersArgs) Kind() string { return "save_video_chapters" }

func (w SaveVideoChaptersArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
	}
}

func (w SaveVideoChaptersArgs) Timeout(job *river.Job[SaveVideoChaptersArgs]) time.Duration {
	return 10 * time.Minute
}

type SaveVideoChaptersWorker struct {
	river.WorkerDefaults[SaveVideoChaptersArgs]
}

func (w SaveVideoChaptersWorker) Work(ctx context.Context, job *river.Job[SaveVideoChaptersArgs]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()
	logger.Info().Msg("starting task")

	store, err := tasks.StoreFromContext(ctx)
	if err != nil {
		return err
	}

	platform, err := tasks.PlatformFromContext(ctx)
	if err != nil {
		return err
	}

	// get all videos
	videos, err := store.Client.Vod.Query().All(ctx)
	if err != nil {
		return err
	}

	for _, video := range videos {
		if video.Type == utils.Live {
			continue
		}
		if video.ExtID == "" {
			continue
		}

		log.Info().Msgf("saving chapters for video %s", video.ExtID)
		platformVideo, err := platform.GetVideo(ctx, video.ExtID, true, true)
		if err != nil {
			return err
		}

		if len(platformVideo.Chapters) > 0 {
			chapterService := chapter.NewService(store)

			existingVideoChapters, err := chapterService.GetVideoChapters(video.ID)
			if err != nil {
				return err
			}

			if len(existingVideoChapters) == 0 {

				// save chapters to database
				for _, c := range platformVideo.Chapters {
					_, err := chapterService.CreateChapter(c, video.ID)
					if err != nil {
						return err
					}
				}

				log.Info().Str("video_id", fmt.Sprintf("%d", video.ID)).Str("chapters", fmt.Sprintf("%d", len(platformVideo.Chapters))).Msgf("saved chapters for video")
			}
		}

		if len(platformVideo.MutedSegments) > 0 {
			existingMutedSegments, err := store.Client.MutedSegment.Query().Where(mutedsegment.HasVodWith(vod.ID(video.ID))).All(ctx)
			if err != nil {
				return err
			}

			if len(existingMutedSegments) == 0 {

				// save muted segments to database
				for _, segment := range platformVideo.MutedSegments {
					// parse twitch duration
					segmentEnd := segment.Offset + segment.Duration
					if segmentEnd > int(platformVideo.Duration.Seconds()) {
						segmentEnd = int(platformVideo.Duration.Seconds())
					}
					// insert into database
					_, err := store.Client.MutedSegment.Create().SetStart(segment.Offset).SetEnd(segmentEnd).SetVod(video).Save(ctx)
					if err != nil {
						return err
					}
				}

				log.Info().Str("video_id", fmt.Sprintf("%d", video.ID)).Str("muted_segments", fmt.Sprintf("%d", len(platformVideo.MutedSegments))).Msgf("saved muted segments for video")
			}
		}

		// avoid rate limiting
		time.Sleep(250 * time.Millisecond)
	}

	logger.Info().Msg("task completed")

	return nil
}

// Save chapters for all archived videos. Going forward this is done as part of the archive task, it's here to backfill old data.
type UpdateLivestreamVodIdsArgs struct{}

func (UpdateLivestreamVodIdsArgs) Kind() string { return "update_live_stream_vod_ids" }

func (w UpdateLivestreamVodIdsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
	}
}

func (w UpdateLivestreamVodIdsArgs) Timeout(job *river.Job[UpdateLivestreamVodIdsArgs]) time.Duration {
	return 10 * time.Minute
}

type UpdateLivestreamVodIdsWorker struct {
	river.WorkerDefaults[UpdateLivestreamVodIdsArgs]
}

func (w UpdateLivestreamVodIdsWorker) Work(ctx context.Context, job *river.Job[UpdateLivestreamVodIdsArgs]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()
	logger.Info().Msg("starting task")

	store, err := tasks.StoreFromContext(ctx)
	if err != nil {
		return err
	}

	platform, err := tasks.PlatformFromContext(ctx)
	if err != nil {
		return err
	}

	channels, err := store.Client.Channel.Query().All(ctx)
	if err != nil {
		return err
	}

	// need to loop over each channel and get all channel videos
	// this is because the 'streamid' is not an id we can query from APIs
	for _, channel := range channels {
		logger.Info().Str("channel", channel.Name).Msg("fetching channel videos")
		videos, err := store.Client.Vod.Query().Where(vod.HasChannelWith(entChannel.ID(channel.ID))).All(ctx)
		if err != nil {
			return err
		}

		// get all channel videos from platform
		platformVideos, err := platform.GetVideos(ctx, channel.ExtID, platformPkg.VideoTypeArchive, false, false)
		if err != nil {
			return err
		}

		logger.Info().Str("channel", channel.Name).Msgf("found %d videos in platform", len(platformVideos))

		for _, video := range videos {
			if video.Type != utils.Live {
				continue
			}
			if video.ExtID == "" {
				continue
			}

			// attempt to find video in list of platform videos
			for _, platformVideo := range platformVideos {
				if platformVideo.StreamID == video.ExtStreamID {
					logger.Info().Str("channel", channel.Name).Str("video_id", video.ID.String()).Msg("found video in platform")
					_, err := store.Client.Vod.UpdateOneID(video.ID).SetExtID(platformVideo.ID).Save(ctx)
					if err != nil {
						return err
					}
					break
				}
			}

		}
	}

	logger.Info().Msg("task completed")

	return nil
}
