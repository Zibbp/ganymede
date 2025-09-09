package tasks_periodic

import (
	"context"
	"fmt"
	"time"

	"github.com/riverqueue/river"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent/mutedsegment"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/chapter"
	"github.com/zibbp/ganymede/internal/tasks"
	"github.com/zibbp/ganymede/internal/utils"
)

// Save chapters for all archived videos. Going forward this is done as part of the archive task, it's here to backfill old data.
type SaveVideoChaptersArgs struct{}

func (SaveVideoChaptersArgs) Kind() string { return tasks.TaskSaveVideoChapters }

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
		if video.Type == utils.Live || video.Type == utils.Clip {
			continue
		}
		if video.ExtID == "" {
			continue
		}

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
				log.Info().Str("video_id", video.ID.String()).Msgf("saving chapters for external video %s", video.ExtID)

				// save chapters to database
				for _, c := range platformVideo.Chapters {
					_, err := chapterService.CreateChapter(c, video.ID)
					if err != nil {
						return err
					}
				}

				log.Info().Str("video_id", video.ID.String()).Str("chapters", fmt.Sprintf("%d", len(platformVideo.Chapters))).Msgf("saved chapters for video")
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
