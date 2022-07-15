package http

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/queue"
	"github.com/zibbp/ganymede/internal/utils"
	"net/http"
)

type QueueService interface {
	CreateQueueItem(queueDto queue.Queue, vID uuid.UUID) (*ent.Queue, error)
	GetQueueItems(c echo.Context) ([]*ent.Queue, error)
	GetQueueItemsFilter(c echo.Context, pro bool) ([]*ent.Queue, error)
	GetQueueItem(id uuid.UUID) (*ent.Queue, error)
	UpdateQueueItem(queueDto queue.Queue, id uuid.UUID) (*ent.Queue, error)
	DeleteQueueItem(c echo.Context, id uuid.UUID) error
	ReadLogFile(c echo.Context, id uuid.UUID, logType string) ([]byte, error)
}

type CreateQueueRequest struct {
	VodID string `json:"vod_id" validate:"required"`
}

type UpdateQueueRequest struct {
	ID                       uuid.UUID        `json:"id"`
	LiveArchive              bool             `json:"live_archive"`
	OnHold                   bool             `json:"on_hold"`
	VideoProcessing          bool             `json:"video_processing"`
	ChatProcessing           bool             `json:"chat_processing"`
	Processing               bool             `json:"processing"`
	TaskVodCreateFolder      utils.TaskStatus `json:"task_vod_create_folder" validate:"required,oneof=pending running success failed"`
	TaskVodDownloadThumbnail utils.TaskStatus `json:"task_vod_download_thumbnail" validate:"required,oneof=pending running success failed"`
	TaskVodSaveInfo          utils.TaskStatus `json:"task_vod_save_info" validate:"required,oneof=pending running success failed"`
	TaskVideoDownload        utils.TaskStatus `json:"task_video_download" validate:"required,oneof=pending running success failed"`
	TaskVideoConvert         utils.TaskStatus `json:"task_video_convert" validate:"required,oneof=pending running success failed"`
	TaskVideoMove            utils.TaskStatus `json:"task_video_move" validate:"required,oneof=pending running success failed"`
	TaskChatDownload         utils.TaskStatus `json:"task_chat_download" validate:"required,oneof=pending running success failed"`
	TaskChatConvert          utils.TaskStatus `json:"task_chat_convert" validate:"required,oneof=pending running success failed"`
	TaskChatRender           utils.TaskStatus `json:"task_chat_render" validate:"required,oneof=pending running success failed"`
	TaskChatMove             utils.TaskStatus `json:"task_chat_move" validate:"required,oneof=pending running success failed"`
}

func (h *Handler) CreateQueueItem(c echo.Context) error {
	cqt := new(CreateQueueRequest)
	if err := c.Bind(cqt); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(cqt); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	vID, err := uuid.Parse(cqt.VodID)
	if err != nil {
		return err
	}

	cqtDto := queue.Queue{}

	que, err := h.Service.QueueService.CreateQueueItem(cqtDto, vID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, que)
}

func (h *Handler) GetQueueItems(c echo.Context) error {
	var pro bool
	processing := c.QueryParam("processing")
	if processing == "true" {
		pro = true
	} else {
		pro = false
	}
	if len(processing) > 0 {
		qFilter, err := h.Service.QueueService.GetQueueItemsFilter(c, pro)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, qFilter)
	}
	q, err := h.Service.QueueService.GetQueueItems(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, q)
}

func (h *Handler) GetQueueItem(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	q, err := h.Service.QueueService.GetQueueItem(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, q)
}

func (h *Handler) UpdateQueueItem(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	uqr := new(UpdateQueueRequest)
	if err := c.Bind(uqr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(uqr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	queueDto := queue.Queue{
		LiveArchive:              uqr.LiveArchive,
		OnHold:                   uqr.OnHold,
		VideoProcessing:          uqr.VideoProcessing,
		ChatProcessing:           uqr.ChatProcessing,
		Processing:               uqr.Processing,
		TaskVodCreateFolder:      uqr.TaskVodCreateFolder,
		TaskVodDownloadThumbnail: uqr.TaskVodDownloadThumbnail,
		TaskVodSaveInfo:          uqr.TaskVodSaveInfo,
		TaskVideoDownload:        uqr.TaskVideoDownload,
		TaskVideoConvert:         uqr.TaskVideoConvert,
		TaskVideoMove:            uqr.TaskVideoMove,
		TaskChatDownload:         uqr.TaskChatDownload,
		TaskChatConvert:          uqr.TaskChatConvert,
		TaskChatRender:           uqr.TaskChatRender,
		TaskChatMove:             uqr.TaskChatMove,
	}

	que, err := h.Service.QueueService.UpdateQueueItem(queueDto, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, que)
}

func (h *Handler) DeleteQueueItem(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	err = h.Service.QueueService.DeleteQueueItem(c, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) ReadQueueLogFile(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	logType := c.QueryParam("type")
	if len(logType) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "type is required: video, video-convert, chat, or chat-render")
	}
	log, err := h.Service.QueueService.ReadLogFile(c, id, logType)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, string(log))
}
