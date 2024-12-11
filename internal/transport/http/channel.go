package http

import (
	"context"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/channel"
)

type ChannelService interface {
	CreateChannel(channelDto channel.Channel) (*ent.Channel, error)
	GetChannels() ([]*ent.Channel, error)
	GetChannel(channelID uuid.UUID) (*ent.Channel, error)
	GetChannelByName(channelName string) (*ent.Channel, error)
	DeleteChannel(channelID uuid.UUID) error
	UpdateChannel(channelID uuid.UUID, channelDto channel.Channel) (*ent.Channel, error)
	UpdateChannelImage(ctx context.Context, channelID uuid.UUID) error
}

type CreateChannelRequest struct {
	ExternalID    string `json:"ext_id"`
	Name          string `json:"name" validate:"required,min=2,max=50"`
	DisplayName   string `json:"display_name" validate:"required,min=2,max=50"`
	ImagePath     string `json:"image_path" validate:"required,min=3"`
	Retention     bool   `json:"retention"`
	RetentionDays int64  `json:"retention_days"`
}

// CreateChannel godoc
//
//	@Summary		Create a channel
//	@Description	Create a channel
//	@Tags			channel
//	@Accept			json
//	@Produce		json
//	@Param			channel	body		CreateChannelRequest	true	"Channel"
//	@Success		200		{object}	ent.Channel
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/channel [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) CreateChannel(c echo.Context) error {
	ccr := new(CreateChannelRequest)
	if err := c.Bind(ccr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(ccr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	if ccr.ExternalID == "" {
		// generate random id - doesn't need to be a uuid
		ccr.ExternalID = strconv.Itoa(rand.Intn(1000000))
	}

	ccDto := channel.Channel{
		ExtID:       ccr.ExternalID,
		Name:        ccr.Name,
		DisplayName: ccr.DisplayName,
		ImagePath:   ccr.ImagePath,
	}

	cha, err := h.Service.ChannelService.CreateChannel(ccDto)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return SuccessResponse(c, cha, "channel created")
}

// GetChannels godoc
//
//	@Summary		Get all channels
//	@Description	Returns all channels
//	@Tags			channel
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	[]ent.Channel
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/channel [get]
func (h *Handler) GetChannels(c echo.Context) error {
	channels, err := h.Service.ChannelService.GetChannels()
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return SuccessResponse(c, channels, "channels")
}

// GetChannel godoc
//
//	@Summary		Get a channel
//	@Description	Returns a channel
//	@Tags			channel
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Channel ID"
//	@Success		200	{object}	ent.Channel
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/channel/{id} [get]
func (h *Handler) GetChannel(c echo.Context) error {
	id := c.Param("id")
	cUUID, err := uuid.Parse(id)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	cha, err := h.Service.ChannelService.GetChannel(cUUID)
	if err != nil {
		if err.Error() == "channel not found" {
			return ErrorResponse(c, http.StatusNotFound, err.Error())
		}
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, cha, "channel")
}

// DeleteChannel godoc
//
//	@Summary		Delete a channel
//	@Description	Delete a channel
//	@Tags			channel
//	@Accept			json
//	@Produce		json
//	@Param			id	path	string	true	"Channel ID"
//	@Success		200
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/channel/{id} [delete]
//	@Security		ApiKeyCookieAuth
func (h *Handler) DeleteChannel(c echo.Context) error {
	id := c.Param("id")
	cUUID, err := uuid.Parse(id)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	err = h.Service.ChannelService.DeleteChannel(cUUID)
	if err != nil {
		if err.Error() == "channel not found" {
			return ErrorResponse(c, http.StatusNotFound, err.Error())
		}
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusOK)
}

// UpdateChannel godoc
//
//	@Summary		Update a channel
//	@Description	Update a channel
//	@Tags			channel
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Channel ID"
//	@Param			channel	body		CreateChannelRequest	true	"Channel"
//	@Success		200		{object}	ent.Channel
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		404		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/channel/{id} [put]
//	@Security		ApiKeyCookieAuth
func (h *Handler) UpdateChannel(c echo.Context) error {
	id := c.Param("id")
	cUUID, err := uuid.Parse(id)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	ccr := new(CreateChannelRequest)
	if err := c.Bind(ccr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(ccr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	ccDto := channel.Channel{
		Name:          ccr.Name,
		DisplayName:   ccr.DisplayName,
		ImagePath:     ccr.ImagePath,
		Retention:     ccr.Retention,
		RetentionDays: ccr.RetentionDays,
	}

	cha, err := h.Service.ChannelService.UpdateChannel(cUUID, ccDto)
	if err != nil {
		if err.Error() == "channel not found" {
			return ErrorResponse(c, http.StatusNotFound, err.Error())
		}
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, cha, "channel updated")
}

// GetChannelByName godoc
//
//	@Summary		Get a channel by name
//	@Description	Returns a channel by name
//	@Tags			channel
//	@Accept			json
//	@Produce		json
//	@Param			name	path		string	true	"Channel name"
//	@Success		200		{object}	ent.Channel
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		404		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/channel/name/{name} [get]
func (h *Handler) GetChannelByName(c echo.Context) error {
	name := c.Param("name")
	cha, err := h.Service.ChannelService.GetChannelByName(name)
	if err != nil {
		if err.Error() == "channel not found" {
			return ErrorResponse(c, http.StatusNotFound, err.Error())
		}
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, cha, "channel")
}

func (h *Handler) UpdateChannelImage(c echo.Context) error {
	id := c.Param("id")
	cUUID, err := uuid.Parse(id)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	err = h.Service.ChannelService.UpdateChannelImage(c.Request().Context(), cUUID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, "", "channel image updated")
}
