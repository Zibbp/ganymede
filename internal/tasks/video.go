package tasks

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/spf13/viper"
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
		client.Insert(ctx, &PostProcessVideoArgs{
			Continue: true,
			Input:    job.Args.Input,
		}, nil)
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
	}

	// convert to HLS if needed
	if viper.GetBool("archive.save_as_hls") {
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
		client.Insert(ctx, &MoveVideoArgs{
			Continue: true,
			Input:    job.Args.Input,
		}, nil)
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

	return nil
}
