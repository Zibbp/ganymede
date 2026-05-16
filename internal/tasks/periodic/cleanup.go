package tasks_periodic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/riverqueue/river"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/tasks"
)

// PruneLogFilesArgs defines the arguments for the log file pruning task
type PruneLogFilesArgs struct{}

func (PruneLogFilesArgs) Kind() string { return tasks.TaskPruneLogFiles }

func (w PruneLogFilesArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
	}
}

func (w PruneLogFilesArgs) Timeout(job *river.Job[PruneLogFilesArgs]) time.Duration {
	return 10 * time.Minute
}

type PruneLogFilesWorker struct {
	river.WorkerDefaults[PruneLogFilesArgs]
}

func (w PruneLogFilesWorker) Work(ctx context.Context, job *river.Job[PruneLogFilesArgs]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()
	logger.Info().Msg("starting task")

	logsDirectory := config.GetEnvConfig().LogsDir
	logRetentionDays := config.Get().LogRetentionDays
	cutoff := time.Now()
	if logRetentionDays <= 0 {
		logger.Warn().
			Int("retention_days", logRetentionDays).
			Time("cutoff", cutoff).
			Msg("skipping log file pruning due to non-positive retention days")
		return nil
	}

	var (
		totalFiles     int
		totalDeleted   int
		totalDeleteErr int
		totalInfoErr   int
	)

	cutoff = cutoff.AddDate(0, 0, -logRetentionDays)

	logFiles, err := os.ReadDir(logsDirectory)
	if err != nil {
		logger.Error().Err(err).Str("logs_directory", logsDirectory).Msg("failed to read logs directory")
		return err
	}

	for _, logFile := range logFiles {
		if logFile.IsDir() {
			continue
		}
		totalFiles++

		info, err := logFile.Info()
		if err != nil {
			totalInfoErr++
			logger.Error().Err(err).Str("file_name", logFile.Name()).Msg("failed to get log file info")
			continue
		}

		if info.ModTime().Before(cutoff) {
			filePath := filepath.Join(logsDirectory, logFile.Name())
			err := os.Remove(filePath)
			if err != nil {
				totalDeleteErr++
				logger.Error().Err(err).Str("file_path", filePath).Msg("failed to delete log file")
			} else {
				totalDeleted++
				logger.Info().Str("file_path", filePath).Msg("deleted old log file")
			}
		}
	}

	logger.Info().
		Int("retention_days", logRetentionDays).
		Int("files_deleted", totalDeleted).
		Msg("task completed")

	return nil
}
