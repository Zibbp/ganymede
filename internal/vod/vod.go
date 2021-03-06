package vod

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/channel"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/utils"
	"time"
)

type Service struct {
	Store *database.Database
}

func NewService(store *database.Database) *Service {
	return &Service{Store: store}
}

type Vod struct {
	ID               uuid.UUID         `json:"id"`
	ExtID            string            `json:"ext_id"`
	Platform         utils.VodPlatform `json:"platform"`
	Type             utils.VodType     `json:"type"`
	Title            string            `json:"title"`
	Duration         int               `json:"duration"`
	Views            int               `json:"views"`
	Resolution       string            `json:"resolution"`
	Processing       bool              `json:"processing"`
	ThumbnailPath    string            `json:"thumbnail_path"`
	WebThumbnailPath string            `json:"web_thumbnail_path"`
	VideoPath        string            `json:"video_path"`
	ChatPath         string            `json:"chat_path"`
	ChatVideoPath    string            `json:"chat_video_path"`
	InfoPath         string            `json:"info_path"`
	StreamedAt       time.Time         `json:"streamed_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	CreatedAt        time.Time         `json:"created_at"`
}

func (s *Service) CreateVod(vodDto Vod, cUUID uuid.UUID) (*ent.Vod, error) {
	v, err := s.Store.Client.Vod.Create().SetID(vodDto.ID).SetChannelID(cUUID).SetExtID(vodDto.ExtID).SetPlatform(vodDto.Platform).SetType(vodDto.Type).SetTitle(vodDto.Title).SetDuration(vodDto.Duration).SetViews(vodDto.Views).SetResolution(vodDto.Resolution).SetProcessing(vodDto.Processing).SetThumbnailPath(vodDto.ThumbnailPath).SetWebThumbnailPath(vodDto.WebThumbnailPath).SetVideoPath(vodDto.VideoPath).SetChatPath(vodDto.ChatPath).SetChatVideoPath(vodDto.ChatVideoPath).SetInfoPath(vodDto.InfoPath).SetStreamedAt(vodDto.StreamedAt).Save(context.Background())
	if err != nil {
		log.Debug().Err(err).Msg("error creating vod")
		if _, ok := err.(*ent.ConstraintError); ok {
			return nil, fmt.Errorf("channel does not exist")
		}
		return nil, fmt.Errorf("error creating vod: %v", err)
	}

	return v, nil
}

func (s *Service) GetVods(c echo.Context) ([]*ent.Vod, error) {
	v, err := s.Store.Client.Vod.Query().WithChannel().Order(ent.Desc(vod.FieldStreamedAt)).All(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vods")
		return nil, fmt.Errorf("error getting vods: %v", err)
	}

	return v, nil
}

func (s *Service) GetVodsByChannel(c echo.Context, cUUID uuid.UUID) ([]*ent.Vod, error) {
	v, err := s.Store.Client.Vod.Query().Where(vod.HasChannelWith(channel.ID(cUUID))).Order(ent.Desc(vod.FieldStreamedAt)).All(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vods by channel")
		return nil, fmt.Errorf("error getting vods by channel: %v", err)
	}

	return v, nil
}

func (s *Service) GetVod(vodID uuid.UUID) (*ent.Vod, error) {
	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).Only(context.Background())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod")
		// if vod not found
		if _, ok := err.(*ent.NotFoundError); ok {
			return nil, fmt.Errorf("vod not found")
		}
		return nil, fmt.Errorf("error getting vod: %v", err)
	}

	return v, nil
}

func (s *Service) GetVodWithChannel(vodID uuid.UUID) (*ent.Vod, error) {
	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).WithChannel().Only(context.Background())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod")
		// if vod not found
		if _, ok := err.(*ent.NotFoundError); ok {
			return nil, fmt.Errorf("vod not found")
		}
		return nil, fmt.Errorf("error getting vod: %v", err)
	}

	return v, nil
}

func (s *Service) DeleteVod(c echo.Context, vodID uuid.UUID) error {
	// delete vod and queue item
	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).WithQueue().Only(c.Request().Context())
	if err != nil {
		if _, ok := err.(*ent.NotFoundError); ok {
			return fmt.Errorf("vod not found")
		}
		return fmt.Errorf("error deleting vod: %v", err)
	}
	if v.Edges.Queue != nil {
		err = s.Store.Client.Queue.DeleteOneID(v.Edges.Queue.ID).Exec(c.Request().Context())
		if err != nil {
			return fmt.Errorf("error deleting queue item: %v", err)
		}
	}

	err = s.Store.Client.Vod.DeleteOneID(vodID).Exec(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error deleting vod")
		return fmt.Errorf("error deleting vod: %v", err)
	}
	return nil
}

func (s *Service) UpdateVod(c echo.Context, vodID uuid.UUID, vodDto Vod, cUUID uuid.UUID) (*ent.Vod, error) {
	v, err := s.Store.Client.Vod.UpdateOneID(vodID).SetChannelID(cUUID).SetExtID(vodDto.ExtID).SetPlatform(vodDto.Platform).SetType(vodDto.Type).SetTitle(vodDto.Title).SetDuration(vodDto.Duration).SetViews(vodDto.Views).SetResolution(vodDto.Resolution).SetProcessing(vodDto.Processing).SetThumbnailPath(vodDto.ThumbnailPath).SetWebThumbnailPath(vodDto.WebThumbnailPath).SetVideoPath(vodDto.VideoPath).SetChatPath(vodDto.ChatPath).SetChatVideoPath(vodDto.ChatVideoPath).SetInfoPath(vodDto.InfoPath).SetStreamedAt(vodDto.StreamedAt).Save(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error updating vod")

		// if vod not found
		if _, ok := err.(*ent.NotFoundError); ok {
			return nil, fmt.Errorf("vod not found")
		}
		return nil, fmt.Errorf("error updating vod: %v", err)
	}

	return v, nil
}

func (s *Service) CheckVodExists(c echo.Context, extID string) (bool, error) {
	_, err := s.Store.Client.Vod.Query().Where(vod.ExtID(extID)).Only(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error checking vod exists")

		// if vod not found
		if _, ok := err.(*ent.NotFoundError); ok {
			return false, nil
		}
		return false, fmt.Errorf("error checking vod exists: %v", err)
	}

	return true, nil
}

func (s *Service) SearchVods(c echo.Context, term string) ([]*ent.Vod, error) {
	v, err := s.Store.Client.Vod.Query().Where(vod.TitleContainsFold(term)).Order(ent.Desc(vod.FieldStreamedAt)).All(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error searching vods")
		return nil, fmt.Errorf("error searching vods: %v", err)
	}

	return v, nil
}
