package vod

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/zibbp/ganymede/ent"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/chapter"
	"github.com/zibbp/ganymede/internal/playlist"
	"github.com/zibbp/ganymede/internal/utils"
)

// ExportMetadata represents the metadata of an exported VOD including related entities.
type ExportMetadata struct {
	Version       int                 `json:"version"`
	ExportedAt    time.Time           `json:"exported_at"`
	Video         Vod                 `json:"video"`
	Channel       channel.Channel     `json:"channel"`
	Playlists     []playlist.Playlist `json:"playlists"`
	Chapters      []chapter.Chapter   `json:"chapters"`
	MutedSegments []*ent.MutedSegment `json:"muted_segments"`
}

// ExportMetadata exports the metadata of a VOD along with its related entities.
// The exported metadata is written to a json file in the video directory.
func (s *Service) ExportMetadata(ctx context.Context, videoId uuid.UUID) error {
	vod, err := s.Store.Client.Vod.
		Query().
		Where(entVod.IDEQ(videoId)).
		WithChannel().
		WithPlaylists().
		WithChapters().
		WithMutedSegments().
		Only(ctx)
	if err != nil {
		return err
	}

	// Convert to DTOs
	var playlists []playlist.Playlist
	for _, p := range vod.Edges.Playlists {
		playlists = append(playlists, playlist.DBPlaylistToDto(p))
	}

	var chapters []chapter.Chapter
	for _, c := range vod.Edges.Chapters {
		chapters = append(chapters, chapter.DBChapterToDto(c))
	}

	// Build the export metadata
	export := &ExportMetadata{
		Version:       1,
		ExportedAt:    time.Now(),
		Video:         DBVodToDto(vod),
		Channel:       channel.DBChannelToDto(vod.Edges.Channel),
		Playlists:     playlists,
		Chapters:      chapters,
		MutedSegments: vod.Edges.MutedSegments,
	}

	// Use info path to get the directory
	dir := filepath.Dir(vod.InfoPath)
	// Non-custom file name for export
	exportPath := filepath.Join(dir, fmt.Sprintf("%s.ganymede.json", vod.ID.String()))

	err = utils.WriteJsonFile(export, exportPath)
	if err != nil {
		return fmt.Errorf("error writing export metadata: %v", err)
	}

	return nil
}
