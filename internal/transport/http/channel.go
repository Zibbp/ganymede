package http

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/channel"
	"net/http"
)

type ChannelService interface {
	CreateChannel(c echo.Context, channelDto channel.Channel) (*ent.Channel, error)
	GetChannels(c echo.Context) ([]*ent.Channel, error)
	GetChannel(c echo.Context, channelID uuid.UUID) (*ent.Channel, error)
	DeleteChannel(c echo.Context, channelID uuid.UUID) error
	UpdateChannel(c echo.Context, channelID uuid.UUID, channelDto channel.Channel) (*ent.Channel, error)
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

	cha, err := h.Service.ChannelService.CreateChannel(c, ccDto)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, cha)
}

func (h *Handler) GetChannels(c echo.Context) error {
	channels, err := h.Service.ChannelService.GetChannels(c)
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
	cha, err := h.Service.ChannelService.GetChannel(c, cUUID)
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
	err = h.Service.ChannelService.DeleteChannel(c, cUUID)
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

	cha, err := h.Service.ChannelService.UpdateChannel(c, cUUID, ccDto)
	if err != nil {
		if err.Error() == "channel not found" {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, cha)
}
