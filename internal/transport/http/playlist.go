package http

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/playlist"
	"net/http"
)

type PlaylistService interface {
	CreatePlaylist(c echo.Context, playlistDto playlist.Playlist) (*ent.Playlist, error)
	AddVodToPlaylist(c echo.Context, playlistID uuid.UUID, vodID uuid.UUID) error
	GetPlaylists(c echo.Context) ([]*ent.Playlist, error)
	GetPlaylist(c echo.Context, playlistID uuid.UUID) (*ent.Playlist, error)
	UpdatePlaylist(c echo.Context, playlistID uuid.UUID, playlistDto playlist.Playlist) (*ent.Playlist, error)
	DeletePlaylist(c echo.Context, playlistID uuid.UUID) error
	DeleteVodFromPlaylist(c echo.Context, playlistID uuid.UUID, vodID uuid.UUID) error
}

type CreatePlaylistRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

type AddVodToPlaylistRequest struct {
	VodID string `json:"vod_id" validate:"required"`
}

func (h *Handler) CreatePlaylist(c echo.Context) error {
	cpr := new(CreatePlaylistRequest)
	if err := c.Bind(cpr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(cpr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	playlistDto := playlist.Playlist{
		Name:        cpr.Name,
		Description: cpr.Description,
	}
	createdPlaylist, err := h.Service.PlaylistService.CreatePlaylist(c, playlistDto)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, createdPlaylist)

}

func (h *Handler) AddVodToPlaylist(c echo.Context) error {
	pID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid playlist id")
	}
	avtpr := new(AddVodToPlaylistRequest)
	if err := c.Bind(avtpr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(avtpr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	vID, err := uuid.Parse(avtpr.VodID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid vod id")
	}
	err = h.Service.PlaylistService.AddVodToPlaylist(c, pID, vID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, "ok")
}

func (h *Handler) GetPlaylists(c echo.Context) error {
	playlists, err := h.Service.PlaylistService.GetPlaylists(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, playlists)
}

func (h *Handler) GetPlaylist(c echo.Context) error {
	pID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid playlist id")
	}
	rPlaylist, err := h.Service.PlaylistService.GetPlaylist(c, pID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, rPlaylist)
}

func (h *Handler) UpdatePlaylist(c echo.Context) error {
	pID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid playlist id")
	}
	upr := new(CreatePlaylistRequest)
	if err := c.Bind(upr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(upr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	playlistDto := playlist.Playlist{
		Name:        upr.Name,
		Description: upr.Description,
	}
	updatedPlaylist, err := h.Service.PlaylistService.UpdatePlaylist(c, pID, playlistDto)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, updatedPlaylist)
}

func (h *Handler) DeletePlaylist(c echo.Context) error {
	pID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid playlist id")
	}
	err = h.Service.PlaylistService.DeletePlaylist(c, pID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, "ok")
}

func (h *Handler) DeleteVodFromPlaylist(c echo.Context) error {
	pID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid playlist id")
	}
	avtpr := new(AddVodToPlaylistRequest)
	if err := c.Bind(avtpr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(avtpr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	vID, err := uuid.Parse(avtpr.VodID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid vod id")
	}
	err = h.Service.PlaylistService.DeleteVodFromPlaylist(c, pID, vID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, "ok")
}
