package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	entQueue "github.com/zibbp/ganymede/ent/queue"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/utils"
)

// ///////////
// Watchdog //
// //////////
type WatchdogArgs struct{}

func (WatchdogArgs) Kind() string { return TaskArchiveWatchdog }

func (w WatchdogArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 1,
		Queue:       "default",
	}
}

func (w WatchdogArgs) Timeout(job *river.Job[WatchdogArgs]) time.Duration {
	return 1 * time.Minute
}

type WatchdogWorker struct {
	river.WorkerDefaults[WatchdogArgs]
}

func (w WatchdogWorker) Work(ctx context.Context, job *river.Job[WatchdogArgs]) error {

	client := river.ClientFromContext[pgx.Tx](ctx)

	if err := runWatchdog(ctx, client); err != nil {
		return err
	}

	return nil
}

// Watchdog tasks that checks the status of archive jobs every minute. It checks if the job is still running and if it has timed out. If it has timed out, it sets the status of the job to retryable.
func runWatchdog(ctx context.Context, riverClient *river.Client[pgx.Tx]) error {
	logger := log.With().Str("task", "watchdog").Logger()
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}
	// get jobs
	params := river.NewJobListParams().States(rivertype.JobStateRunning).First(10000)
	jobs, err := riverClient.JobList(ctx, params)
	if err != nil {
		return err
	}

	logger.Debug().Str("jobs", fmt.Sprintf("%d", len(jobs.Jobs))).Msg("jobs found")

	// check jobs
	for _, job := range jobs.Jobs {
		// only check archive jobs
		if utils.Contains(job.Tags, "archive") {
			// unmarshal args
			var args RiverJobArgs

			if err := json.Unmarshal(job.EncodedArgs, &args); err != nil {
				return err
			}

			// check if job has timed out
			if !args.Input.HeartBeatTime.IsZero() && time.Since(args.Input.HeartBeatTime) > 90*time.Second {
				// job heartbeat timed out
				logger.Info().Str("job_id", fmt.Sprintf("%d", job.ID)).Msg("job heartbeat timed out")

				if job.Attempt < job.MaxAttempts {
					// set job to retryable
					err := forceJobRetry(ctx, store.ConnPool, job.ID)
					if err != nil {
						return err
					}
					logger.Info().Str("job_id", fmt.Sprintf("%d", job.ID)).Msg("job set to retryable")
				} else {
					// set job to failed
					_, err := riverClient.JobCancel(ctx, job.ID)
					if err != nil {
						return err
					}
					err = forceDeleteJob(ctx, store.ConnPool, job.ID)
					if err != nil {
						return err
					}
					logger.Info().Str("job_id", fmt.Sprintf("%d", job.ID)).Msg("job set to failed and deleted")

					// attempt to finish archiving live video
					// if job was live video download then proceed with next jobs
					if job.Kind == string(utils.TaskDownloadLiveVideo) {
						logger.Info().Str("job_id", fmt.Sprintf("%d", job.ID)).Msg("detected job was live video download; proceeding with next jobs")
						if err := recoverInterruptedLiveVideoArchive(ctx, store, riverClient, args.Input.QueueId); err != nil {
							return err
						}
					}

					// if job was chat download then proceed with next jobs
					if job.Kind == string(utils.TaskDownloadLiveChat) {
						logger.Info().Str("job_id", fmt.Sprintf("%d", job.ID)).Msg("detected job was live chat download; proceeding with next jobs")
						// get db items
						dbItems, err := getDatabaseItems(ctx, store.Client, args.Input.QueueId)
						if err != nil {
							return err
						}

						// mark channel as not live
						if err := setWatchChannelAsNotLive(ctx, store, dbItems.Channel.ID); err != nil {
							return err
						}

						// set queue status to completed
						err = setQueueStatus(ctx, store.Client, QueueStatusInput{
							Status:  utils.Success,
							QueueId: dbItems.Queue.ID,
							Task:    utils.TaskDownloadChat,
						})
						if err != nil {
							return err
						}
						// queue chat convert
						_, err = riverClient.Insert(ctx, &ConvertLiveChatArgs{
							Continue: true,
							Input: ArchiveVideoInput{
								QueueId: args.Input.QueueId,
							},
						}, nil)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}

	if err := recoverOrphanedLiveVideoArchives(ctx, store, riverClient); err != nil {
		return err
	}

	return nil
}

func recoverOrphanedLiveVideoArchives(ctx context.Context, store *database.Database, riverClient *river.Client[pgx.Tx]) error {
	activeLiveDownloads, err := activeArchiveJobQueues(ctx, riverClient, string(utils.TaskDownloadLiveVideo))
	if err != nil {
		return err
	}

	stuckQueues, err := store.Client.Queue.Query().
		Where(
			entQueue.LiveArchive(true),
			entQueue.Processing(true),
			entQueue.TaskVideoDownloadEQ(utils.Running),
		).
		All(ctx)
	if err != nil {
		return err
	}

	for _, q := range stuckQueues {
		if activeLiveDownloads[q.ID] {
			continue
		}

		log.Info().
			Str("queue_id", q.ID.String()).
			Msg("detected orphaned live video archive queue with no active download job; attempting recovery")
		if err := recoverInterruptedLiveVideoArchive(ctx, store, riverClient, q.ID); err != nil {
			return err
		}
	}

	return nil
}

func activeArchiveJobQueues(ctx context.Context, riverClient *river.Client[pgx.Tx], kind string) (map[uuid.UUID]bool, error) {
	params := river.NewJobListParams().States(
		rivertype.JobStateAvailable,
		rivertype.JobStatePending,
		rivertype.JobStateScheduled,
		rivertype.JobStateRunning,
		rivertype.JobStateRetryable,
	).First(10000)
	jobs, err := riverClient.JobList(ctx, params)
	if err != nil {
		return nil, err
	}

	queues := make(map[uuid.UUID]bool)
	for _, job := range jobs.Jobs {
		if job.Kind != kind || !utils.Contains(job.Tags, "archive") {
			continue
		}

		var args RiverJobArgs
		if err := json.Unmarshal(job.EncodedArgs, &args); err != nil {
			return nil, err
		}
		if args.Input.QueueId != uuid.Nil {
			queues[args.Input.QueueId] = true
		}
	}

	return queues, nil
}

func recoverInterruptedLiveVideoArchive(ctx context.Context, store *database.Database, riverClient *river.Client[pgx.Tx], queueID uuid.UUID) error {
	dbItems, err := getDatabaseItems(ctx, store.Client, queueID)
	if err != nil {
		return err
	}

	if err := validateRecoverableLiveVideoInput(&dbItems.Video); err != nil {
		log.Error().
			Err(err).
			Str("queue_id", queueID.String()).
			Str("video_id", dbItems.Video.ID.String()).
			Msg("live video archive recovery skipped because captured media is missing or empty")
		return setQueueStatus(ctx, store.Client, QueueStatusInput{
			Status:  utils.Failed,
			QueueId: queueID,
			Task:    utils.TaskDownloadVideo,
		})
	}

	if err := setWatchChannelAsNotLive(ctx, store, dbItems.Channel.ID); err != nil {
		return err
	}

	params := river.NewJobListParams().States(
		rivertype.JobStateAvailable,
		rivertype.JobStatePending,
		rivertype.JobStateScheduled,
		rivertype.JobStateRunning,
		rivertype.JobStateRetryable,
	).First(10000)
	postProcessJobID, err := getTaskId(ctx, riverClient, GetTaskFilter{
		Kind:    string(utils.TaskPostProcessVideo),
		QueueId: queueID,
		Tags:    []string{"archive"},
	}, params)
	if err != nil {
		return err
	}
	if postProcessJobID != 0 {
		log.Info().
			Str("queue_id", queueID.String()).
			Int64("job_id", postProcessJobID).
			Msg("live video archive recovery found existing post-process job")
		return setQueueStatus(ctx, store.Client, QueueStatusInput{
			Status:  utils.Success,
			QueueId: queueID,
			Task:    utils.TaskDownloadVideo,
		})
	}

	_, err = riverClient.Insert(ctx, &PostProcessVideoArgs{
		Continue: true,
		Input: ArchiveVideoInput{
			QueueId: queueID,
		},
	}, nil)
	if err != nil {
		return err
	}

	if err := setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Success,
		QueueId: queueID,
		Task:    utils.TaskDownloadVideo,
	}); err != nil {
		return err
	}

	log.Info().
		Str("queue_id", queueID.String()).
		Str("video_id", dbItems.Video.ID.String()).
		Msg("live video archive recovery queued post-process")

	return nil
}

func validateRecoverableLiveVideoInput(video *ent.Vod) error {
	if video.VideoHlsPath != "" {
		playlistPath := fmt.Sprintf("%s/%s-video.m3u8", video.TmpVideoHlsPath, video.ExtID)
		return validateNonEmptyFile(playlistPath, "live HLS playlist")
	}

	if video.TmpVideoConvertPath != "" && utils.FileExists(video.TmpVideoConvertPath) {
		return validateNonEmptyFile(video.TmpVideoConvertPath, "live converted video input")
	}

	return validateNonEmptyFile(video.TmpVideoDownloadPath, "live downloaded video input")
}
