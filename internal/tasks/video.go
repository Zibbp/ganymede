package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/rs/zerolog/log"
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
	}
}

func (w DownloadVideoArgs) Timeout(job *river.Job[DownloadVideoArgs]) time.Duration {
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

	// start task heartbeat
	go startHeartBeatForTask(ctx, HeartBeatInput{
		TaskId: job.ID,
		conn:   store.ConnPool,
	})

	dbItems, err := getDatabaseItems(ctx, store.Client, job.Args.Input.QueueId)
	if err != nil {
		return err
	}

	// download video
	err = exec.DownloadTwitchVideo(ctx, dbItems.Video)
	if err != nil {
		return err
	}

	// set queue status to completed
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Success,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskDownloadVideo,
	})
	if err != nil {
		return err
	}

	// continue with next job
	if job.Args.Continue {
		client := river.ClientFromContext[pgx.Tx](ctx)
		_, err = client.Insert(ctx, &PostProcessVideoArgs{
			Continue: true,
			Input:    job.Args.Input,
		}, nil)
		if err != nil {
			return err
		}
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
	}
}

func (w *PostProcessVideoArgs) Timeout(job *river.Job[PostProcessVideoArgs]) time.Duration {
	return 24 * time.Hour
}

type PostProcessVideoWorker struct {
	river.WorkerDefaults[PostProcessVideoArgs]
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

	// start task heartbeat
	go startHeartBeatForTask(ctx, HeartBeatInput{
		TaskId: job.ID,
		conn:   store.ConnPool,
	})

	dbItems, err := getDatabaseItems(ctx, store.Client, job.Args.Input.QueueId)
	if err != nil {
		return err
	}

	// download video
	err = exec.PostProcessVideo(ctx, dbItems.Video)
	if err != nil {
		return err
	}

	// update video duration for live archive
	if dbItems.Queue.LiveArchive {
		duration, err := exec.GetVideoDuration(ctx, dbItems.Video.TmpVideoConvertPath)
		if err != nil {
			return err
		}
		_, err = dbItems.Video.Update().SetDuration(duration).Save(ctx)
		if err != nil {
			return err
		}

		// Update last chapter end time for live stream archive
		videoChapters, err := dbItems.Video.QueryChapters().All(ctx)
		if err != nil {
			return err
		}
		fmt.Println(videoChapters)

		if len(videoChapters) > 0 {
			for _, chapter := range videoChapters {
				fmt.Println(chapter)
				if chapter.End == 0 {
					fmt.Println("updating chapter end time")
					_, err = chapter.Update().SetEnd(duration).Save(ctx)
					if err != nil {
						return err
					}
				}
			}
		}
	}

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

	// set queue status to completed
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Success,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskPostProcessVideo,
	})
	if err != nil {
		return err
	}

	// continue with next job
	if job.Args.Continue {
		client := river.ClientFromContext[pgx.Tx](ctx)
		_, err = client.Insert(ctx, &MoveVideoArgs{
			Continue: true,
			Input:    job.Args.Input,
		}, nil)
		if err != nil {
			return err
		}
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
	}
}

func (w *MoveVideoArgs) Timeout(job *river.Job[MoveVideoArgs]) time.Duration {
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

	// start task heartbeat
	go startHeartBeatForTask(ctx, HeartBeatInput{
		TaskId: job.ID,
		conn:   store.ConnPool,
	})

	dbItems, err := getDatabaseItems(ctx, store.Client, job.Args.Input.QueueId)
	if err != nil {
		return err
	}

	// move standard video
	if dbItems.Video.VideoHlsPath == "" {
		err := utils.MoveFile(ctx, dbItems.Video.TmpVideoConvertPath, dbItems.Video.VideoPath)
		if err != nil {
			return err
		}
	} else {
		// move hls video
		err := utils.MoveDirectory(ctx, dbItems.Video.TmpVideoHlsPath, dbItems.Video.VideoHlsPath)
		if err != nil {
			return err
		}

		// clean up temp hls directory
		if err := utils.DeleteDirectory(dbItems.Video.TmpVideoHlsPath); err != nil {
			return err
		}
		// delete temp converted video
		if utils.FileExists(dbItems.Video.TmpVideoConvertPath) {
			err = utils.DeleteFile(dbItems.Video.TmpVideoConvertPath)
			if err != nil {
				return err
			}
		}
	}

	// set queue status to completed
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Success,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskMoveVideo,
	})
	if err != nil {
		return err
	}

	// check if tasks are done
	if err := checkIfTasksAreDone(ctx, store.Client, job.Args.Input); err != nil {
		return err
	}

	// Queue extra tasks that are not critical to the archive process

	// Queue task to regenerate thumbnail if livestream
	client := river.ClientFromContext[pgx.Tx](ctx)
	if dbItems.Video.Type == utils.Live {
		logger.Debug().Msg("queueing task to regenerate static thumbnail")
		_, err = client.Insert(ctx, GenerateStaticThumbnailArgs{
			VideoId: dbItems.Video.ID.String(),
		}, nil)
		if err != nil {
			return err
		}
	}

	// Queue task to generate sprite thumbnails if enabled
	if !dbItems.Video.SpriteThumbnailsEnabled && config.Get().Archive.GenerateSpriteThumbnails {
		logger.Debug().Msg("queueing task to generate sprite thumbnails")
		_, err = client.Insert(ctx, GenerateSpriteThumbnailArgs{
			VideoId: dbItems.Video.ID.String(),
		}, nil)
		if err != nil {
			return err
		}
	}

	return nil
}
