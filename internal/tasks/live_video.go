package tasks

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

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
		UniqueOpts:  archiveUniqueOpts(),
	}
}

func (w *DownloadLiveVideoWorker) Timeout(job *river.Job[DownloadLiveVideoArgs]) time.Duration {
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

	dbItems, err := getDatabaseItems(ctx, store.Client, job.Args.Input.QueueId)
	if err != nil {
		return err
	}

	// Buffered so the ffmpeg-start signal in exec never blocks this worker if
	// cancellation lands first and the receiver below has already exited.
	startChatDownload := make(chan bool, 1)

	go func(workCtx context.Context) {
		for {
			select {
			case <-startChatDownload:
				// start chat download if requested
				if dbItems.Queue.ArchiveChat {
					log.Debug().Str("channel", dbItems.Channel.Name).Msgf("starting chat download for %s", dbItems.Video.ExtID)
					client := river.ClientFromContext[pgx.Tx](workCtx)
					_, insertErr := client.Insert(workCtx, &DownloadLiveChatArgs{
						Continue: true,
						Input:    nextArchiveInput(job.Args.Input),
					}, nil)
					if insertErr != nil {
						log.Error().Err(insertErr).Msg("failed to start chat download")
					}
				}
			case <-workCtx.Done():
				return
			}
		}
	}(ctx)

	// download live video
	// Note: even when download fails unexpectedly, continue with finalization steps
	// (cancel live chat, mark channel not live, enqueue post-process) so partial archive
	// can still be completed/moved instead of being left in a stuck state.
	downloadErr := exec.DownloadTwitchLiveVideo(ctx, dbItems.Video, dbItems.Channel, startChatDownload)
	logLiveCaptureOutcome(dbItems, job.Args.Input.QueueId, downloadErr)
	remotelyCancelled := false
	if downloadErr != nil {
		if errors.Is(downloadErr, context.Canceled) {
			if !errors.Is(context.Cause(ctx), rivertype.ErrJobCancelledRemotely) {
				// Process shutdown is recovered from the partial media by the
				// watchdog after restart; don't hide it as a successful job.
				return downloadErr
			}
			remotelyCancelled = true
			finalizeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), liveArchiveFinalizationTimeout)
			defer cancel()
			ctx = finalizeCtx
		} else {
			finalizeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), liveArchiveFinalizationTimeout)
			defer cancel()
			ctx = finalizeCtx
			log.Error().Err(downloadErr).Str("queue_id", job.Args.Input.QueueId.String()).Msg("live video download failed; continuing with archive finalization")
		}
	}

	// cancel chat download when video download is done
	// get chat download job id
	params := river.NewJobListParams().States(
		rivertype.JobStateAvailable,
		rivertype.JobStatePending,
		rivertype.JobStateScheduled,
		rivertype.JobStateRunning,
		rivertype.JobStateRetryable,
	).First(500)
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

	next := []transactionalJob{}
	if job.Args.Continue {
		next = append(next,
			transactionalJob{Args: &PostProcessVideoArgs{Continue: true, Input: nextArchiveInput(job.Args.Input)}},
			transactionalJob{
				Args: &UpdateStreamVideoIdArgs{Input: nextArchiveInput(job.Args.Input)},
				Opts: &river.InsertOpts{ScheduledAt: time.Now().Add(10 * time.Minute)},
			},
		)
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
	if remotelyCancelled {
		return downloadErr
	}

	return nil
}

// logLiveCaptureOutcome logs playlist size and segment count at capture stop.
func logLiveCaptureOutcome(dbItems *GetDatabaseItemsResponse, queueID uuid.UUID, downloadErr error) {
	var playlistBytes int64 = -1
	segments := 0
	if dbItems.Video.TmpVideoHlsPath != "" {
		if info, err := os.Stat(liveHlsPlaylistPath(&dbItems.Video)); err == nil {
			playlistBytes = info.Size()
		}
		for _, pattern := range []string{"*_segment*.m4s", "*_segment*.ts"} {
			if matches, err := filepath.Glob(filepath.Join(dbItems.Video.TmpVideoHlsPath, pattern)); err == nil {
				for _, match := range matches {
					if info, err := os.Stat(match); err == nil && info.Size() > 0 {
						segments++
					}
				}
			}
		}
	}

	logger := log.With().
		Str("queue_id", queueID.String()).
		Str("video_id", dbItems.Video.ID.String()).
		Int64("playlist_bytes", playlistBytes).
		Int("segments", segments).
		Logger()
	switch {
	case downloadErr == nil:
		logger.Info().Msg("live video download ended; capture output on disk")
	case errors.Is(downloadErr, context.Canceled):
		logger.Info().Err(downloadErr).Msg("live video download stopped; capture output on disk")
	default:
		logger.Error().Err(downloadErr).Msg("live video download failed; capture output on disk")
	}
}
