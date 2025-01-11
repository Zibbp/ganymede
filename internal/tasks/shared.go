package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	entChannel "github.com/zibbp/ganymede/ent/channel"
	entLive "github.com/zibbp/ganymede/ent/live"
	"github.com/zibbp/ganymede/ent/queue"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/errors"
	"github.com/zibbp/ganymede/internal/notification"
	"github.com/zibbp/ganymede/internal/platform"
	tasks_shared "github.com/zibbp/ganymede/internal/tasks/shared"
	"github.com/zibbp/ganymede/internal/utils"
)

var archive_tag = "archive"
var allow_fail_tag = "allow_fail"

var (
	TaskUpdateStreamVideoId         = "update_stream_video_id"
	TaskGenerateStaticThumbnails    = "generate_static_thumbnails"
	TaskGenerateSpriteThumbnails    = "generate_sprite_thumbnails"
	TaskArchiveWatchdog             = "archive_watchdog"
	TaskCheckChannelsForLivestreams = "chanel_channels_for_livestreams"
	TaskCheckChannelsForNewVideos   = "check_channels_for_new_videos"
	TaskCheckChannelsForNewClips    = "check_channels_for_new_clips"
	TaskPruneVideos                 = "prune_videos"
	TaskImportVideos                = "import_videos"
	TaskAuthenticatePlatform        = "authenticate_platform"
	TaskFetchJWKS                   = "fetch_jwks"
	TaskSaveVideoChapters           = "save_video_chapters"
)

var (
	QueueVideoDownload            = "video-download"
	QueueVideoPostProcess         = "video-postprocess"
	QueueChatDownload             = "chat-download"
	QueueChatRender               = "chat-render"
	QueueGenerateThumbnailSprites = "generate-thumbnail-sprites"
)

type ArchiveVideoInput struct {
	QueueId       uuid.UUID `json:"queue_id"`
	HeartBeatTime time.Time `json:"heartbeat_time"` // do not set this field
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

	if dbItems.Queue.LiveArchive {
		if dbItems.Queue.TaskVideoDownload == utils.Success && dbItems.Queue.TaskVideoConvert == utils.Success && dbItems.Queue.TaskVideoMove == utils.Success && dbItems.Queue.TaskChatDownload == utils.Success && dbItems.Queue.TaskChatConvert == utils.Success && dbItems.Queue.TaskChatRender == utils.Success && dbItems.Queue.TaskChatMove == utils.Success {
			log.Debug().Msgf("all tasks for video %s are done", dbItems.Video.ID.String())

			_, err := dbItems.Queue.Update().SetVideoProcessing(false).SetChatProcessing(false).SetProcessing(false).Save(context.Background())
			if err != nil {
				return err
			}

			_, err = entClient.Vod.UpdateOneID(dbItems.Video.ID).SetProcessing(false).Save(context.Background())
			if err != nil {
				return err
			}

			notification.SendLiveArchiveSuccessNotification(&dbItems.Channel, &dbItems.Video, &dbItems.Queue)
		}
	} else {
		if dbItems.Queue.TaskVideoDownload == utils.Success && dbItems.Queue.TaskVideoConvert == utils.Success && dbItems.Queue.TaskVideoMove == utils.Success && dbItems.Queue.TaskChatDownload == utils.Success && dbItems.Queue.TaskChatRender == utils.Success && dbItems.Queue.TaskChatMove == utils.Success {
			log.Debug().Msgf("all tasks for video %s are done", dbItems.Video.ID.String())

			_, err := dbItems.Queue.Update().SetVideoProcessing(false).SetChatProcessing(false).SetProcessing(false).Save(context.Background())
			if err != nil {
				return err
			}

			_, err = entClient.Vod.UpdateOneID(dbItems.Video.ID).SetProcessing(false).Save(context.Background())
			if err != nil {
				return err
			}

			notification.SendVideoArchiveSuccessNotification(&dbItems.Channel, &dbItems.Video, &dbItems.Queue)
		}
	}

	return nil
}

// forceJobRetry forces a job to be retried. River's retry function does not touch running jobs, so we have to do it ourselves.
func forceJobRetry(ctx context.Context, conn *pgxpool.Pool, id int64) error {
	query := `
		UPDATE river_job
		SET state = $1
		WHERE id = $2
	`

	r, err := conn.Exec(ctx, query, rivertype.JobStateRetryable, id)
	if err != nil {
		return err
	}
	if r.RowsAffected() == 0 {
		return fmt.Errorf("job not found")
	}

	return nil
}

// forceDeleteJob forces a job to be deleted. River's delete function does not touch running jobs, so we have to do it ourselves.
func forceDeleteJob(ctx context.Context, conn *pgxpool.Pool, id int64) error {
	query := `
		DELETE FROM river_job
		WHERE id = $1
		RETURNING id
	`

	r, err := conn.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if r.RowsAffected() == 0 {
		return fmt.Errorf("job not found")
	}

	return nil
}

type GetTaskFilter struct {
	Kind    string
	QueueId uuid.UUID
	Tags    []string
}

func getTaskId(ctx context.Context, client *river.Client[pgx.Tx], filter GetTaskFilter, params *river.JobListParams) (int64, error) {
	jobs, err := client.JobList(ctx, params)
	if err != nil {
		return 0, err
	}

	for _, job := range jobs.Jobs {
		var args RiverJobArgs
		if err := json.Unmarshal(job.EncodedArgs, &args); err != nil {
			return 0, err
		}

		// Apply filters
		if filter.Kind != "" && job.Kind != filter.Kind {
			continue
		}
		if filter.QueueId != uuid.Nil && args.Input.QueueId != filter.QueueId {
			continue
		}
		if len(filter.Tags) > 0 && !containsAllTags(job.Tags, filter.Tags) {
			continue
		}

		// If all filters pass, return the job ID
		return job.ID, nil
	}
	return 0, nil
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

type CustomErrorHandler struct{}

func (*CustomErrorHandler) HandleError(ctx context.Context, job *rivertype.JobRow, err error) *river.ErrorHandlerResult {
	log.Error().Str("job_id", fmt.Sprintf("%d", job.ID)).Str("attempt", fmt.Sprintf("%d", job.Attempt)).Str("attempted_by", job.AttemptedBy[job.Attempt-1]).Str("args", string(job.EncodedArgs)).Err(err).Msg("task error")

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
		notification.SendErrorNotification(&dbItems.Channel, &dbItems.Video, &dbItems.Queue, job.Kind)
	}
	return nil
}

func (*CustomErrorHandler) HandlePanic(ctx context.Context, job *rivertype.JobRow, panicVal any, trace string) *river.ErrorHandlerResult {
	log.Error().Str("job_id", fmt.Sprintf("%d", job.ID)).Str("attempt", fmt.Sprintf("%d", job.Attempt)).Str("attempted_by", job.AttemptedBy[job.Attempt-1]).Str("args", string(job.EncodedArgs)).Str("panic_val", fmt.Sprintf("%v", panicVal)).Str("trace", trace).Msg("task error")

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
		notification.SendErrorNotification(&dbItems.Channel, &dbItems.Video, &dbItems.Queue, job.Kind)
	}

	return nil
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
