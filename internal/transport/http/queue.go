package http

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
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
	GetIsQueueActive(c echo.Context) (bool, error)
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
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	q, err := h.Service.QueueService.GetQueueItem(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, q)
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
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	err = h.Service.QueueService.DeleteQueueItem(c, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
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
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	logType := c.QueryParam("type")
	if len(logType) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "type is required: video, video-convert, chat, chat-render, or chat-convert")
	}
	// Validate logType
	validLogType, err := utils.ValidateLogType(logType)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	log, err := h.Service.QueueService.ReadLogFile(c, uuid, validLogType)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, string(log))
}

// GetIsQueueActive godoc
// @Summary		Returns true if queue is active
// @Description	Returns true if queue is active
// @Tags			queue
// @Accept			json
// @Produce		json
// @Success		200		{object}	bool
// @Failure		500		{object}	utils.ErrorResponse
// @Router			/queue/active [get]
func (h *Handler) GetIsQueueActive(c echo.Context) error {
	active, err := h.Service.QueueService.GetIsQueueActive(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, active)
}
