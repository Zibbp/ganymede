package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/queue"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/tasks"
	tasks_client "github.com/zibbp/ganymede/internal/tasks/client"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
)

type Service struct {
	Store          *database.Database
	VodService     *vod.Service
	ChannelService *channel.Service
	RiverClient    *tasks_client.RiverClient
}

type StartQueueTaskInput struct {
	QueueId  uuid.UUID
	TaskName string
	Continue bool
}

func NewService(store *database.Database, vodService *vod.Service, channelService *channel.Service, riverClient *tasks_client.RiverClient) *Service {
	return &Service{Store: store, VodService: vodService, ChannelService: channelService, RiverClient: riverClient}
}

type Queue struct {
	ID                       uuid.UUID        `json:"id"`
	LiveArchive              bool             `json:"live_archive"`
	OnHold                   bool             `json:"on_hold"`
	VideoProcessing          bool             `json:"video_processing"`
	ChatProcessing           bool             `json:"chat_processing"`
	Processing               bool             `json:"processing"`
	TaskVodCreateFolder      utils.TaskStatus `json:"task_vod_create_folder"`
	TaskVodDownloadThumbnail utils.TaskStatus `json:"task_vod_download_thumbnail"`
	TaskVodSaveInfo          utils.TaskStatus `json:"task_vod_save_info"`
	TaskVideoDownload        utils.TaskStatus `json:"task_video_download"`
	TaskVideoConvert         utils.TaskStatus `json:"task_video_convert"`
	TaskVideoMove            utils.TaskStatus `json:"task_video_move"`
	TaskChatDownload         utils.TaskStatus `json:"task_chat_download"`
	TaskChatConvert          utils.TaskStatus `json:"task_chat_convert"`
	TaskChatRender           utils.TaskStatus `json:"task_chat_render"`
	TaskChatMove             utils.TaskStatus `json:"task_chat_move"`
	ArchiveChat              bool             `json:"archive_chat"`
	RenderChat               bool             `json:"render_chat"`
	UpdatedAt                time.Time        `json:"updated_at"`
	CreatedAt                time.Time        `json:"created_at"`
}

func (s *Service) CreateQueueItem(queueDto Queue, vID uuid.UUID) (*ent.Queue, error) {
	if queueDto.LiveArchive {
		q, err := s.Store.Client.Queue.Create().SetVodID(vID).SetLiveArchive(true).SetArchiveChat(queueDto.ArchiveChat).SetRenderChat(queueDto.RenderChat).Save(context.Background())
		if err != nil {
			if _, ok := err.(*ent.ConstraintError); ok {
				return nil, fmt.Errorf("queue item exists for vod or vod does not exist")
			}
			log.Debug().Err(err).Msg("error creating queue")
			return nil, fmt.Errorf("error creating queue: %v", err)
		}
		return q, nil
	} else {
		q, err := s.Store.Client.Queue.Create().SetVodID(vID).SetArchiveChat(queueDto.ArchiveChat).SetRenderChat(queueDto.RenderChat).Save(context.Background())
		if err != nil {
			if _, ok := err.(*ent.ConstraintError); ok {
				return nil, fmt.Errorf("queue item exists for vod or vod does not exist")
			}
			log.Debug().Err(err).Msg("error creating queue")
			return nil, fmt.Errorf("error creating queue: %v", err)
		}
		return q, nil
	}

}

func (s *Service) UpdateQueueItem(queueDto Queue, qID uuid.UUID) (*ent.Queue, error) {
	q, err := s.Store.Client.Queue.UpdateOneID(qID).SetLiveArchive(queueDto.LiveArchive).SetOnHold(queueDto.OnHold).SetVideoProcessing(queueDto.VideoProcessing).SetChatProcessing(queueDto.ChatProcessing).SetProcessing(queueDto.Processing).SetTaskVodCreateFolder(queueDto.TaskVodCreateFolder).SetTaskVodDownloadThumbnail(queueDto.TaskVodDownloadThumbnail).SetTaskVodSaveInfo(queueDto.TaskVodSaveInfo).SetTaskVideoDownload(queueDto.TaskVideoDownload).SetTaskVideoConvert(queueDto.TaskVideoConvert).SetTaskVideoMove(queueDto.TaskVideoMove).SetTaskChatDownload(queueDto.TaskChatDownload).SetTaskChatConvert(queueDto.TaskChatConvert).SetArchiveChat(queueDto.ArchiveChat).SetRenderChat(queueDto.RenderChat).SetTaskChatRender(queueDto.TaskChatRender).SetTaskChatMove(queueDto.TaskChatMove).Save(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error updating queue: %v", err)
	}
	return q, nil
}

func (s *Service) GetQueueItems(c echo.Context) ([]*ent.Queue, error) {
	q, err := s.Store.Client.Queue.Query().WithVod(func(q *ent.VodQuery) {
		q.WithChannel()
	}).Order(ent.Desc(queue.FieldCreatedAt)).All(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error getting queue task: %v", err)
	}
	return q, nil
}
func (s *Service) GetQueueItemsFilter(c echo.Context, processing bool) ([]*ent.Queue, error) {
	q, err := s.Store.Client.Queue.Query().Where(queue.Processing(processing)).WithVod(func(q *ent.VodQuery) {
		q.WithChannel()
	}).Order(ent.Asc(queue.FieldCreatedAt)).All(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error getting queue task: %v", err)
	}
	return q, nil
}

func (s *Service) DeleteQueueItem(c echo.Context, qID uuid.UUID) error {
	err := s.Store.Client.Queue.DeleteOneID(qID).Exec(c.Request().Context())
	if err != nil {
		return fmt.Errorf("error deleting queue: %v", err)
	}
	return nil
}

func (s *Service) GetQueueItem(qID uuid.UUID) (*ent.Queue, error) {
	q, err := s.Store.Client.Queue.Query().Where(queue.ID(qID)).WithVod().Only(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error getting queue task: %v", err)
	}
	return q, nil
}

func (s *Service) ReadLogFile(c echo.Context, qID uuid.UUID, logType string) ([]byte, error) {
	env := config.GetEnvConfig()
	q, err := s.GetQueueItem(qID)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("%s/%s-%s.log", env.LogsDir, q.Edges.Vod.ID, logType)
	logLines, err := utils.ReadLastLines(path, 30)
	if err != nil {
		return nil, err
	}
	return []byte(logLines), nil
}

func (s *Service) ArchiveGetQueueItem(qID uuid.UUID) (*ent.Queue, error) {
	q, err := s.Store.Client.Queue.Query().Where(queue.ID(qID)).WithVod().Only(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error archiving queue: %v", err)
	}
	return q, nil
}

// StopQueueItem stops a queue item's tasks by canceling each job's context
func (s *Service) StopQueueItem(ctx context.Context, id uuid.UUID) error {

	err := s.RiverClient.CancelJobsForQueueId(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) StartQueueTask(ctx context.Context, input StartQueueTaskInput) (*rivertype.JobRow, error) {

	// ensure queue exists
	_, err := s.GetQueueItem(input.QueueId)
	if err != nil {
		return nil, err
	}

	var task river.JobArgs

	taskInput := tasks.ArchiveVideoInput{
		QueueId: input.QueueId,
	}

	switch input.TaskName {
	case "task_vod_create_folder":
		task = tasks.CreateDirectoryArgs{
			Continue: input.Continue,
			Input:    taskInput,
		}

	case "task_vod_download_thumbnail":
		task = tasks.DownloadThumbnailArgs{
			Continue: input.Continue,
			Input:    taskInput,
		}

	case "task_vod_save_info":
		task = tasks.SaveVideoInfoArgs{
			Continue: input.Continue,
			Input:    taskInput,
		}

	case "task_video_download":
		task = tasks.DownloadVideoArgs{
			Continue: input.Continue,
			Input:    taskInput,
		}

	case "task_video_convert":
		task = tasks.PostProcessVideoArgs{
			Continue: input.Continue,
			Input:    taskInput,
		}

	case "task_video_move":
		task = tasks.MoveVideoArgs{
			Continue: input.Continue,
			Input:    taskInput,
		}

	case "task_chat_download":
		task = tasks.DownloadChatArgs{
			Continue: input.Continue,
			Input:    taskInput,
		}

	case "task_chat_convert":
		task = tasks.ConvertLiveChatArgs{
			Continue: input.Continue,
			Input:    taskInput,
		}

	case "task_chat_render":
		task = tasks.RenderChatArgs{
			Continue: input.Continue,
			Input:    taskInput,
		}

	case "task_chat_move":
		task = tasks.MoveChatArgs{
			Continue: input.Continue,
			Input:    taskInput,
		}

	case "task_live_chat_download":
		task = tasks.DownloadLiveChatArgs{
			Continue: input.Continue,
			Input:    taskInput,
		}

	case "task_live_video_download":
		task = tasks.DownloadLiveVideoArgs{
			Continue: input.Continue,
			Input:    taskInput,
		}

	default:
		return nil, fmt.Errorf("unknown task: %s", input.TaskName)
	}

	job, err := s.RiverClient.Client.Insert(ctx, task, nil)
	if err != nil {
		return nil, err
	}

	return job.Job, err
}
