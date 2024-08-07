package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog/log"
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
							Task:    utils.TaskDownloadVideo,
						})
						if err != nil {
							return err
						}
						// queue video postprocess
						_, err = riverClient.Insert(ctx, &PostProcessVideoArgs{
							Continue: true,
							Input: ArchiveVideoInput{
								QueueId: args.Input.QueueId,
							},
						}, nil)
						if err != nil {
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

	return nil
}
