package http

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/playlist"
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

// CreatePlaylist godoc
//
//	@Summary		Create playlist
//	@Description	Create playlist
//	@Tags			Playlist
//	@Accept			json
//	@Produce		json
//	@Param			playlist	body		CreatePlaylistRequest	true	"playlist"
//	@Success		200			{object}	ent.Playlist
//	@Failure		400			{object}	utils.ErrorResponse
//	@Failure		500			{object}	utils.ErrorResponse
//	@Router			/playlist [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) CreatePlaylist(c echo.Context) error {
	cpr := new(CreatePlaylistRequest)
	if err := c.Bind(cpr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(cpr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	playlistDto := playlist.Playlist{
		Name:        cpr.Name,
		Description: cpr.Description,
	}
	createdPlaylist, err := h.Service.PlaylistService.CreatePlaylist(c, playlistDto)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, createdPlaylist, "created playlist")

}

// AddVodToPlaylist godoc
//
//	@Summary		Add vod to playlist
//	@Description	Add vod to playlist
//	@Tags			Playlist
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"playlist id"
//	@Param			vod	body		AddVodToPlaylistRequest	true	"vod"
//	@Success		200	{object}	string
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/playlist/{id} [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) AddVodToPlaylist(c echo.Context) error {
	pID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid playlist id")
	}
	avtpr := new(AddVodToPlaylistRequest)
	if err := c.Bind(avtpr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(avtpr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	vID, err := uuid.Parse(avtpr.VodID)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid vod id")
	}
	err = h.Service.PlaylistService.AddVodToPlaylist(c, pID, vID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, "", "video added to playlist")
}

// GetPlaylists godoc
//
//	@Summary		Get playlists
//	@Description	Get playlists
//	@Tags			Playlist
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	[]ent.Playlist
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/playlist [get]
func (h *Handler) GetPlaylists(c echo.Context) error {
	playlists, err := h.Service.PlaylistService.GetPlaylists(c)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, playlists, "playlists")
}

// GetPlaylist godoc
//
//	@Summary		Get playlist
//	@Description	Get playlist
//	@Tags			Playlist
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"playlist id"
//	@Success		200	{object}	ent.Playlist
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/playlist/{id} [get]
func (h *Handler) GetPlaylist(c echo.Context) error {
	pID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid playlist id")
	}
	rPlaylist, err := h.Service.PlaylistService.GetPlaylist(c, pID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, rPlaylist, "playlist")
}

// UpdatePlaylist godoc
//
//	@Summary		Update playlist
//	@Description	Update playlist
//	@Tags			Playlist
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string					true	"playlist id"
//	@Param			playlist	body		CreatePlaylistRequest	true	"playlist"
//	@Success		200			{object}	ent.Playlist
//	@Failure		400			{object}	utils.ErrorResponse
//	@Failure		500			{object}	utils.ErrorResponse
//	@Router			/playlist/{id} [put]
//	@Security		ApiKeyCookieAuth
func (h *Handler) UpdatePlaylist(c echo.Context) error {
	pID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid playlist id")
	}
	upr := new(CreatePlaylistRequest)
	if err := c.Bind(upr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(upr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	playlistDto := playlist.Playlist{
		Name:        upr.Name,
		Description: upr.Description,
	}
	updatedPlaylist, err := h.Service.PlaylistService.UpdatePlaylist(c, pID, playlistDto)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, updatedPlaylist, "playlist updated")
}

// DeletePlaylist godoc
//
//	@Summary		Delete playlist
//	@Description	Delete playlist
//	@Tags			Playlist
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"playlist id"
//	@Success		200	{object}	string
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/playlist/{id} [delete]
//	@Security		ApiKeyCookieAuth
func (h *Handler) DeletePlaylist(c echo.Context) error {
	pID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid playlist id")
	}
	err = h.Service.PlaylistService.DeletePlaylist(c, pID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, "", "playlist deleted")
}

// DeleteVodFromPlaylist godoc
//
//	@Summary		Delete vod from playlist
//	@Description	Delete vod from playlist
//	@Tags			Playlist
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"playlist id"
//	@Param			vod	body		AddVodToPlaylistRequest	true	"vod"
//	@Success		200	{object}	string
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/playlist/{id}/vod [delete]
//	@Security		ApiKeyCookieAuth
func (h *Handler) DeleteVodFromPlaylist(c echo.Context) error {
	pID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid playlist id")
	}
	avtpr := new(AddVodToPlaylistRequest)
	if err := c.Bind(avtpr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(avtpr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	vID, err := uuid.Parse(avtpr.VodID)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid vod id")
	}
	err = h.Service.PlaylistService.DeleteVodFromPlaylist(c, pID, vID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, "", "video removed from playlist")
}
