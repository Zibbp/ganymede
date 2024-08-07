package tasks

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/exec"
	"github.com/zibbp/ganymede/internal/utils"
)

// //////////////////////
// Download Live Video //
// //////////////////////
// This task is special as it will create it's own context if the task is cancelled so the rest of the task can be completed.
type DownloadLiveVideoArgs struct {
	Continue bool              `json:"continue"`
	Input    ArchiveVideoInput `json:"input"`
}

func (DownloadLiveVideoArgs) Kind() string { return string(utils.TaskDownloadLiveVideo) }

func (args DownloadLiveVideoArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 1,
		Tags:        []string{"archive"},
	}
}

func (w DownloadLiveVideoArgs) Timeout(job *river.Job[DownloadLiveVideoArgs]) time.Duration {
	return 49 * time.Hour
}

type DownloadLiveVideoWorker struct {
	river.WorkerDefaults[DownloadLiveVideoArgs]
}

func (w DownloadLiveVideoWorker) Work(ctx context.Context, job *river.Job[DownloadLiveVideoArgs]) error {
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
	client := river.ClientFromContext[pgx.Tx](ctx)

	// start task heartbeat
	go startHeartBeatForTask(ctx, HeartBeatInput{
		TaskId: job.ID,
		conn:   store.ConnPool,
	})

	dbItems, err := getDatabaseItems(ctx, store.Client, job.Args.Input.QueueId)
	if err != nil {
		return err
	}

	startChatDownload := make(chan bool)

	go func() {
		for {
			select {
			case <-startChatDownload:
				// start chat download if requested
				if dbItems.Queue.ArchiveChat {
					log.Debug().Str("channel", dbItems.Channel.Name).Msgf("starting chat download for %s", dbItems.Video.ExtID)
					client := river.ClientFromContext[pgx.Tx](ctx)
					_, err = client.Insert(ctx, &DownloadLiveChatArgs{
						Continue: true,
						Input:    job.Args.Input,
					}, nil)
					if err != nil {
						log.Error().Err(err).Msg("failed to start chat download")
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// download live video
	err = exec.DownloadTwitchLiveVideo(ctx, dbItems.Video, dbItems.Channel, startChatDownload)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			// create new context to finish the task
			ctx = context.Background()
		} else {
			return err
		}
	}

	// cancel chat download when video download is done
	// get chat download job id
	params := river.NewJobListParams().States(rivertype.JobStateRunning, rivertype.JobStateRetryable).First(10000)
	chatDownloadJobId, err := getTaskId(ctx, client, GetTaskFilter{
		Kind:    string(utils.TaskDownloadLiveChat),
		QueueId: job.Args.Input.QueueId,
		Tags:    []string{"archive"},
	}, params)
	if err != nil {
		return err
	}
	// cancel chat download if it exists
	if chatDownloadJobId != 0 {
		_, err = client.JobCancel(ctx, chatDownloadJobId)
		if err != nil {
			return err
		}
	}

	// mark channel as not live
	if err := setWatchChannelAsNotLive(ctx, store, dbItems.Channel.ID); err != nil {
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
		_, err = client.Insert(ctx, &PostProcessVideoArgs{
			Continue: true,
			Input:    job.Args.Input,
		}, nil)
		if err != nil {
			return err
		}

		// insert task to update stream id with video id
		_, err := client.Insert(ctx, &UpdateStreamVideoIdArgs{
			Input: job.Args.Input,
		}, &river.InsertOpts{
			// schedule task to run after 10 minutes to ensure the video is processed by the platform
			ScheduledAt: time.Now().Add(10 * time.Minute),
		})
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
