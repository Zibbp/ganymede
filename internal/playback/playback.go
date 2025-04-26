package playback

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/playback"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/utils"
)

type Service struct {
	Store *database.Database
}

func NewService(store *database.Database) *Service {
	return &Service{Store: store}
}

type GetPlaybackResp struct {
	Playback []*ent.Playback `json:"playback"`
	Data     []*GetPlayback  `json:"data"`
}

type GetPlayback struct {
	Playback *ent.Playback `json:"playback"`
	Vod      *ent.Vod      `json:"vod"`
}

var ErrorPlaybackNotFound = fmt.Errorf("playback not found")

func (s *Service) UpdateProgress(ctx context.Context, userId uuid.UUID, videoId uuid.UUID, time int) error {
	check, err := s.Store.Client.Playback.Query().Where(playback.UserID(userId)).Where(playback.VodID(videoId)).Order(playback.ByUpdatedAt(sql.OrderAsc())).First(ctx)
	if err != nil {
		if _, ok := err.(*ent.NotFoundError); ok {

			_, err = s.Store.Client.Playback.Create().SetUserID(userId).SetVodID(videoId).SetTime(time).Save(ctx)
			if err != nil {
				return fmt.Errorf("error creating playback: %v", err)
			}

			return nil
		}
		return fmt.Errorf("error checking playback: %v", err)
	}
	if check != nil {
		_, err = s.Store.Client.Playback.Update().Where(playback.UserID(userId)).Where(playback.VodID(videoId)).SetTime(time).Save(ctx)
		if err != nil {
			return fmt.Errorf("error updating playback: %v", err)
		}
	}

	return nil
}

func (s *Service) GetProgress(ctx context.Context, userId uuid.UUID, videoId uuid.UUID) (*ent.Playback, error) {
	playbackEntry, err := s.Store.Client.Playback.Query().Where(playback.UserID(userId)).Where(playback.VodID(videoId)).Order(playback.ByUpdatedAt(sql.OrderAsc())).First(ctx)
	if err != nil {
		if _, ok := err.(*ent.NotFoundError); ok {
			return nil, ErrorPlaybackNotFound
		}
		return nil, fmt.Errorf("error getting playback: %v", err)
	}

	return playbackEntry, nil
}

func (s *Service) GetAllProgress(ctx context.Context, userId uuid.UUID) ([]*ent.Playback, error) {
	playbackEntries, err := s.Store.Client.Playback.Query().Where(playback.UserID(userId)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting all playback: %v", err)
	}

	return playbackEntries, nil
}

func (s *Service) UpdateStatus(ctx context.Context, userId uuid.UUID, videoId uuid.UUID, status string) error {
	_, err := s.Store.Client.Playback.Query().Where(playback.UserID(userId)).Where(playback.VodID(videoId)).Order(playback.ByUpdatedAt(sql.OrderAsc())).First(ctx)
	if err != nil {
		if _, ok := err.(*ent.NotFoundError); ok {

			_, err = s.Store.Client.Playback.Create().SetUserID(userId).SetVodID(videoId).SetStatus(utils.PlaybackStatus(status)).Save(ctx)
			if err != nil {
				return fmt.Errorf("error creating playback: %v", err)
			}

			return nil
		}
		return fmt.Errorf("error checking playback: %v", err)
	}

	_, err = s.Store.Client.Playback.Update().Where(playback.UserID(userId)).Where(playback.VodID(videoId)).SetStatus(utils.PlaybackStatus(status)).Save(ctx)
	if err != nil {
		if _, ok := err.(*ent.NotFoundError); ok {
			return fmt.Errorf("playback not found")
		}
		return fmt.Errorf("error updating playback: %v", err)
	}

	return nil
}

func (s *Service) DeleteProgress(ctx context.Context, userId uuid.UUID, videoId uuid.UUID) error {
	_, err := s.Store.Client.Playback.Delete().Where(playback.UserID(userId)).Where(playback.VodID(videoId)).Exec(ctx)
	if err != nil {
		if _, ok := err.(*ent.NotFoundError); ok {
			return fmt.Errorf("playback not found")
		}
		return fmt.Errorf("error deleting playback: %v", err)
	}

	return nil
}

func (s *Service) GetLastPlaybacks(ctx context.Context, userId uuid.UUID, limit int) (*GetPlaybackResp, error) {
	// Fetch all playbacks for the user
	playbackEntries, err := s.Store.Client.Playback.Query().Where(playback.UserID(userId)).Where(playback.StatusEQ("in_progress")).Order(ent.Desc(playback.FieldUpdatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting last playbacks: %v", err)
	}

	var getPlayback []*GetPlayback

	var getPlaybackResp *GetPlaybackResp

	var tmpPlaybackEntries []*ent.Playback

	// Process the fetched playbacks
	for _, playbackEntry := range playbackEntries {
		vod, err := s.Store.Client.Vod.Query().Where(entVod.ID(playbackEntry.VodID)).WithChannel().Only(ctx)
		if err != nil {
			// Skip if vod not found
			continue
		}

		// Append only if VOD is found
		getPlayback = append(getPlayback, &GetPlayback{
			Playback: playbackEntry,
			Vod:      vod,
		})
		tmpPlaybackEntries = append(tmpPlaybackEntries, playbackEntry)

		// Break the loop if we've reached the required limit
		if len(getPlayback) == limit {
			break
		}
	}

	getPlaybackResp = &GetPlaybackResp{
		Playback: tmpPlaybackEntries,
		Data:     getPlayback,
	}

	return getPlaybackResp, nil
}

func (s *Service) StartPlayback(c echo.Context, videoId uuid.UUID) error {
	video, err := s.Store.Client.Vod.Query().Where(entVod.ID(videoId)).WithChannel().Only(c.Request().Context())
	if err != nil {
		return fmt.Errorf("error getting video: %v", err)
	}

	// add a view to the video
	err = s.Store.Client.Vod.UpdateOne(video).AddLocalViews(1).Exec(c.Request().Context())
	if err != nil {
		return fmt.Errorf("error adding view to video: %v", err)
	}

	return nil
}
