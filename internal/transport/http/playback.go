package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/playback"
)

type PlaybackService interface {
	UpdateProgress(ctx context.Context, userId uuid.UUID, videoId uuid.UUID, time int) error
	GetProgress(ctx context.Context, userId uuid.UUID, videoId uuid.UUID) (*ent.Playback, error)
	GetAllProgress(ctx context.Context, userId uuid.UUID) ([]*ent.Playback, error)
	UpdateStatus(ctx context.Context, userId uuid.UUID, videoId uuid.UUID, status string) error
	DeleteProgress(ctx context.Context, userId uuid.UUID, videoId uuid.UUID) error
	GetLastPlaybacks(ctx context.Context, userId uuid.UUID, limit int) (*playback.GetPlaybackResp, error)
	StartPlayback(c echo.Context, videoId uuid.UUID) error
}

type UpdateProgressRequest struct {
	VodID string `json:"vod_id" validate:"required"`
	Time  int    `json:"time" validate:"required"`
}

type UpdateStatusRequest struct {
	VodID  string `json:"vod_id" validate:"required"`
	Status string `json:"status" validate:"required,oneof=in_progress finished"`
}

// UpdateProgress godoc
//
//	@Summary		Update progress
//	@Description	Update playback progress
//	@Tags			Playback
//	@Accept			json
//	@Produce		json
//	@Param			progress	body		UpdateProgressRequest	true	"progress"
//	@Success		200			{object}	string
//	@Failure		400			{object}	utils.ErrorResponse
//	@Failure		500			{object}	utils.ErrorResponse
//	@Router			/playback/progress [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) UpdateProgress(c echo.Context) error {
	user := userFromContext(c)
	upr := new(UpdateProgressRequest)
	if err := c.Bind(upr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(upr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	vID, err := uuid.Parse(upr.VodID)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid vod id")
	}
	err = h.Service.PlaybackService.UpdateProgress(c.Request().Context(), user.ID, vID, upr.Time)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, "", "ok")
}

// GetProgress godoc
//
//	@Summary		Get progress
//	@Description	Get playback progress
//	@Tags			Playback
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"vod id"
//	@Success		200	{object}	ent.Playback
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/playback/progress/{id} [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetProgress(c echo.Context) error {
	user := userFromContext(c)
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid vod id")
	}
	playbackEntry, err := h.Service.PlaybackService.GetProgress(c.Request().Context(), user.ID, vID)
	if err != nil {
		if errors.Is(err, playback.ErrorPlaybackNotFound) {
			return ErrorResponse(c, http.StatusOK, "playback not found")
		}
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return SuccessResponse(c, playbackEntry, fmt.Sprintf("playback data for %s", vID))
}

// GetAllProgress godoc
//
//	@Summary		Get all progress
//	@Description	Get all playback progress
//	@Tags			Playback
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	[]ent.Playback
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/playback [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetAllProgress(c echo.Context) error {
	user := userFromContext(c)
	playbackEntries, err := h.Service.PlaybackService.GetAllProgress(c.Request().Context(), user.ID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, playbackEntries, "playback entries")
}

// UpdateStatus godoc
//
//	@Summary		Update status
//	@Description	Update playback status
//	@Tags			Playback
//	@Accept			json
//	@Produce		json
//	@Param			status	body		UpdateStatusRequest	true	"status"
//	@Success		200		{object}	string
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/playback/status [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) UpdateStatus(c echo.Context) error {
	user := userFromContext(c)
	usr := new(UpdateStatusRequest)
	if err := c.Bind(usr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(usr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	vID, err := uuid.Parse(usr.VodID)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid vod id")
	}
	err = h.Service.PlaybackService.UpdateStatus(c.Request().Context(), user.ID, vID, usr.Status)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return SuccessResponse(c, "", "ok")
}

// DeleteProgress godoc
//
//	@Summary		Delete progress
//	@Description	Delete playback progress
//	@Tags			Playback
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"vod id"
//	@Success		200	{object}	string
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/playback/{id} [delete]
//	@Security		ApiKeyCookieAuth
func (h *Handler) DeleteProgress(c echo.Context) error {
	user := userFromContext(c)
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid vod id")
	}
	err = h.Service.PlaybackService.DeleteProgress(c.Request().Context(), user.ID, vID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return SuccessResponse(c, "", "ok")
}

func (h *Handler) GetLastPlaybacks(c echo.Context) error {
	user := userFromContext(c)
	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid limit")
	}

	playbackEntries, err := h.Service.PlaybackService.GetLastPlaybacks(c.Request().Context(), user.ID, limit)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return SuccessResponse(c, playbackEntries, "playback entries")
}

// StartPlayback godoc
//
//	@Summary		Start playback
//	@Description	Adds a view to the video local view count
//	@Tags			Playback
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"vod id"
//	@Success		200	{object}	string
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/playback/start [post]
func (h *Handler) StartPlayback(c echo.Context) error {
	videoId, err := uuid.Parse(c.QueryParam("video_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid video id")
	}

	err = h.Service.PlaybackService.StartPlayback(c, videoId)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return SuccessResponse(c, "", "ok")
}
