package tasks_periodic

import (
	"context"
	"fmt"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/storage"
	"github.com/zibbp/ganymede/internal/tasks"
)

// Reconcile the videos directory with the database and store what is left over.
type ReconcileStorageArgs struct{}

func (ReconcileStorageArgs) Kind() string { return tasks.TaskReconcileStorage }

func (ReconcileStorageArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 1,
		// One reconciliation at a time, two would only write the same findings. The states are
		// listed explicitly because the default set includes completed, which would keep a
		// second scan from running until the job cleaner removes the first one.
		UniqueOpts: river.UniqueOpts{
			ByArgs: true,
			ByState: []rivertype.JobState{
				rivertype.JobStateAvailable,
				rivertype.JobStatePending,
				rivertype.JobStateRetryable,
				rivertype.JobStateRunning,
				rivertype.JobStateScheduled,
			},
		},
	}
}

type ReconcileStorageWorker struct {
	river.WorkerDefaults[ReconcileStorageArgs]
}

func (w *ReconcileStorageWorker) Timeout(job *river.Job[ReconcileStorageArgs]) time.Duration {
	return 1 * time.Hour
}

func (w ReconcileStorageWorker) Work(ctx context.Context, job *river.Job[ReconcileStorageArgs]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()
	logger.Info().Msg("starting task")

	store, err := tasks.StoreFromContext(ctx)
	if err != nil {
		return err
	}

	count, err := storage.ReconcileAndStore(ctx, store)
	if err != nil {
		return err
	}

	logger.Info().Msgf("task completed, %d directories are not tied to a video", count)
	return nil
}
