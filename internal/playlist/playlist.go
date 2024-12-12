package playlist

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/multistreaminfo"
	"github.com/zibbp/ganymede/ent/playlist"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/database"
)

type Service struct {
	Store *database.Database
}

type Playlist struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ImagePath   string    `json:"image_path"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

func NewService(store *database.Database) *Service {
	return &Service{Store: store}
}

func (s *Service) CreatePlaylist(c echo.Context, playlistDto Playlist) (*ent.Playlist, error) {
	playlistEntry, err := s.Store.Client.Playlist.Create().SetName(playlistDto.Name).SetDescription(playlistDto.Description).Save(c.Request().Context())
	if err != nil {
		if _, ok := err.(*ent.ConstraintError); ok {
			return nil, fmt.Errorf("playlist already exists")
		}
		return nil, fmt.Errorf("error creating playlist: %v", err)
	}

	return playlistEntry, nil
}

func (s *Service) AddVodToPlaylist(c echo.Context, playlistID uuid.UUID, vodID uuid.UUID) error {
	_, err := s.Store.Client.Playlist.Query().Where(playlist.ID(playlistID)).Only(c.Request().Context())
	if err != nil {
		return fmt.Errorf("playlist not found")
	}

	_, err = s.Store.Client.Playlist.UpdateOneID(playlistID).AddVodIDs(vodID).Save(c.Request().Context())
	if err != nil {
		if _, ok := err.(*ent.ConstraintError); ok {
			return fmt.Errorf("vod already exists in playlist")
		}
		return fmt.Errorf("error adding vod to playlist: %v", err)
	}

	return nil
}

func (s *Service) GetPlaylists(c echo.Context) ([]*ent.Playlist, error) {
	playlists, err := s.Store.Client.Playlist.Query().Order(ent.Desc(playlist.FieldCreatedAt)).All(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error getting playlists: %v", err)
	}

	return playlists, nil
}

func (s *Service) GetPlaylist(c echo.Context, playlistID uuid.UUID, withMultistreamInfo bool) (*ent.Playlist, error) {
	playlistQuery := s.Store.Client.Playlist.Query().Where(playlist.ID(playlistID)).WithVods(func(q *ent.VodQuery) {
		q.WithChannel()
	})
	if withMultistreamInfo {
		playlistQuery.WithMultistreamInfo(func(miq *ent.MultistreamInfoQuery) { miq.WithVod() })
	}
	rPlaylist, err := playlistQuery.Order(ent.Desc(playlist.FieldCreatedAt)).Only(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error getting playlist: %v", err)
	}
	// Order VODs by date streamed
	tmpVods := rPlaylist.Edges.Vods
	sort.Slice(tmpVods, func(i, j int) bool {
		return tmpVods[i].StreamedAt.After(tmpVods[j].StreamedAt)
	})
	rPlaylist.Edges.Vods = tmpVods

	return rPlaylist, nil
}

func (s *Service) UpdatePlaylist(c echo.Context, playlistID uuid.UUID, playlistDto Playlist) (*ent.Playlist, error) {
	_, err := s.Store.Client.Playlist.Query().Where(playlist.ID(playlistID)).Only(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("playlist not found")
	}

	uPlaylist, err := s.Store.Client.Playlist.UpdateOneID(playlistID).SetName(playlistDto.Name).SetDescription(playlistDto.Description).Save(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error updating playlist: %v", err)
	}

	return uPlaylist, nil
}

func (s *Service) DeletePlaylist(c echo.Context, playlistID uuid.UUID) error {
	_, err := s.Store.Client.Playlist.Query().Where(playlist.ID(playlistID)).Only(c.Request().Context())
	if err != nil {
		return fmt.Errorf("playlist not found")
	}

	err = s.Store.Client.Playlist.DeleteOneID(playlistID).Exec(c.Request().Context())
	if err != nil {
		return fmt.Errorf("error deleting playlist: %v", err)
	}

	return nil
}

func (s *Service) DeleteVodFromPlaylist(c echo.Context, playlistID uuid.UUID, vodID uuid.UUID) error {
	_, err := s.Store.Client.Playlist.Query().Where(playlist.ID(playlistID)).Only(c.Request().Context())
	if err != nil {
		return fmt.Errorf("playlist not found")
	}

	_, err = s.Store.Client.Playlist.UpdateOneID(playlistID).RemoveVodIDs(vodID).Save(c.Request().Context())
	if err != nil {
		return fmt.Errorf("error deleting vod from playlist: %v", err)
	}

	return nil
}

func (s *Service) SetVodDelayOnPlaylist(c echo.Context, playlistID uuid.UUID, vodID uuid.UUID, delayMs int) error {
	dbPlaylist, err := s.Store.Client.Playlist.Query().Where(playlist.ID(playlistID)).WithVods().Only(c.Request().Context())
	if err != nil {
		return fmt.Errorf("playlist not found")
	}
	// If one day, we need to store more than just the delay, we should remove the deletion here
	if delayMs == 0 {
		return s.deleteMultistreamInfo(c, playlistID, vodID)
	}
	// Check if vod exists in playlist before creating new data
	found := false
	for _, vod := range dbPlaylist.Edges.Vods {
		if vod.ID == vodID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("vod not found in playlist")
	}
	dbMultistreamInfo, err := s.Store.Client.MultistreamInfo.Query().Where(
		multistreaminfo.And(
			multistreaminfo.HasPlaylistWith(playlist.ID(playlistID)),
			multistreaminfo.HasVodWith(vod.ID(vodID)),
		),
	).Only(c.Request().Context())
	if err != nil && ent.IsNotFound(err) {
		_, err = s.Store.Client.MultistreamInfo.Create().SetDelayMs(delayMs).SetPlaylistID(playlistID).SetVodID(vodID).Save(c.Request().Context())
		if err != nil {
			return fmt.Errorf("error creating multistream info: %v", err)
		}
	} else {
		_, err = s.Store.Client.MultistreamInfo.UpdateOne(dbMultistreamInfo).SetDelayMs(delayMs).Save(c.Request().Context())
		if err != nil {
			return fmt.Errorf("error updating multistream info: %v", err)
		}
	}
	return nil
}
func (s *Service) deleteMultistreamInfo(c echo.Context, playlistID uuid.UUID, vodID uuid.UUID) error {
	_, err := s.Store.Client.MultistreamInfo.Delete().Where(
		multistreaminfo.And(
			multistreaminfo.HasPlaylistWith(playlist.ID(playlistID)),
			multistreaminfo.HasVodWith(vod.ID(vodID)),
		),
	).Exec(c.Request().Context())
	if err != nil {
		return fmt.Errorf("error deleting multistream info: %v", err)
	}
	return nil
}
