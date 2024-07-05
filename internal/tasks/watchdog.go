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

func (WatchdogArgs) Kind() string { return "archive-watchdog" }

func (w WatchdogArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 1,
		Queue:       "default",
	}
}

func (w WatchdogArgs) Timeout(job *river.Job[WatchdogArgs]) time.Duration {
	return 45 * time.Second
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

// Watchdog tasks that checks the status of jobs every minutes. It checks if the job is still running and if it has timed out. If it has timed out, it sets the status of the job to retryable.
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
				}
			}
		}

	}

	return nil
}
