package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	entChannel "github.com/zibbp/ganymede/ent/channel"
	entLive "github.com/zibbp/ganymede/ent/live"
	"github.com/zibbp/ganymede/ent/queue"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/notification"
	"github.com/zibbp/ganymede/internal/platform"

	tasks_shared "github.com/zibbp/ganymede/internal/tasks/shared"
	"github.com/zibbp/ganymede/internal/utils"
	vods_utility "github.com/zibbp/ganymede/internal/vod/utility"
)

var archive_tag = "archive"
var allow_fail_tag = "allow_fail"

// Live download workers switch to a detached context after River asks them to
// stop so they can flush partial media and atomically enqueue the next stage.
// The watchdog must allow this entire window to elapse before taking over.
const liveArchiveFinalizationTimeout = 2 * time.Minute

var (
	TaskUpdateStreamVideoId         = "update_stream_video_id"
	TaskGenerateStaticThumbnails    = "generate_static_thumbnails"
	TaskGenerateSpriteThumbnails    = "generate_sprite_thumbnails"
	TaskArchiveWatchdog             = "archive_watchdog"
	TaskCheckChannelsForLivestreams = "check_channels_for_livestreams"
	TaskCheckChannelsForNewVideos   = "check_channels_for_new_videos"
	TaskCheckChannelsForNewClips    = "check_channels_for_new_clips"
	TaskPruneVideos                 = "prune_videos"
	TaskImportVideos                = "import_videos"
	TaskAuthenticatePlatform        = "authenticate_platform"
	TaskFetchJWKS                   = "fetch_jwks"
	TaskSaveVideoChapters           = "save_video_chapters"
	TaskUpdateVideoStorageUsage     = "update_video_storage_usage"
	TaskUpdateChannelStorageUsage   = "update_channel_storage_usage"
	TaskProcessPlaylistVideoRules   = "process_playlist_video_rules"
	TaskUpdateTwitchChannels        = "update_twitch_channels"
	TaskPruneLogFiles               = "prune_log_files"
)

var (
	QueueVideoDownload            = "video-download"
	QueueVideoPostProcess         = "video-postprocess"
	QueueChatDownload             = "chat-download"
	QueueChatRender               = "chat-render"
	QueueGenerateThumbnailSprites = "generate-thumbnail-sprites"
)

type ArchiveVideoInput struct {
	QueueId            uuid.UUID `json:"queue_id" river:"unique"`
	RecoveryGeneration int       `json:"recovery_generation,omitempty" river:"unique"`
	HeartBeatTime      time.Time `json:"heartbeat_time,omitempty"` // legacy read-only field
}

func archiveUniqueOpts() river.UniqueOpts {
	return river.UniqueOpts{
		ByArgs: true,
		ByState: []rivertype.JobState{
			rivertype.JobStateAvailable,
			rivertype.JobStatePending,
			rivertype.JobStateRunning,
			rivertype.JobStateRetryable,
			rivertype.JobStateScheduled,
		},
	}
}

func nextArchiveInput(input ArchiveVideoInput) ArchiveVideoInput {
	return ArchiveVideoInput{QueueId: input.QueueId}
}

type GetDatabaseItemsResponse struct {
	Queue   ent.Queue
	Video   ent.Vod
	Channel ent.Channel
}

type QueueStatusInput struct {
	Status  utils.TaskStatus
	QueueId uuid.UUID
	Task    utils.TaskName
}

type transactionalJob struct {
	Args river.JobArgs
	Opts *river.InsertOpts
}

func StoreFromContext(ctx context.Context) (*database.Database, error) {
	store, exists := ctx.Value(tasks_shared.StoreKey).(*database.Database)
	if !exists || store == nil {
		return nil, errors.New("store not found in context")
	}

	return store, nil
}

func PlatformFromContext(ctx context.Context) (platform.Platform, error) {
	platform, exists := ctx.Value(tasks_shared.PlatformTwitchKey).(platform.Platform)
	if !exists || platform == nil {
		log.Error().Msg("platform not found in context, this usually means the platform authentication failed, check your platform client_id and client_secret.")
		return nil, errors.New("platform not found in context")
	}

	return platform, nil
}

func NotificationServiceFromContext(ctx context.Context) (*notification.Service, error) {
	svc, exists := ctx.Value(tasks_shared.NotificationServiceKey).(*notification.Service)
	if !exists || svc == nil {
		return nil, errors.New("notification service not found in context")
	}
	return svc, nil
}

func EnqueuerFromContext(ctx context.Context) (tasks_shared.Enqueuer, error) {
	enqueuer, exists := ctx.Value(tasks_shared.EnqueuerKey).(tasks_shared.Enqueuer)
	if !exists || enqueuer == nil {
		return nil, errors.New("transactional River enqueuer not found in context")
	}
	return enqueuer, nil
}

// getDatabaseItems retrieves the database items associated with the provided queueId. This is used instead of passing all the structs to each job so that they can be easily updated in the database.
func getDatabaseItems(ctx context.Context, entClient *ent.Client, queueId uuid.UUID) (*GetDatabaseItemsResponse, error) {
	queue, err := entClient.Queue.Query().Where(queue.ID(queueId)).WithVod().Only(ctx)
	if err != nil {
		return nil, err
	}

	qC := queue.Edges.Vod.QueryChannel()
	channel, err := qC.Only(ctx)
	if err != nil {
		return nil, err
	}

	return &GetDatabaseItemsResponse{
		Queue:   *queue,
		Video:   *queue.Edges.Vod,
		Channel: *channel,
	}, nil

}

// setQueueStatus updates the status of a queue item in the database based on the provided queueStatusInput.
func setQueueStatus(ctx context.Context, entClient *ent.Client, queueStatusInput QueueStatusInput) error {

	q := entClient.Queue.UpdateOneID(queueStatusInput.QueueId)

	switch queueStatusInput.Task {
	case utils.TaskCreateFolder:
		q = q.SetTaskVodCreateFolder(queueStatusInput.Status)
	case utils.TaskDownloadThumbnail:
		q = q.SetTaskVodDownloadThumbnail(queueStatusInput.Status)
	case utils.TaskSaveInfo:
		q = q.SetTaskVodSaveInfo(queueStatusInput.Status)
	case utils.TaskDownloadVideo:
		q = q.SetTaskVideoDownload(queueStatusInput.Status)
	case utils.TaskPostProcessVideo:
		q = q.SetTaskVideoConvert(queueStatusInput.Status)
	case utils.TaskMoveVideo:
		q = q.SetTaskVideoMove(queueStatusInput.Status)
	case utils.TaskDownloadChat:
		q = q.SetTaskChatDownload(queueStatusInput.Status)
	case utils.TaskConvertChat:
		q = q.SetTaskChatConvert(queueStatusInput.Status)
	case utils.TaskRenderChat:
		q = q.SetTaskChatRender(queueStatusInput.Status)
	case utils.TaskMoveChat:
		q = q.SetTaskChatMove(queueStatusInput.Status)
	}

	_, err := q.Save(ctx)
	if err != nil {
		return err
	}

	return nil
}

// setQueueStatusAndEnqueue commits a stage transition and all of its successor
// jobs atomically. A crash can therefore leave either the old stage state or
// the complete handoff, but never a successful stage with no next job.
func setQueueStatusAndEnqueue(ctx context.Context, store *database.Database, status QueueStatusInput, jobs ...transactionalJob) error {
	enqueuer, err := EnqueuerFromContext(ctx)
	if err != nil {
		return err
	}
	return store.WithTx(ctx, func(txClient *ent.Client, tx *sql.Tx) error {
		if err := setQueueStatus(ctx, txClient, status); err != nil {
			return err
		}
		for _, next := range jobs {
			if _, err := enqueuer.InsertTx(ctx, tx, next.Args, next.Opts); err != nil {
				return err
			}
		}
		return nil
	})
}

// replaceThumbnailPlaceholders replaces the placeholders in the provided url with the provided width and height.
func replaceThumbnailPlaceholders(url, width, height string, isLive bool) string {
	if isLive {
		url = strings.ReplaceAll(url, "{width}", width)
		url = strings.ReplaceAll(url, "{height}", height)
	} else {
		url = strings.ReplaceAll(url, "%{width}", width)
		url = strings.ReplaceAll(url, "%{height}", height)
	}
	return url
}
func checkIfTasksAreDone(ctx context.Context, entClient *ent.Client, input ArchiveVideoInput) error {
	dbItems, err := getDatabaseItems(ctx, entClient, input.QueueId)
	if err != nil {
		return err
	}

	videoDone := dbItems.Queue.TaskVideoDownload == utils.Success && dbItems.Queue.TaskVideoConvert == utils.Success && dbItems.Queue.TaskVideoMove == utils.Success
	chatDone := dbItems.Queue.TaskChatDownload == utils.Success && dbItems.Queue.TaskChatRender == utils.Success && dbItems.Queue.TaskChatMove == utils.Success
	if dbItems.Queue.LiveArchive {
		chatDone = chatDone && dbItems.Queue.TaskChatConvert == utils.Success
	}
	if !videoDone || !chatDone || !dbItems.Queue.Processing {
		return nil
	}

	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}
	enqueuer, err := EnqueuerFromContext(ctx)
	if err != nil {
		return err
	}
	finalized := false
	if err := store.WithTx(ctx, func(txClient *ent.Client, tx *sql.Tx) error {
		if _, err := txClient.Queue.UpdateOneID(dbItems.Queue.ID).
			Where(queue.Processing(true)).
			SetVideoProcessing(false).
			SetChatProcessing(false).
			SetProcessing(false).
			Save(ctx); err != nil {
			if ent.IsNotFound(err) {
				return nil
			}
			return err
		}
		finalized = true
		if _, err := txClient.Vod.UpdateOneID(dbItems.Video.ID).SetProcessing(false).Save(ctx); err != nil {
			return err
		}
		_, err := enqueuer.InsertTx(ctx, tx, &UpdateVideoStorageUsage{VideoID: &dbItems.Video.ID}, nil)
		return err
	}); err != nil {
		return err
	}
	if !finalized {
		return nil
	}

	log.Debug().Msgf("all tasks for video %s are done", dbItems.Video.ID.String())
	if notifSvc, err := NotificationServiceFromContext(ctx); err == nil {
		go func() {
			notifCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
			defer cancel()
			defer func() {
				if r := recover(); r != nil {
					log.Error().Interface("panic", r).Msg("panic in notification")
				}
			}()
			if dbItems.Queue.LiveArchive {
				notifSvc.SendLiveArchiveSuccess(notifCtx, &dbItems.Channel, &dbItems.Video, &dbItems.Queue)
			} else {
				notifSvc.SendVideoArchiveSuccess(notifCtx, &dbItems.Channel, &dbItems.Video, &dbItems.Queue)
			}
		}()
	}

	return nil
}

type GetTaskFilter struct {
	Kind    string
	QueueId uuid.UUID
	Tags    []string
}

func getTaskId(ctx context.Context, client *river.Client[pgx.Tx], filter GetTaskFilter, params *river.JobListParams) (int64, error) {
	if filter.Kind != "" {
		params = params.Kinds(filter.Kind)
	}
	for {
		jobs, err := client.JobList(ctx, params)
		if err != nil {
			return 0, err
		}

		for _, job := range jobs.Jobs {
			var args RiverJobArgs
			if err := json.Unmarshal(job.EncodedArgs, &args); err != nil {
				return 0, err
			}

			if filter.Kind != "" && job.Kind != filter.Kind {
				continue
			}
			if filter.QueueId != uuid.Nil && args.Input.QueueId != filter.QueueId {
				continue
			}
			if len(filter.Tags) > 0 && !containsAllTags(job.Tags, filter.Tags) {
				continue
			}
			return job.ID, nil
		}
		if len(jobs.Jobs) < 500 || jobs.LastCursor == nil {
			return 0, nil
		}
		params = params.After(jobs.LastCursor)
	}
}

// Helper function to check if job tags contain all filter tags
func containsAllTags(jobTags, filterTags []string) bool {
	tagSet := make(map[string]struct{})
	for _, tag := range jobTags {
		tagSet[tag] = struct{}{}
	}

	for _, tag := range filterTags {
		if _, exists := tagSet[tag]; !exists {
			return false
		}
	}
	return true
}

// CustomErrorHandler implements river.ErrorHandler to handle errors and panics in jobs.
type CustomErrorHandler struct{}

func (*CustomErrorHandler) HandleError(ctx context.Context, job *rivertype.JobRow, err error) *river.ErrorHandlerResult {
	log.Error().Str("job_id", fmt.Sprintf("%d", job.ID)).Str("attempt", fmt.Sprintf("%d", job.Attempt)).Str("attempted_by", attemptedBy(job)).Str("args", string(job.EncodedArgs)).Err(err).Msg("task error")

	// Check if this is a phantom live stream and cleanup (GH#760)
	// This is behind an experimental flag
	if config.Get().Experimental.BetterLiveStreamDetectionAndCleanup {
		var e platform.ErrorNoStreamsFound
		if errors.As(err, &e) {
			// Job reported no stream found so we can clean up the live stream
			log.Warn().Msgf("phantom live stream detected for job %d, cleaning up", job.ID)

			// Unmarshal custom arguments
			var args RiverJobArgs
			if err := json.Unmarshal(job.EncodedArgs, &args); err != nil {
				log.Error().Err(err).Msg("failed to unmarshal job arguments")
				return nil
			}

			// Get store from context
			store, err := StoreFromContext(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to get store from context")
				return nil
			}

			// Query queue
			q, err := store.Client.Queue.Query().Where(queue.ID(args.Input.QueueId)).WithVod().Only(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to query queue")
				return nil
			}

			// Query channel
			c, err := q.Edges.Vod.QueryChannel().Only(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to query channel")
				return nil
			}

			// Set the watched channel as not live
			if err := setWatchChannelAsNotLive(ctx, store, c.ID); err != nil {
				log.Error().Err(err).Msg("failed to set watched channel as not live")
				return nil
			}

			// Delete the video
			if err := vods_utility.DeleteVod(ctx, store, q.Edges.Vod.ID, true); err != nil {
				log.Error().Err(err).Msg("failed to delete video")
				return nil
			}
			// Stop the job from being retried
			return &river.ErrorHandlerResult{
				SetCancelled: true, // Set the job as cancelled
			}
		}
	}

	// River invokes the error handler after every failed attempt. Keep queue
	// failure state and notifications for terminal errors only. A remote cancel
	// of a live capture is terminal in River, but the worker deliberately
	// finalizes its partial media before returning, so don't overwrite that
	// successful handoff. A client shutdown is likewise recovered on restart.
	if !shouldFinalizeArchiveError(ctx, job, err) {
		return nil
	}

	// if the job is an archive job, mark it as failed in the queue and send an error notification
	if utils.Contains(job.Tags, archive_tag) && !utils.Contains(job.Tags, allow_fail_tag) {
		// unmarshal custom arguments
		var args RiverJobArgs
		if err := json.Unmarshal(job.EncodedArgs, &args); err != nil {
			return nil
		}
		// get store
		store, err := StoreFromContext(ctx)
		if err != nil {
			return nil
		}
		// set queue status to failed
		if err := setQueueStatus(ctx, store.Client, QueueStatusInput{
			Status:  utils.Failed,
			QueueId: args.Input.QueueId,
			Task:    utils.GetTaskName(job.Kind),
		}); err != nil {
			return nil
		}

		dbItems, err := getDatabaseItems(ctx, store.Client, args.Input.QueueId)
		if err != nil {
			return nil
		}
		// send error notification
		if notifSvc, err := NotificationServiceFromContext(ctx); err == nil {
			go func() {
				notifCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
				defer cancel()
				defer func() {
					if r := recover(); r != nil {
						log.Error().Interface("panic", r).Msg("panic in notification")
					}
				}()
				notifSvc.SendError(notifCtx, &dbItems.Channel, &dbItems.Video, &dbItems.Queue, job.Kind)
			}()
		}
	}
	return nil
}

func (*CustomErrorHandler) HandlePanic(ctx context.Context, job *rivertype.JobRow, panicVal any, trace string) *river.ErrorHandlerResult {
	log.Error().Str("job_id", fmt.Sprintf("%d", job.ID)).Str("attempt", fmt.Sprintf("%d", job.Attempt)).Str("attempted_by", attemptedBy(job)).Str("args", string(job.EncodedArgs)).Str("panic_val", fmt.Sprintf("%v", panicVal)).Str("trace", trace).Msg("task error")
	if ctx.Err() != nil || job.Attempt < job.MaxAttempts {
		return nil
	}

	// if the job is an archive job, mark it as failed in the queue and send an error notification
	if utils.Contains(job.Tags, archive_tag) && !utils.Contains(job.Tags, allow_fail_tag) {
		// unmarshal custom arguments
		var args RiverJobArgs
		if err := json.Unmarshal(job.EncodedArgs, &args); err != nil {
			return nil
		}
		store, err := StoreFromContext(ctx)
		if err != nil {
			return nil
		}
		// set queue status to failed
		if err := setQueueStatus(ctx, store.Client, QueueStatusInput{
			Status:  utils.Failed,
			QueueId: args.Input.QueueId,
			Task:    utils.GetTaskName(job.Kind),
		}); err != nil {
			return nil
		}

		dbItems, err := getDatabaseItems(ctx, store.Client, args.Input.QueueId)
		if err != nil {
			return nil
		}
		// send error notification
		if notifSvc, err := NotificationServiceFromContext(ctx); err == nil {
			go func() {
				notifCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
				defer cancel()
				defer func() {
					if r := recover(); r != nil {
						log.Error().Interface("panic", r).Msg("panic in notification")
					}
				}()
				notifSvc.SendError(notifCtx, &dbItems.Channel, &dbItems.Video, &dbItems.Queue, job.Kind)
			}()
		}
	}

	return nil
}

func attemptedBy(job *rivertype.JobRow) string {
	index := job.Attempt - 1
	if index < 0 || index >= len(job.AttemptedBy) {
		return ""
	}
	return job.AttemptedBy[index]
}

func shouldFinalizeArchiveError(ctx context.Context, job *rivertype.JobRow, err error) bool {
	remoteCancellation := errors.Is(err, rivertype.ErrJobCancelledRemotely)
	if remoteCancellation {
		return job.Kind != string(utils.TaskDownloadLiveVideo) && job.Kind != string(utils.TaskDownloadLiveChat)
	}
	if ctx.Err() != nil && errors.Is(err, context.Canceled) {
		return false
	}
	return job.Attempt >= job.MaxAttempts
}

// setWatchChannelAsNotLive marks the watched channel as not live
func setWatchChannelAsNotLive(ctx context.Context, store *database.Database, channelId uuid.UUID) error {
	watchedChannel, err := store.Client.Live.Query().Where(entLive.HasChannelWith(entChannel.ID(channelId))).Only(ctx)
	if err != nil {
		if _, ok := err.(*ent.NotFoundError); ok {
			log.Debug().Str("channel_id", channelId.String()).Msg("watched channel not found")
		} else {
			return err
		}
	}
	// mark channel as not live if it exists
	if watchedChannel != nil {
		err = store.Client.Live.UpdateOneID(watchedChannel.ID).SetIsLive(false).Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// Update video storage usage
type UpdateVideoStorageUsage struct {
	VideoID *uuid.UUID // Optional: if provided, only update this specific video
}

func (UpdateVideoStorageUsage) Kind() string { return TaskUpdateVideoStorageUsage }

func (w UpdateVideoStorageUsage) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
	}
}

func (w *UpdateVideoStorageUsageWorker) Timeout(job *river.Job[UpdateVideoStorageUsage]) time.Duration {
	return 5 * time.Minute
}

type UpdateVideoStorageUsageWorker struct {
	river.WorkerDefaults[UpdateVideoStorageUsage]
}

// updateVideoStorageSize helper to update storage size for a single video
func updateVideoStorageSize(ctx context.Context, logger zerolog.Logger, store *database.Database, video *ent.Vod) error {
	if video.VideoPath == "" {
		logger.Warn().Msgf("video %s has no video path, skipping storage size update", video.ID)
		return nil // Skip if no video path
	}
	directory := filepath.Dir(video.VideoPath)
	// If VideoHlsPath is set, the actual video files are in a parent directory, so go up one more level.
	if video.VideoHlsPath != "" {
		directory = filepath.Dir(directory)
	}
	size, err := utils.GetSizeOfDirectory(directory)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to get size of directory %s for video %s", directory, video.ID)
		return fmt.Errorf("failed to get size of directory %s for video %s: %w", directory, video.ID, err)
	}
	// Update the video storage size
	if video.StorageSizeBytes != size {
		_, err = store.Client.Vod.UpdateOneID(video.ID).SetStorageSizeBytes(size).Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to update video %s storage size: %v", video.ID, err)
		}
		logger.Info().Msgf("updated video %s storage size to %d bytes", video.ID, size)
	} else {
		logger.Debug().Msgf("video %s storage size is already %d bytes, skipping update", video.ID, size)
	}
	return nil
}

func (w UpdateVideoStorageUsageWorker) Work(ctx context.Context, job *river.Job[UpdateVideoStorageUsage]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()
	logger.Info().Msg("starting task")

	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	if job.Args.VideoID != nil {
		video, err := store.Client.Vod.Get(ctx, *job.Args.VideoID)
		if err != nil {
			return fmt.Errorf("failed to fetch video %s: %v", job.Args.VideoID, err)
		}
		if err := updateVideoStorageSize(ctx, logger, store, video); err != nil {
			return err
		}
	} else {
		const batchSize = 100
		offset := 0
		for {
			videos, err := store.Client.Vod.Query().Limit(batchSize).Offset(offset).All(ctx)
			if err != nil {
				return fmt.Errorf("failed to fetch videos: %v", err)
			}
			if len(videos) == 0 {
				break
			}
			for _, video := range videos {
				if err := updateVideoStorageSize(ctx, logger, store, video); err != nil {
					// Only log and continue on error for individual videos
					logger.Error().Err(err).Msgf("failed to update storage size for video %s", video.ID)
					continue
				}
			}
			offset += batchSize
		}
	}

	// Queue task to update channel storage usage
	enqueuer, err := EnqueuerFromContext(ctx)
	if err != nil {
		return err
	}
	_, err = enqueuer.Insert(ctx, &UpdateChannelStorageUsage{}, nil)
	if err != nil {
		logger.Error().Err(err).Msg("error queuing channel storage usage update task")
		return fmt.Errorf("error queuing channel storage usage update task: %w", err)
	}

	logger.Info().Msg("task completed")
	return nil
}

// Update channel storage usage
type UpdateChannelStorageUsage struct {
}

func (UpdateChannelStorageUsage) Kind() string { return TaskUpdateChannelStorageUsage }

func (w UpdateChannelStorageUsage) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
	}
}

func (w *UpdateChannelStorageUsageWorker) Timeout(job *river.Job[UpdateChannelStorageUsage]) time.Duration {
	return 5 * time.Minute
}

type UpdateChannelStorageUsageWorker struct {
	river.WorkerDefaults[UpdateChannelStorageUsage]
}

func (w UpdateChannelStorageUsageWorker) Work(ctx context.Context, job *river.Job[UpdateChannelStorageUsage]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()
	logger.Info().Msg("starting task")

	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	vods, err := store.Client.Vod.Query().
		WithChannel().
		Select(entVod.FieldStorageSizeBytes).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch VODs with channels: %w", err)
	}

	channelStorageMap := make(map[uuid.UUID]int64)

	for _, vod := range vods {
		if vod.Edges.Channel != nil {
			channelStorageMap[vod.Edges.Channel.ID] += vod.StorageSizeBytes
		}
	}

	channels, err := store.Client.Channel.Query().All(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch channels: %w", err)
	}

	for _, channel := range channels {
		totalStorage := channelStorageMap[channel.ID]
		if channel.StorageSizeBytes != totalStorage {
			if _, err := store.Client.Channel.
				UpdateOneID(channel.ID).
				SetStorageSizeBytes(totalStorage).
				Save(ctx); err != nil {
				return fmt.Errorf("failed to update channel %s storage: %w", channel.Name, err)
			}
			logger.Info().Msgf("updated channel %s storage size to %d", channel.Name, totalStorage)
		} else {
			logger.Debug().Msgf("channel %s storage size is already correct", channel.Name)
		}
	}

	logger.Info().Msg("task completed")
	return nil
}
