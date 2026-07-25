package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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

const (
	archiveHeartbeatTimeout      = 90 * time.Second
	liveArchiveCancellationGrace = liveArchiveFinalizationTimeout + 30*time.Second
	liveArchiveMediaQuietPeriod  = 30 * time.Second
)

type staleLiveArchiveAction uint8

const (
	staleLiveArchiveActionCancel staleLiveArchiveAction = iota
	staleLiveArchiveActionWait
	staleLiveArchiveActionRecover
)

func (WatchdogArgs) Kind() string { return TaskArchiveWatchdog }

func (w WatchdogArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 1,
		Queue:       "default",
	}
}

func (w *WatchdogWorker) Timeout(job *river.Job[WatchdogArgs]) time.Duration {
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

// runWatchdog finds archive jobs whose public progress output has stopped
// advancing. It uses only River's public APIs: the stale execution is
// cancelled, and a replacement generation is inserted when retry budget
// remains. Historical jobs are retained for River's normal lifecycle cleanup.
func runWatchdog(ctx context.Context, riverClient *river.Client[pgx.Tx]) error {
	logger := log.With().Str("task", "watchdog").Logger()
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	params := river.NewJobListParams().States(rivertype.JobStateRunning).First(500)
	if err := forEachJobPage(ctx, riverClient, params, func(job *rivertype.JobRow) error {
		if !utils.Contains(job.Tags, archive_tag) {
			return nil
		}
		heartbeat, _, err := archiveJobProgress(job)
		if err != nil {
			return err
		}
		if heartbeat.IsZero() || time.Since(heartbeat) <= archiveHeartbeatTimeout {
			return nil
		}

		// Re-read immediately before cancellation so a heartbeat racing with
		// this watchdog pass cannot cause a healthy execution to be replaced.
		fresh, err := riverClient.JobGet(ctx, job.ID)
		if err != nil {
			if errors.Is(err, rivertype.ErrNotFound) {
				return nil
			}
			return err
		}
		freshHeartbeat, freshArgs, err := archiveJobProgress(fresh)
		if err != nil {
			return err
		}
		if fresh.State != rivertype.JobStateRunning || freshHeartbeat.IsZero() || time.Since(freshHeartbeat) <= archiveHeartbeatTimeout {
			return nil
		}

		needsRecovery, err := archiveQueueStageNeedsRecovery(ctx, store, freshArgs.Input.QueueId, fresh.Kind)
		if err != nil {
			return err
		}
		if !needsRecovery {
			cancelAttemptedAt, err := archiveJobCancellationAttemptedAt(fresh)
			if err != nil {
				return err
			}
			if cancelAttemptedAt.IsZero() {
				if _, err := riverClient.JobCancel(ctx, fresh.ID); err != nil {
					return err
				}
			}
			logger.Debug().
				Int64("job_id", fresh.ID).
				Str("kind", fresh.Kind).
				Str("queue_id", freshArgs.Input.QueueId.String()).
				Msg("archive queue stage does not need recovery; canceled retained River job")
			return nil
		}

		if isLiveArchiveDownload(fresh.Kind) {
			action, graceRemaining, err := staleLiveArchiveRecoveryAction(fresh, time.Now())
			if err != nil {
				return err
			}
			switch action {
			case staleLiveArchiveActionWait:
				logger.Info().
					Int64("job_id", fresh.ID).
					Str("kind", fresh.Kind).
					Dur("grace_remaining", graceRemaining).
					Msg("waiting for live archive worker to finish cancellation")
				return nil
			case staleLiveArchiveActionRecover:
				if fresh.Kind == string(utils.TaskDownloadLiveVideo) {
					quiet, err := liveArchiveVideoInputIsQuiet(ctx, store, freshArgs.Input.QueueId, time.Now())
					if err != nil {
						return err
					}
					if !quiet {
						logger.Warn().
							Int64("job_id", fresh.ID).
							Str("kind", fresh.Kind).
							Str("queue_id", freshArgs.Input.QueueId.String()).
							Msg("live archive cancellation grace elapsed but capture output is still changing; deferring recovery")
						return nil
					}
				}
				logger.Warn().
					Int64("job_id", fresh.ID).
					Str("kind", fresh.Kind).
					Str("queue_id", freshArgs.Input.QueueId.String()).
					Msg("live archive worker did not finish after cancellation grace; recovering partial archive")
				return recoverExhaustedArchiveJob(ctx, store, riverClient, fresh, freshArgs.Input.QueueId)
			}
		}

		logger.Warn().Int64("job_id", fresh.ID).Str("kind", fresh.Kind).Msg("archive job heartbeat timed out")
		if _, err := riverClient.JobCancel(ctx, fresh.ID); err != nil {
			return err
		}
		if isLiveArchiveDownload(fresh.Kind) {
			// Live workers flush and enqueue their own partial-media finalization
			// after observing River's remote-cancellation cause. Starting that
			// work here would race the still-running capture process and its files.
			// If the worker process is gone, a later watchdog pass will observe the
			// preserved cancellation timestamp and recover after the grace period.
			return nil
		}

		remainingAttempts := fresh.MaxAttempts - fresh.Attempt
		if remainingAttempts > 0 {
			replacement, err := newRecoveredArchiveArgs(fresh, freshArgs.Input.RecoveryGeneration+1, remainingAttempts)
			if err != nil {
				return err
			}
			result, err := riverClient.Insert(ctx, replacement, nil)
			if err != nil {
				return err
			}
			logger.Info().Int64("old_job_id", fresh.ID).Int64("replacement_job_id", result.Job.ID).Int("remaining_attempts", remainingAttempts).Msg("inserted replacement archive job")
			return nil
		}

		return recoverExhaustedArchiveJob(ctx, store, riverClient, fresh, freshArgs.Input.QueueId)
	}); err != nil {
		return err
	}

	if err := recoverOrphanedLiveVideoArchives(ctx, store, riverClient); err != nil {
		return err
	}
	if err := recoverOrphanedLiveChatArchives(ctx, store, riverClient); err != nil {
		return err
	}

	return nil
}

func isLiveArchiveDownload(kind string) bool {
	return kind == string(utils.TaskDownloadLiveVideo) || kind == string(utils.TaskDownloadLiveChat)
}

func archiveQueueStageNeedsRecovery(ctx context.Context, store *database.Database, queueID uuid.UUID, kind string) (bool, error) {
	queue, err := store.Client.Queue.Get(ctx, queueID)
	if err != nil {
		if ent.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return archiveQueueStageStatusNeedsRecovery(queue, kind), nil
}

func archiveQueueStageStatusNeedsRecovery(queue *ent.Queue, kind string) bool {
	if !queue.Processing {
		return false
	}

	var status utils.TaskStatus
	switch utils.GetTaskName(kind) {
	case utils.TaskCreateFolder:
		status = queue.TaskVodCreateFolder
	case utils.TaskDownloadThumbnail:
		status = queue.TaskVodDownloadThumbnail
	case utils.TaskSaveInfo:
		status = queue.TaskVodSaveInfo
	case utils.TaskDownloadVideo:
		status = queue.TaskVideoDownload
	case utils.TaskPostProcessVideo:
		status = queue.TaskVideoConvert
	case utils.TaskMoveVideo:
		status = queue.TaskVideoMove
	case utils.TaskDownloadChat:
		status = queue.TaskChatDownload
	case utils.TaskConvertChat:
		status = queue.TaskChatConvert
	case utils.TaskRenderChat:
		status = queue.TaskChatRender
	case utils.TaskMoveChat:
		status = queue.TaskChatMove
	default:
		return false
	}

	return status == utils.Running
}

func archiveJobCancellationAttemptedAt(job *rivertype.JobRow) (time.Time, error) {
	var metadata struct {
		CancelAttemptedAt time.Time `json:"cancel_attempted_at"`
	}
	if err := json.Unmarshal(job.Metadata, &metadata); err != nil {
		return time.Time{}, fmt.Errorf("decode archive job %d cancellation metadata: %w", job.ID, err)
	}
	return metadata.CancelAttemptedAt, nil
}

func staleLiveArchiveRecoveryAction(job *rivertype.JobRow, now time.Time) (staleLiveArchiveAction, time.Duration, error) {
	cancelAttemptedAt, err := archiveJobCancellationAttemptedAt(job)
	if err != nil {
		return staleLiveArchiveActionCancel, 0, err
	}
	if cancelAttemptedAt.IsZero() {
		return staleLiveArchiveActionCancel, 0, nil
	}

	graceRemaining := cancelAttemptedAt.Add(liveArchiveCancellationGrace).Sub(now)
	if graceRemaining > 0 {
		return staleLiveArchiveActionWait, graceRemaining, nil
	}
	return staleLiveArchiveActionRecover, 0, nil
}

func liveArchiveVideoInputIsQuiet(ctx context.Context, store *database.Database, queueID uuid.UUID, now time.Time) (bool, error) {
	dbItems, err := getDatabaseItems(ctx, store.Client, queueID)
	if err != nil {
		return false, err
	}

	path := recoverableLiveVideoInputPath(&dbItems.Video)
	return fileIsQuiet(path, now)
}

func fileIsQuiet(path string, now time.Time) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, fmt.Errorf("stat live archive recovery input %q: %w", path, err)
	}
	return now.Sub(info.ModTime()) >= liveArchiveMediaQuietPeriod, nil
}

func recoverExhaustedArchiveJob(ctx context.Context, store *database.Database, riverClient *river.Client[pgx.Tx], job *rivertype.JobRow, queueID uuid.UUID) error {
	switch job.Kind {
	case string(utils.TaskDownloadLiveVideo):
		return recoverInterruptedLiveVideoArchive(ctx, store, riverClient, queueID)
	case string(utils.TaskDownloadLiveChat):
		return recoverInterruptedLiveChatArchive(ctx, store, queueID)
	default:
		taskName := utils.GetTaskName(job.Kind)
		if taskName == "" {
			return nil
		}
		return setQueueStatus(ctx, store.Client, QueueStatusInput{Status: utils.Failed, QueueId: queueID, Task: taskName})
	}
}

func archiveJobProgress(job *rivertype.JobRow) (time.Time, RiverJobArgs, error) {
	var args RiverJobArgs
	if err := json.Unmarshal(job.EncodedArgs, &args); err != nil {
		return time.Time{}, args, fmt.Errorf("decode archive job %d args: %w", job.ID, err)
	}

	var metadata struct {
		Output ArchiveProgressOutput `json:"output"`
	}
	if err := json.Unmarshal(job.Metadata, &metadata); err == nil && !metadata.Output.HeartbeatAt.IsZero() {
		return metadata.Output.HeartbeatAt, args, nil
	}
	return args.Input.HeartBeatTime, args, nil
}

func forEachJobPage(ctx context.Context, client *river.Client[pgx.Tx], params *river.JobListParams, visit func(*rivertype.JobRow) error) error {
	for {
		result, err := client.JobList(ctx, params)
		if err != nil {
			return err
		}
		for _, job := range result.Jobs {
			if err := visit(job); err != nil {
				return err
			}
		}
		if len(result.Jobs) < 500 || result.LastCursor == nil {
			return nil
		}
		params = params.After(result.LastCursor)
	}
}

// recoveredArchiveArgs preserves the original encoded argument shape and kind
// while incrementing the recovery generation. River will decode it into the
// registered concrete args type when the replacement is worked.
type recoveredArchiveArgs struct {
	kind string
	raw  json.RawMessage
	opts river.InsertOpts
}

func (a *recoveredArchiveArgs) Kind() string                 { return a.kind }
func (a *recoveredArchiveArgs) InsertOpts() river.InsertOpts { return a.opts }
func (a *recoveredArchiveArgs) MarshalJSON() ([]byte, error) { return a.raw, nil }

func newRecoveredArchiveArgs(job *rivertype.JobRow, generation, remainingAttempts int) (*recoveredArchiveArgs, error) {
	var encoded map[string]any
	if err := json.Unmarshal(job.EncodedArgs, &encoded); err != nil {
		return nil, fmt.Errorf("decode archive job %d for recovery: %w", job.ID, err)
	}
	input, ok := encoded["input"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("archive job %d has no input object", job.ID)
	}
	input["recovery_generation"] = generation
	raw, err := json.Marshal(encoded)
	if err != nil {
		return nil, err
	}

	return &recoveredArchiveArgs{
		kind: job.Kind,
		raw:  raw,
		opts: river.InsertOpts{
			MaxAttempts: remainingAttempts,
			Priority:    job.Priority,
			Queue:       job.Queue,
			Tags:        append([]string(nil), job.Tags...),
			UniqueOpts:  archiveUniqueOpts(),
		},
	}, nil
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

func recoverOrphanedLiveChatArchives(ctx context.Context, store *database.Database, riverClient *river.Client[pgx.Tx]) error {
	activeLiveDownloads, err := activeArchiveJobQueues(ctx, riverClient, string(utils.TaskDownloadLiveChat))
	if err != nil {
		return err
	}

	stuckQueues, err := store.Client.Queue.Query().
		Where(
			entQueue.LiveArchive(true),
			entQueue.Processing(true),
			entQueue.ArchiveChat(true),
			entQueue.TaskChatDownloadEQ(utils.Running),
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
			Msg("detected orphaned live chat archive queue with no active download job; attempting recovery")
		if err := recoverInterruptedLiveChatArchive(ctx, store, q.ID); err != nil {
			return err
		}
	}

	return nil
}

func recoverInterruptedLiveChatArchive(ctx context.Context, store *database.Database, queueID uuid.UUID) error {
	dbItems, err := getDatabaseItems(ctx, store.Client, queueID)
	if err != nil {
		return err
	}
	if !liveArchiveDownloadNeedsRecovery(&dbItems.Queue, string(utils.TaskDownloadLiveChat)) {
		log.Debug().
			Str("queue_id", queueID.String()).
			Str("status", string(dbItems.Queue.TaskChatDownload)).
			Msg("live chat archive recovery already completed; skipping retained River job")
		return nil
	}
	if err := setWatchChannelAsNotLive(ctx, store, dbItems.Channel.ID); err != nil {
		return err
	}
	return setQueueStatusAndEnqueue(ctx, store,
		QueueStatusInput{Status: utils.Success, QueueId: queueID, Task: utils.TaskDownloadChat},
		transactionalJob{Args: &ConvertLiveChatArgs{Continue: true, Input: ArchiveVideoInput{QueueId: queueID}}},
	)
}

func activeArchiveJobQueues(ctx context.Context, riverClient *river.Client[pgx.Tx], kind string) (map[uuid.UUID]bool, error) {
	params := river.NewJobListParams().States(
		rivertype.JobStateAvailable,
		rivertype.JobStatePending,
		rivertype.JobStateScheduled,
		rivertype.JobStateRunning,
		rivertype.JobStateRetryable,
	).Kinds(kind).First(500)

	queues := make(map[uuid.UUID]bool)
	err := forEachJobPage(ctx, riverClient, params, func(job *rivertype.JobRow) error {
		if !utils.Contains(job.Tags, archive_tag) {
			return nil
		}

		var args RiverJobArgs
		if err := json.Unmarshal(job.EncodedArgs, &args); err != nil {
			return err
		}
		if args.Input.QueueId != uuid.Nil {
			queues[args.Input.QueueId] = true
		}
		return nil
	})
	return queues, err
}

func recoverInterruptedLiveVideoArchive(ctx context.Context, store *database.Database, riverClient *river.Client[pgx.Tx], queueID uuid.UUID) error {
	dbItems, err := getDatabaseItems(ctx, store.Client, queueID)
	if err != nil {
		return err
	}
	if !liveArchiveDownloadNeedsRecovery(&dbItems.Queue, string(utils.TaskDownloadLiveVideo)) {
		log.Debug().
			Str("queue_id", queueID.String()).
			Str("status", string(dbItems.Queue.TaskVideoDownload)).
			Msg("live video archive recovery already completed; skipping retained River job")
		return nil
	}

	if err := validateRecoverableLiveVideoInput(&dbItems.Video); err != nil {
		log.Error().
			Err(err).
			Str("queue_id", queueID.String()).
			Str("video_id", dbItems.Video.ID.String()).
			Msg("live video archive recovery skipped because captured media is missing or empty")
		if err := setWatchChannelAsNotLive(ctx, store, dbItems.Channel.ID); err != nil {
			return err
		}
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
	).First(500)
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

	if err := setQueueStatusAndEnqueue(ctx, store,
		QueueStatusInput{Status: utils.Success, QueueId: queueID, Task: utils.TaskDownloadVideo},
		transactionalJob{Args: &PostProcessVideoArgs{Continue: true, Input: ArchiveVideoInput{QueueId: queueID}}},
	); err != nil {
		return err
	}

	log.Info().
		Str("queue_id", queueID.String()).
		Str("video_id", dbItems.Video.ID.String()).
		Msg("live video archive recovery queued post-process")

	return nil
}

func validateRecoverableLiveVideoInput(video *ent.Vod) error {
	return validateNonEmptyFile(recoverableLiveVideoInputPath(video), "live video recovery input")
}

func recoverableLiveVideoInputPath(video *ent.Vod) string {
	if video.VideoHlsPath != "" {
		return fmt.Sprintf("%s/%s-video.m3u8", video.TmpVideoHlsPath, video.ExtID)
	}
	if video.TmpVideoConvertPath != "" && utils.FileExists(video.TmpVideoConvertPath) {
		return video.TmpVideoConvertPath
	}
	return video.TmpVideoDownloadPath
}

func liveArchiveDownloadNeedsRecovery(queue *ent.Queue, kind string) bool {
	switch kind {
	case string(utils.TaskDownloadLiveVideo):
		return queue.TaskVideoDownload == utils.Running
	case string(utils.TaskDownloadLiveChat):
		return queue.TaskChatDownload == utils.Running
	default:
		return false
	}
}
