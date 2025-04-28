package chapter

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/zibbp/ganymede/ent"
	entChapter "github.com/zibbp/ganymede/ent/chapter"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/utils"
)

type Service struct {
	Store *database.Database
}

func NewService(store *database.Database) *Service {
	return &Service{
		Store: store,
	}
}

type Chapter struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

func (s *Service) CreateChapter(c Chapter, videoId uuid.UUID) (*ent.Chapter, error) {
	dbVideo, err := s.Store.Client.Vod.Query().Where(vod.ID(videoId)).First(context.Background())
	if err != nil {
		if _, ok := err.(*ent.NotFoundError); ok {
			return nil, fmt.Errorf("video not found")
		}
		return nil, fmt.Errorf("error getting video: %v", err)
	}

	dbChapter, err := s.Store.Client.Chapter.Create().SetType(c.Type).SetTitle(c.Title).SetStart(c.Start).SetEnd(c.End).SetVod(dbVideo).Save(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error creating chapter: %v", err)
	}

	return dbChapter, nil
}

func (s *Service) UpdateChapter(c Chapter, chapterId uuid.UUID) (*ent.Chapter, error) {
	dbChapter, err := s.Store.Client.Chapter.UpdateOneID(chapterId).SetType(c.Type).SetTitle(c.Title).SetStart(c.Start).SetEnd(c.End).Save(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error updating chapter: %v", err)
	}

	return dbChapter, nil
}

func (s *Service) GetVideoChapters(videoId uuid.UUID) ([]*ent.Chapter, error) {
	chapters, err := s.Store.Client.Chapter.Query().Where(entChapter.HasVodWith(vod.ID(videoId))).All(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error getting chapters: %v", err)
	}

	return chapters, nil
}

func (s *Service) CreateWebVtt(chapters []*ent.Chapter) (string, error) {
	webVtt := "WEBVTT\n\n"

	for _, chapter := range chapters {
		webVtt += fmt.Sprintf("%s --> %s\n%s\n\n", utils.SecondsToHHMMSS(chapter.Start), utils.SecondsToHHMMSS(chapter.End), chapter.Title)
	}

	return webVtt, nil
}
