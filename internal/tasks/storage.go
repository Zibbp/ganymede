package tasks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/notification"
	"github.com/zibbp/ganymede/internal/storage"
	tasks_shared "github.com/zibbp/ganymede/internal/tasks/shared"
	"github.com/zibbp/ganymede/internal/utils"
)

// /////////////////
// Storage watchdog //
// //////////////////

const TaskStorageWatchdog = "storage_watchdog"

type StorageWatchdogArgs struct{}

func (StorageWatchdogArgs) Kind() string { return TaskStorageWatchdog }

func (w StorageWatchdogArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 1,
		Queue:       "default",
	}
}

func (w *StorageWatchdogWorker) Timeout(job *river.Job[StorageWatchdogArgs]) time.Duration {
	return 1 * time.Minute
}

// StorageWatchdogWorker monitors temp-directory free space. It notifies once
// per state crossing (not on every run) and stops running live captures at the
// critical threshold so partial media is finalized instead of corrupted by a
// full disk.
type StorageWatchdogWorker struct {
	river.WorkerDefaults[StorageWatchdogArgs]

	mu       sync.Mutex
	lastSeen storage.State
}

func storageCheckerFromContext(ctx context.Context) (storage.Checker, error) {
	checker, exists := ctx.Value(tasks_shared.StorageCheckerKey).(storage.Checker)
	if !exists || checker == nil {
		return nil, fmt.Errorf("storage checker not found in context")
	}
	return checker, nil
}

func (w *StorageWatchdogWorker) Work(ctx context.Context, job *river.Job[StorageWatchdogArgs]) error {
	logger := log.With().Str("task", job.Kind).Logger()

	checker, err := storageCheckerFromContext(ctx)
	if err != nil {
		return err
	}
	status, _ := checker.Check(ctx)

	w.mu.Lock()
	previous := w.lastSeen
	w.lastSeen = status.State
	w.mu.Unlock()

	if status.State == previous {
		return nil
	}

	notify, stopCaptures := storageWatchdogAction(previous, status.State)
	if !notify {
		return nil
	}

	freeGiB := float64(status.FreeBytes) / (1024 * 1024 * 1024)
	switch status.State {
	case storage.StateWarning:
		logger.Warn().Int64("free_bytes", status.FreeBytes).Msg("temp directory low on space; new archives are paused")
		w.sendWarning(ctx, fmt.Sprintf("temp directory is low on space (%.1fGiB free); new archives are paused until space is available", freeGiB), status)
	case storage.StateCritical:
		logger.Error().Int64("free_bytes", status.FreeBytes).Msg("temp directory critically low on space; stopping live captures and pausing new archives")
		w.sendWarning(ctx, fmt.Sprintf("temp directory is critically low on space (%.1fGiB free); live captures are being stopped and new archives are paused", freeGiB), status)
	case storage.StateOK:
		logger.Info().Msg("temp directory storage recovered; new archives can start")
		w.sendWarning(ctx, "temp directory storage has recovered; new archives can start", status)
	}

	if stopCaptures {
		if err := w.stopRunningLiveCaptures(ctx, logger); err != nil {
			// Live finalization races this cancel by design; a later pass
			// retries while the state remains critical.
			logger.Error().Err(err).Msg("error stopping live captures for critical storage state")
		}
	}

	return nil
}

// storageWatchdogAction decides what a state crossing should do. Crossings to
// warning or critical notify once and stop live captures only at critical;
// returning to OK from a paused state notifies recovery.
func storageWatchdogAction(previous, current storage.State) (notify bool, stopCaptures bool) {
	switch current {
	case storage.StateWarning:
		return true, false
	case storage.StateCritical:
		return true, true
	case storage.StateOK:
		return previous == storage.StateWarning || previous == storage.StateCritical, false
	}
	return false, false
}

// sendWarning dispatches through the generic system-warning trigger. It runs
// detached so a slow webhook cannot consume the watchdog job's time budget.
func (w *StorageWatchdogWorker) sendWarning(ctx context.Context, message string, status storage.Status) {
	notifSvc, err := NotificationServiceFromContext(ctx)
	if err != nil {
		return
	}
	go func() {
		warnCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).Msg("panic in notification")
			}
		}()
		notifSvc.SendSystemWarning(warnCtx, notification.SystemWarning{
			Message: message,
			Variables: map[string]interface{}{
				"storage_state":              string(status.State),
				"temp_directory_free_space":  fmt.Sprintf("%.1fGiB", float64(status.FreeBytes)/(1024*1024*1024)),
				"temp_directory_total_space": fmt.Sprintf("%.1fGiB", float64(status.TotalBytes)/(1024*1024*1024)),
			},
		})
	}()
}

// stopRunningLiveCaptures cancels running live video download jobs. The live
// workers observe the remote cancellation, flush partial media, and enqueue
// finalization themselves.
func (w *StorageWatchdogWorker) stopRunningLiveCaptures(ctx context.Context, logger zerolog.Logger) error {
	riverClient := river.ClientFromContext[pgx.Tx](ctx)
	params := river.NewJobListParams().States(rivertype.JobStateRunning).
		Kinds(string(utils.TaskDownloadLiveVideo)).First(500)
	stopped := 0
	if err := forEachJobPage(ctx, riverClient, params, func(job *rivertype.JobRow) error {
		if !utils.Contains(job.Tags, archive_tag) {
			return nil
		}
		if _, err := riverClient.JobCancel(ctx, job.ID); err != nil {
			return fmt.Errorf("cancel live capture job %d: %w", job.ID, err)
		}
		stopped++
		logger.Info().Int64("job_id", job.ID).Msg("stopping live capture due to critical temp storage")
		return nil
	}); err != nil {
		return err
	}
	if stopped > 0 {
		logger.Info().Int("stopped", stopped).Msg("stopped live captures due to critical temp storage")
	}
	return nil
}
