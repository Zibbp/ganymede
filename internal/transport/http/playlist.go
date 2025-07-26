package http

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/playlist"
	"github.com/zibbp/ganymede/internal/utils"
)

type PlaylistService interface {
	CreatePlaylist(ctx context.Context, playlistDto playlist.Playlist) (*ent.Playlist, error)
	AddVodToPlaylist(ctx context.Context, playlistID uuid.UUID, vodID uuid.UUID) error
	GetPlaylists(ctx context.Context) ([]*ent.Playlist, error)
	GetPlaylist(ctx context.Context, playlistID uuid.UUID, withMultistreamInfo bool) (*ent.Playlist, error)
	UpdatePlaylist(ctx context.Context, playlistID uuid.UUID, playlistDto playlist.Playlist) (*ent.Playlist, error)
	SetVodDelayOnPlaylist(ctx context.Context, playlistID uuid.UUID, vodId uuid.UUID, delayMs int) error
	DeletePlaylist(ctx context.Context, playlistID uuid.UUID) error
	DeleteVodFromPlaylist(ctx context.Context, playlistID uuid.UUID, vodID uuid.UUID) error
	SetPlaylistRules(ctx context.Context, playlistID uuid.UUID, ruleGroups []playlist.RuleGroupInput) ([]*ent.PlaylistRuleGroup, error)
	GetPlaylistRules(ctx context.Context, playlistID uuid.UUID) ([]*ent.PlaylistRuleGroup, error)
	TestPlaylistRules(ctx context.Context, playlistID uuid.UUID, videoID uuid.UUID) (bool, error)
}

type CreatePlaylistRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

type AddVodToPlaylistRequest struct {
	VodID string `json:"vod_id" validate:"required"`
}

type SetVodDelayPlaylistRequest struct {
	VodID   string `json:"vod_id" validate:"required"`
	DelayMs int    `json:"delay_ms"`
}

// SetPlaylistRulesRequest defines the structure for setting playlist rules.
// Also update enums in utils/enums.go if necessary.
type SetPlaylistRulesRequest struct {
	RuleGroups []struct {
		Operator string `json:"operator" validate:"required,oneof=AND OR"`
		Position int    `json:"position"`
		Rules    []struct {
			Name     string                     `json:"name"`
			Field    utils.PlaylistRuleField    `json:"field" validate:"required,oneof=title category type platform channel_name"`
			Operator utils.PlaylistRuleOperator `json:"operator" validate:"required,oneof=equals contains regex"`
			Value    string                     `json:"value" validate:"required"`
			Position int                        `json:"position"`
			Enabled  bool                       `json:"enabled"`
		} `json:"rules" validate:"required,dive"`
	} `json:"rule_groups" validate:"required,dive"`
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
	createdPlaylist, err := h.Service.PlaylistService.CreatePlaylist(c.Request().Context(), playlistDto)
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
	err = h.Service.PlaylistService.AddVodToPlaylist(c.Request().Context(), pID, vID)
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
	playlists, err := h.Service.PlaylistService.GetPlaylists(c.Request().Context())
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
//	@Param			id						path		string	true	"playlist id"
//	@Param			with_multistream_info	query		boolean	false	"include multistream info"
//	@Success		200						{object}	ent.Playlist
//	@Failure		400						{object}	utils.ErrorResponse
//	@Failure		500						{object}	utils.ErrorResponse
//	@Router			/playlist/{id} [get]
func (h *Handler) GetPlaylist(c echo.Context) error {
	pID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid playlist id")
	}
	rPlaylist, err := h.Service.PlaylistService.GetPlaylist(c.Request().Context(), pID, c.QueryParams().Get("with_multistream_info") == "true")
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
	updatedPlaylist, err := h.Service.PlaylistService.UpdatePlaylist(c.Request().Context(), pID, playlistDto)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, updatedPlaylist, "playlist updated")
}

// SetVodDelayOnPlaylistMultistream godoc
//
//	@Summary		Set delay of vod in playlist for multistream
//	@Description	Set delay of vod in playlist for multistream
//	@Tags			Playlist
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"playlist id"
//	@Param			delay	body		SetVodDelayPlaylistRequest	true	"delay"
//	@Success		200		{object}	string
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/playlist/{id} [put]
//	@Security		ApiKeyCookieAuth
func (h *Handler) SetVodDelayOnPlaylistMultistream(c echo.Context) error {
	pID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid playlist id")
	}
	svdpr := new(SetVodDelayPlaylistRequest)
	if err := c.Bind(svdpr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(svdpr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	vID, err := uuid.Parse(svdpr.VodID)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid vod id")
	}
	err = h.Service.PlaylistService.SetVodDelayOnPlaylist(c.Request().Context(), pID, vID, svdpr.DelayMs)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, "", "ok")
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
	err = h.Service.PlaylistService.DeletePlaylist(c.Request().Context(), pID)
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
	err = h.Service.PlaylistService.DeleteVodFromPlaylist(c.Request().Context(), pID, vID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, "", "video removed from playlist")
}

// SetPlaylistRules godoc
//
//	@Summary		Set playlist rules
//	@Description	Set rules for a playlist this will delete all existing rules and set new ones
//	@Tags			Playlist
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"playlist id"
//	@Param			rules	body		SetPlaylistRulesRequest	true	"rules"
//	@Success		200	{object}	string
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/playlist/{id}/rules [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) SetPlaylistRules(c echo.Context) error {
	pID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid playlist id")
	}
	var req SetPlaylistRulesRequest
	if err := c.Bind(&req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(&req); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	ruleGroups := make([]playlist.RuleGroupInput, len(req.RuleGroups))
	for i, g := range req.RuleGroups {
		rules := make([]playlist.RuleInput, len(g.Rules))
		for j, r := range g.Rules {
			rules[j] = playlist.RuleInput{
				Name:     r.Name,
				Field:    r.Field,
				Operator: r.Operator,
				Value:    r.Value,
				Position: r.Position,
				Enabled:  r.Enabled,
			}
		}
		ruleGroups[i] = playlist.RuleGroupInput{
			Operator: g.Operator,
			Position: g.Position,
			Rules:    rules,
		}
	}
	createdGroups, err := h.Service.PlaylistService.SetPlaylistRules(c.Request().Context(),
		pID, ruleGroups)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, createdGroups, "playlist rules set")
}

// GetPlaylistRules godoc
//
//	@Summary		Get playlist rules
//	@Description	Get rules for a playlist
//	@Tags			Playlist
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"playlist id"
//	@Success		200	{object}	[]ent.PlaylistRuleGroup
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/playlist/{id}/rules [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetPlaylistRules(c echo.Context) error {
	pID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid playlist id")
	}
	rules, err := h.Service.PlaylistService.GetPlaylistRules(c.Request().Context(), pID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, rules, "playlist rules")
}

// TestPlaylistRules godoc
//
//	@Summary		Test playlist rules
//	@Description	Test rules for a playlist against a video id
//	@Tags			Playlist
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"playlist id"
//	@Param			video_id	query	string					true	"video id"
//	@Success		200	{object}	bool
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/playlist/{id}/rules [put]
//	@Security		ApiKeyCookieAuth
func (h *Handler) TestPlaylistRules(c echo.Context) error {
	pID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid playlist id")
	}

	videoID, err := uuid.Parse(c.QueryParam("video_id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, "invalid video id")
	}

	result, err := h.Service.PlaylistService.TestPlaylistRules(c.Request().Context(), pID, videoID)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, result, "playlist rules test result")
}
