package tasks

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/riverqueue/river"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/exec"
	"github.com/zibbp/ganymede/internal/utils"
)

// ///////////////////////
// Download Video (VOD) //
// ///////////////////////
type DownloadVideoArgs struct {
	Continue bool              `json:"continue"`
	Input    ArchiveVideoInput `json:"input"`
}

func (DownloadVideoArgs) Kind() string { return string(utils.TaskDownloadVideo) }

func (args DownloadVideoArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
		Queue:       QueueVideoDownload,
		Tags:        []string{"archive"},
		UniqueOpts:  archiveUniqueOpts(),
	}
}

func (w *DownloadVideoWorker) Timeout(job *river.Job[DownloadVideoArgs]) time.Duration {
	return 49 * time.Hour
}

type DownloadVideoWorker struct {
	river.WorkerDefaults[DownloadVideoArgs]
}

func (w DownloadVideoWorker) Work(ctx context.Context, job *river.Job[DownloadVideoArgs]) error {
	// get store from context
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Running,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskDownloadVideo,
	})
	if err != nil {
		return err
	}

	dbItems, err := getDatabaseItems(ctx, store.Client, job.Args.Input.QueueId)
	if err != nil {
		return err
	}

	// download video
	err = exec.DownloadTwitchVideo(ctx, dbItems.Video)
	if err != nil {
		return err
	}

	next := []transactionalJob{}
	if job.Args.Continue {
		next = append(next, transactionalJob{Args: &PostProcessVideoArgs{Continue: true, Input: nextArchiveInput(job.Args.Input)}})
	}
	err = setQueueStatusAndEnqueue(ctx, store, QueueStatusInput{
		Status:  utils.Success,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskDownloadVideo,
	}, next...)
	if err != nil {
		return err
	}

	// check if tasks are done
	if err := checkIfTasksAreDone(ctx, store.Client, job.Args.Input); err != nil {
		return err
	}

	return nil
}

// ////////////////////
// Postprocess Video //
// ////////////////////
type PostProcessVideoArgs struct {
	Continue bool              `json:"continue"`
	Input    ArchiveVideoInput `json:"input"`
}

func (PostProcessVideoArgs) Kind() string { return string(utils.TaskPostProcessVideo) }

func (args PostProcessVideoArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
		Queue:       QueueVideoPostProcess,
		Tags:        []string{"archive"},
		UniqueOpts:  archiveUniqueOpts(),
	}
}

func (w *PostProcessVideoWorker) Timeout(job *river.Job[PostProcessVideoArgs]) time.Duration {
	return 24 * time.Hour
}

type PostProcessVideoWorker struct {
	river.WorkerDefaults[PostProcessVideoArgs]
}

func validateNonEmptyFile(path string, label string) error {
	if !utils.FileExists(path) {
		return fmt.Errorf("missing %s: %s", label, path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat %s %s: %w", label, path, err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("empty %s: %s", label, path)
	}

	return nil
}

// setLiveVideoDurationAndFinalizeChapters records the finalized live video
// duration and closes any open chapter at that duration.
func setLiveVideoDurationAndFinalizeChapters(ctx context.Context, video *ent.Vod, duration int) error {
	if _, err := video.Update().SetDuration(duration).Save(ctx); err != nil {
		return err
	}

	chapters, err := video.QueryChapters().All(ctx)
	if err != nil {
		return err
	}
	for _, chapter := range chapters {
		if chapter.End != 0 {
			continue
		}
		log.Debug().Str("video_id", video.ID.String()).Int("duration", duration).Msg("updating live chapter end time")
		if _, err := chapter.Update().SetEnd(duration).Save(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (w PostProcessVideoWorker) Work(ctx context.Context, job *river.Job[PostProcessVideoArgs]) error {
	// get store from context
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	// set queue status to running
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Running,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskPostProcessVideo,
	})
	if err != nil {
		return err
	}

	dbItems, err := getDatabaseItems(ctx, store.Client, job.Args.Input.QueueId)
	if err != nil {
		return err
	}

	// explicit pre-flight validation so downstream failures are clear/actionable.
	if !dbItems.Queue.LiveArchive {
		if err := validateNonEmptyFile(dbItems.Video.TmpVideoDownloadPath, "downloaded video input"); err != nil {
			return err
		}
	}

	// Live archives capture HLS; the finalized playlist (rebuilt from
	// segments if a kill truncated it) is required by the duration block below.
	// Post-process to MP4 only for non-live archives.
	shouldPostProcessVideo := !dbItems.Queue.LiveArchive

	if shouldPostProcessVideo {
		err = exec.PostProcessVideo(ctx, dbItems.Video)
		if err != nil {
			return err
		}
	}

	// update video duration for live archive
	if dbItems.Queue.LiveArchive {
		playlistPath, err := exec.EnsureLiveHlsPlaylist(ctx, dbItems.Video.TmpVideoHlsPath, liveCaptureID(&dbItems.Video))
		if err != nil {
			return err
		}
		duration, err := exec.GetVideoDuration(ctx, playlistPath)
		if err != nil {
			return err
		}
		if err := setLiveVideoDurationAndFinalizeChapters(ctx, &dbItems.Video, duration); err != nil {
			return err
		}

		// Final MP4 converts HLS here; final HLS needs no conversion.
		// Reuse an existing export for idempotent retries.
		if dbItems.Video.VideoHlsPath == "" {
			if utils.FileExists(dbItems.Video.TmpVideoConvertPath) {
				if err := validateNonEmptyFile(dbItems.Video.TmpVideoConvertPath, "live exported MP4 output"); err != nil {
					return err
				}
			} else {
				if err := exec.ExportHlsToMp4(ctx, dbItems.Video, dbItems.Video.TmpVideoConvertPath); err != nil {
					return err
				}
			}
		}
	}

	// convert non live archive video
	if !dbItems.Queue.LiveArchive {
		// convert to HLS if needed
		if config.Get().Archive.SaveAsHls {
			// create temp hls directory
			if err := utils.CreateDirectory(dbItems.Video.TmpVideoHlsPath); err != nil {
				return err
			}

			// convert to hls
			err = exec.ConvertVideoToHLS(ctx, dbItems.Video)
			if err != nil {
				return err
			}
		}

		// delete source video
		if utils.FileExists(dbItems.Video.TmpVideoDownloadPath) {
			err = utils.DeleteFile(dbItems.Video.TmpVideoDownloadPath)
			if err != nil {
				return err
			}
		}
	}

	next := []transactionalJob{}
	if job.Args.Continue {
		next = append(next, transactionalJob{Args: &MoveVideoArgs{Continue: true, Input: nextArchiveInput(job.Args.Input)}})
	}
	err = setQueueStatusAndEnqueue(ctx, store, QueueStatusInput{
		Status:  utils.Success,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskPostProcessVideo,
	}, next...)
	if err != nil {
		return err
	}

	// check if tasks are done
	if err := checkIfTasksAreDone(ctx, store.Client, job.Args.Input); err != nil {
		return err
	}

	return nil
}

// /////////////
// Move Video //
// /////////////
type MoveVideoArgs struct {
	Continue bool              `json:"continue"`
	Input    ArchiveVideoInput `json:"input"`
}

func (MoveVideoArgs) Kind() string { return string(utils.TaskMoveVideo) }

func (args MoveVideoArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
		Queue:       "default",
		Tags:        []string{"archive"},
		UniqueOpts:  archiveUniqueOpts(),
	}
}

func (w *MoveVideoWorker) Timeout(job *river.Job[MoveVideoArgs]) time.Duration {
	return 24 * time.Hour
}

type MoveVideoWorker struct {
	river.WorkerDefaults[MoveVideoArgs]
}

func (w MoveVideoWorker) Work(ctx context.Context, job *river.Job[MoveVideoArgs]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()

	// get store from context
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	// set queue status to running
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Running,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskMoveVideo,
	})
	if err != nil {
		return err
	}

	dbItems, err := getDatabaseItems(ctx, store.Client, job.Args.Input.QueueId)
	if err != nil {
		return err
	}

	// move standard video
	if dbItems.Video.VideoHlsPath == "" {
		// Standard (non-HLS) video move source is the finalized converted MP4.
		// For live archives a missing export means an earlier attempt already
		// published it; the playlist must never move as the video.
		tmpVideoPath := dbItems.Video.TmpVideoConvertPath
		if utils.FileExists(tmpVideoPath) {
			if err := validateNonEmptyFile(tmpVideoPath, "video move source"); err != nil {
				return err
			}
			if err := utils.MoveFile(ctx, tmpVideoPath, dbItems.Video.VideoPath); err != nil {
				return err
			}
		} else if dbItems.Queue.LiveArchive {
			if err := validateNonEmptyFile(dbItems.Video.VideoPath, "video move destination"); err != nil {
				return err
			}
		} else {
			// Fallback to download path for backwards compatibility if conversion output is missing.
			tmpVideoPath = dbItems.Video.TmpVideoDownloadPath
			if err := validateNonEmptyFile(tmpVideoPath, "video move source"); err != nil {
				return err
			}
			if err := utils.MoveFile(ctx, tmpVideoPath, dbItems.Video.VideoPath); err != nil {
				return err
			}
		}

		// delete temp hls directory if exists for watching while live
		if utils.DirectoryExists(dbItems.Video.TmpVideoHlsPath) {
			err = utils.DeleteDirectory(dbItems.Video.TmpVideoHlsPath)
			if err != nil {
				return err
			}
		}

	} else {
		// Finalize the playlist before the whole directory is moved below.
		if _, err := exec.EnsureLiveHlsPlaylist(ctx, dbItems.Video.TmpVideoHlsPath, liveCaptureID(&dbItems.Video)); err != nil {
			return err
		}

		// move hls video
		err = utils.MoveDirectory(ctx, dbItems.Video.TmpVideoHlsPath, dbItems.Video.VideoHlsPath)
		if err != nil {
			return err
		}

		// clean up temp hls directory
		if err := utils.DeleteDirectory(dbItems.Video.TmpVideoHlsPath); err != nil {
			return err
		}
		// delete temp converted video when present (unused for HLS-final).
		if utils.FileExists(dbItems.Video.TmpVideoConvertPath) {
			err = utils.DeleteFile(dbItems.Video.TmpVideoConvertPath)
			if err != nil {
				return err
			}
		}
	}

	next := []transactionalJob{}
	if dbItems.Video.Type == utils.Live {
		logger.Debug().Msg("queueing task to regenerate static thumbnail")
		next = append(next, transactionalJob{Args: GenerateStaticThumbnailArgs{VideoId: dbItems.Video.ID.String()}})
	}
	if !dbItems.Video.SpriteThumbnailsEnabled && config.Get().Archive.GenerateSpriteThumbnails {
		logger.Debug().Msg("queueing task to generate sprite thumbnails")
		next = append(next, transactionalJob{Args: GenerateSpriteThumbnailArgs{VideoId: dbItems.Video.ID.String()}})
	}
	err = setQueueStatusAndEnqueue(ctx, store, QueueStatusInput{
		Status:  utils.Success,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskMoveVideo,
	}, next...)
	if err != nil {
		return err
	}

	// check if tasks are done
	if err := checkIfTasksAreDone(ctx, store.Client, job.Args.Input); err != nil {
		return err
	}

	return nil
}
