package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/riverqueue/river/rivertype"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/queue"
	"github.com/zibbp/ganymede/internal/utils"
)

type QueueService interface {
	CreateQueueItem(queueDto queue.Queue, vID uuid.UUID) (*ent.Queue, error)
	GetQueueItems(c echo.Context) ([]*ent.Queue, error)
	GetQueueItemsFilter(c echo.Context, pro bool) ([]*ent.Queue, error)
	GetQueueItem(id uuid.UUID) (*ent.Queue, error)
	UpdateQueueItem(queueDto queue.Queue, id uuid.UUID) (*ent.Queue, error)
	DeleteQueueItem(c echo.Context, id uuid.UUID) error
	ReadLogFile(c echo.Context, id uuid.UUID, logType string) ([]byte, error)
	StopQueueItem(ctx context.Context, id uuid.UUID) error
	StartQueueTask(ctx context.Context, input queue.StartQueueTaskInput) (*rivertype.JobRow, error)
}

type CreateQueueRequest struct {
	VodID string `json:"vod_id" validate:"required"`
}

type StartQueueTaskRequest struct {
	QueueId  uuid.UUID `json:"queue_id" validate:"required,uuid4"`
	TaskName string    `json:"task_name" validate:"required,oneof=task_vod_create_folder task_vod_download_thumbnail task_vod_save_info task_video_download task_video_convert task_video_move task_chat_download task_chat_convert task_chat_render task_chat_move task_live_chat_download task_live_video_download"`
	Continue bool      `json:"continue"`
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

// CreateQueueItem godoc
//
//	@Summary		Create a queue item
//	@Description	Create a queue item
//	@Tags			queue
//	@Accept			json
//	@Produce		json
//	@Param			body	body		CreateQueueRequest	true	"Create queue item"
//	@Success		201		{object}	ent.Queue
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/queue [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) CreateQueueItem(c echo.Context) error {
	cqt := new(CreateQueueRequest)
	if err := c.Bind(cqt); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(cqt); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	vID, err := uuid.Parse(cqt.VodID)
	if err != nil {
		return err
	}

	cqtDto := queue.Queue{}

	que, err := h.Service.QueueService.CreateQueueItem(cqtDto, vID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, que, "created queue entry")
}

// GetQueueItems godoc
//
//	@Summary		Get queue items
//	@Description	Get queue items
//	@Tags			queue
//	@Accept			json
//	@Produce		json
//	@Param			processing	query		string	false	"Get processing queue items"
//	@Success		200			{object}	[]ent.Queue
//	@Failure		400			{object}	utils.ErrorResponse
//	@Failure		500			{object}	utils.ErrorResponse
//	@Router			/queue [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetQueueItems(c echo.Context) error {
	processing := false
	processingParam := c.QueryParam("processing")
	if processingParam == "true" {
		processing = true
	}

	if processing {
		qFilter, err := h.Service.QueueService.GetQueueItemsFilter(c, processing)
		if err != nil {
			return ErrorResponse(c, http.StatusInternalServerError, err.Error())
		}
		return SuccessResponse(c, qFilter, "queue items")
	}

	q, err := h.Service.QueueService.GetQueueItems(c)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, q, "queue items")
}

// GetQueueItem godoc
//
//	@Summary		Get queue item
//	@Description	Get queue item
//	@Tags			queue
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Queue item id"
//	@Success		200	{object}	ent.Queue
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/queue/{id} [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetQueueItem(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	q, err := h.Service.QueueService.GetQueueItem(id)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, q, "queue item")
}

// UpdateQueueItem godoc
//
//	@Summary		Update queue item
//	@Description	Update queue item
//	@Tags			queue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Queue item id"
//	@Param			body	body		UpdateQueueRequest	true	"Update queue item"
//	@Success		200		{object}	ent.Queue
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/queue/{id} [put]
//	@Security		ApiKeyCookieAuth
func (h *Handler) UpdateQueueItem(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	uqr := new(UpdateQueueRequest)
	if err := c.Bind(uqr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(uqr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
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
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, que, "updated queue item")
}

// DeleteQueueItem godoc
//
//	@Summary		Delete queue item
//	@Description	Delete queue item
//	@Tags			queue
//	@Accept			json
//	@Produce		json
//	@Param			id	path	string	true	"Queue item id"
//	@Success		204
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/queue/{id} [delete]
//	@Security		ApiKeyCookieAuth
func (h *Handler) DeleteQueueItem(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	err = h.Service.QueueService.DeleteQueueItem(c, id)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, "", "queue item deleted")
}

// ReadQueueLogFile godoc
//
//	@Summary		Read queue log file
//	@Description	Read queue log file
//	@Tags			queue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string	true	"Queue item id"
//	@Param			type	query		string	true	"Log type: video, video-convert, chat, chat-render, or chat-convert"
//	@Success		200		{object}	string
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/queue/{id}/tail [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) ReadQueueLogFile(c echo.Context) error {
	id := c.Param("id")

	uuid, err := utils.IsValidUUID(id)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid id")
	}

	logType := c.QueryParam("type")
	if len(logType) == 0 {
		return ErrorResponse(c, http.StatusBadRequest, "type is required: video, video-convert, chat, chat-render, or chat-convert")
	}
	// Validate logType
	validLogType, err := utils.ValidateLogType(logType)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	log, err := h.Service.QueueService.ReadLogFile(c, uuid, validLogType)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, string(log), fmt.Sprintf("%s log file for %s", logType, uuid))
}

// StopQueueItem godoc
//
//	@Summary		Stop a queue item
//	@Description	Stop processing the video and chat downloads of an active queue item
//	@Tags			queue
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string	true	"Queue item id"
//	@Success		200		{object}	string
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/queue/{id}/stop [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) StopQueueItem(c echo.Context) error {
	id := c.Param("id")

	uuid, err := utils.IsValidUUID(id)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid id")
	}

	err = h.Service.QueueService.StopQueueItem(c.Request().Context(), uuid)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, "", "queue item stopped")
}

// StartQueueTask godoc
//
//	@Summary		Start a queue task for a queue
//	@Description	Start a specific queue task
//	@Tags			queue
//	@Accept			json
//	@Produce		json
//	@Success		200		{object}	string
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/queue/task/start [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) StartQueueTask(c echo.Context) error {
	body := new(StartQueueTaskRequest)
	if err := c.Bind(body); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(body); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	_, err := h.Service.QueueService.StartQueueTask(c.Request().Context(), queue.StartQueueTaskInput{
		QueueId:  body.QueueId,
		TaskName: body.TaskName,
		Continue: body.Continue,
	})

	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return SuccessResponse(c, "", fmt.Sprintf("started %s for %s", body.TaskName, body.QueueId))
}
