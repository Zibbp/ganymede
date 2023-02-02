package http

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/auth"
)

type PlaybackService interface {
	UpdateProgress(c *auth.CustomContext, vID uuid.UUID, time int) error
	GetProgress(c *auth.CustomContext, vID uuid.UUID) (*ent.Playback, error)
	GetAllProgress(c *auth.CustomContext) ([]*ent.Playback, error)
	UpdateStatus(c *auth.CustomContext, vID uuid.UUID, status string) error
	DeleteProgress(c *auth.CustomContext, vID uuid.UUID) error
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
	cc := c.(*auth.CustomContext)
	upr := new(UpdateProgressRequest)
	if err := c.Bind(upr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(upr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	vID, err := uuid.Parse(upr.VodID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid vod id")
	}
	err = h.Service.PlaybackService.UpdateProgress(cc, vID, upr.Time)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, "ok")
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
	cc := c.(*auth.CustomContext)
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid vod id")
	}
	playbackEntry, err := h.Service.PlaybackService.GetProgress(cc, vID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, playbackEntry)
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
	cc := c.(*auth.CustomContext)
	playbackEntries, err := h.Service.PlaybackService.GetAllProgress(cc)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, playbackEntries)
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
	cc := c.(*auth.CustomContext)
	usr := new(UpdateStatusRequest)
	if err := c.Bind(usr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(usr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	vID, err := uuid.Parse(usr.VodID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid vod id")
	}
	err = h.Service.PlaybackService.UpdateStatus(cc, vID, usr.Status)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, "ok")
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
	cc := c.(*auth.CustomContext)
	vID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid vod id")
	}
	err = h.Service.PlaybackService.DeleteProgress(cc, vID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, "ok")
}
