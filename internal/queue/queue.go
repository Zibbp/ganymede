package queue

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/queue"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
	"time"
)

type Service struct {
	Store          *database.Database
	VodService     *vod.Service
	ChannelService *channel.Service
}

func NewService(store *database.Database, vodService *vod.Service, channelService *channel.Service) *Service {
	return &Service{Store: store, VodService: vodService, ChannelService: channelService}
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
	UpdatedAt                time.Time        `json:"updated_at"`
	CreatedAt                time.Time        `json:"created_at"`
}

func (s *Service) CreateQueueItem(queueDto Queue, vID uuid.UUID) (*ent.Queue, error) {
	if queueDto.LiveArchive == true {
		q, err := s.Store.Client.Queue.Create().SetVodID(vID).SetLiveArchive(true).Save(context.Background())
		if err != nil {
			if _, ok := err.(*ent.ConstraintError); ok {
				return nil, fmt.Errorf("queue item exists for vod or vod does not exist")
			}
			log.Debug().Err(err).Msg("error creating queue")
			return nil, fmt.Errorf("error creating queue: %v", err)
		}
		return q, nil
	} else {
		q, err := s.Store.Client.Queue.Create().SetVodID(vID).Save(context.Background())
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
	q, err := s.Store.Client.Queue.UpdateOneID(qID).SetLiveArchive(queueDto.LiveArchive).SetOnHold(queueDto.OnHold).SetVideoProcessing(queueDto.VideoProcessing).SetChatProcessing(queueDto.ChatProcessing).SetProcessing(queueDto.Processing).SetTaskVodCreateFolder(queueDto.TaskVodCreateFolder).SetTaskVodDownloadThumbnail(queueDto.TaskVodDownloadThumbnail).SetTaskVodSaveInfo(queueDto.TaskVodSaveInfo).SetTaskVideoDownload(queueDto.TaskVideoDownload).SetTaskVideoConvert(queueDto.TaskVideoConvert).SetTaskVideoMove(queueDto.TaskVideoMove).SetTaskChatDownload(queueDto.TaskChatDownload).SetTaskChatConvert(queueDto.TaskChatConvert).SetTaskChatRender(queueDto.TaskChatRender).SetTaskChatMove(queueDto.TaskChatMove).Save(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error updating queue: %v", err)
	}
	return q, nil
}

func (s *Service) GetQueueItems(c echo.Context) ([]*ent.Queue, error) {
	q, err := s.Store.Client.Queue.Query().WithVod().Order(ent.Desc(queue.FieldCreatedAt)).All(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error getting queue task: %v", err)
	}
	return q, nil
}
func (s *Service) GetQueueItemsFilter(c echo.Context, processing bool) ([]*ent.Queue, error) {
	q, err := s.Store.Client.Queue.Query().Where(queue.Processing(processing)).WithVod().Order(ent.Asc(queue.FieldCreatedAt)).All(c.Request().Context())
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
	q, err := s.GetQueueItem(qID)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/logs/%s_%s-%s.log", q.Edges.Vod.ExtID, q.Edges.Vod.ID, logType)
	logLines, err := utils.ReadLastLines(path, 20)
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
