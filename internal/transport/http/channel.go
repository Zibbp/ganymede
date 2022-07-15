package http

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/channel"
	"net/http"
)

type ChannelService interface {
	CreateChannel(channelDto channel.Channel) (*ent.Channel, error)
	GetChannels() ([]*ent.Channel, error)
	GetChannel(channelID uuid.UUID) (*ent.Channel, error)
	GetChannelByName(channelName string) (*ent.Channel, error)
	DeleteChannel(channelID uuid.UUID) error
	UpdateChannel(channelID uuid.UUID, channelDto channel.Channel) (*ent.Channel, error)
}

type CreateChannelRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=50"`
	DisplayName string `json:"display_name" validate:"required,min=2,max=50"`
	ImagePath   string `json:"image_path" validate:"required,min=3"`
}

func (h *Handler) CreateChannel(c echo.Context) error {
	ccr := new(CreateChannelRequest)
	if err := c.Bind(ccr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(ccr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ccDto := channel.Channel{
		Name:        ccr.Name,
		DisplayName: ccr.DisplayName,
		ImagePath:   ccr.ImagePath,
	}

	cha, err := h.Service.ChannelService.CreateChannel(ccDto)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, cha)
}

func (h *Handler) GetChannels(c echo.Context) error {
	channels, err := h.Service.ChannelService.GetChannels()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, channels)
}

func (h *Handler) GetChannel(c echo.Context) error {
	id := c.Param("id")
	cUUID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	cha, err := h.Service.ChannelService.GetChannel(cUUID)
	if err != nil {
		if err.Error() == "channel not found" {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, cha)
}

func (h *Handler) DeleteChannel(c echo.Context) error {
	id := c.Param("id")
	cUUID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	err = h.Service.ChannelService.DeleteChannel(cUUID)
	if err != nil {
		if err.Error() == "channel not found" {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusOK)
}

func (h *Handler) UpdateChannel(c echo.Context) error {
	id := c.Param("id")
	cUUID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	ccr := new(CreateChannelRequest)
	if err := c.Bind(ccr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(ccr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ccDto := channel.Channel{
		Name:        ccr.Name,
		DisplayName: ccr.DisplayName,
		ImagePath:   ccr.ImagePath,
	}

	cha, err := h.Service.ChannelService.UpdateChannel(cUUID, ccDto)
	if err != nil {
		if err.Error() == "channel not found" {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, cha)
}

func (h *Handler) GetChannelByName(c echo.Context) error {
	name := c.Param("name")
	cha, err := h.Service.ChannelService.GetChannelByName(name)
	if err != nil {
		if err.Error() == "channel not found" {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, cha)
}
