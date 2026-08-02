package tasks

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/utils"
)

type testMovieNFO struct {
	Title     string   `xml:"title"`
	Premiered string   `xml:"premiered"`
	Studio    string   `xml:"studio"`
	Genres    []string `xml:"genre"`
	UniqueID  struct {
		Type    string `xml:"type,attr"`
		Default bool   `xml:"default,attr"`
		Value   string `xml:",chardata"`
	} `xml:"uniqueid"`
}

func TestEnsureVideoNFOCreatesCoreMetadata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mediaPath := filepath.Join(dir, "vod.mp4")
	require.NoError(t, os.WriteFile(mediaPath, []byte("video"), 0o644))

	video := &ent.Vod{
		ID:         uuid.New(),
		Title:      "A stream title",
		VideoPath:  mediaPath,
		Platform:   utils.PlatformTwitch,
		ExtID:      "987654321",
		StreamedAt: time.Date(2026, time.July, 30, 19, 15, 0, 0, time.UTC),
		Edges: ent.VodEdges{
			Channel: &ent.Channel{
				Name:        "streamer",
				DisplayName: "Streamer",
			},
			Chapters: []*ent.Chapter{
				{Title: " Just Chatting "},
				{Title: "Games"},
				{Title: "just chatting"},
				{Title: "  "},
			},
		},
	}

	require.NoError(t, ensureVideoNFO(zerolog.Nop(), video))

	data, err := os.ReadFile(filepath.Join(dir, "vod.nfo"))
	require.NoError(t, err)
	var got testMovieNFO
	require.NoError(t, xml.Unmarshal(data, &got))
	require.Equal(t, "A stream title", got.Title)
	require.Equal(t, "2026-07-30", got.Premiered)
	require.Equal(t, "Streamer", got.Studio)
	require.Equal(t, []string{"Just Chatting", "Games"}, got.Genres)
	require.Equal(t, "twitch", got.UniqueID.Type)
	require.True(t, got.UniqueID.Default)
	require.Equal(t, "987654321", got.UniqueID.Value)
}

func TestEnsureVideoNFOSupportsHLSAndChannelNameFallback(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mediaPath := filepath.Join(dir, "index.m3u8")
	require.NoError(t, os.WriteFile(mediaPath, []byte("#EXTM3U"), 0o644))
	video := &ent.Vod{
		ID:        uuid.New(),
		Title:     "HLS archive",
		VideoPath: mediaPath,
		Edges: ent.VodEdges{
			Channel: &ent.Channel{Name: "channel_login"},
		},
	}

	require.NoError(t, ensureVideoNFO(zerolog.Nop(), video))

	data, err := os.ReadFile(filepath.Join(dir, "index.nfo"))
	require.NoError(t, err)
	var got testMovieNFO
	require.NoError(t, xml.Unmarshal(data, &got))
	require.Equal(t, "channel_login", got.Studio)
}

func TestEnsureVideoNFOPreservesExistingSidecar(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mediaPath := filepath.Join(dir, "vod.mp4")
	nfoPath := filepath.Join(dir, "vod.nfo")
	require.NoError(t, os.WriteFile(mediaPath, []byte("video"), 0o644))
	require.NoError(t, os.WriteFile(nfoPath, []byte("user-authored"), 0o600))
	video := &ent.Vod{
		ID:        uuid.New(),
		Title:     "Database title",
		VideoPath: mediaPath,
		Edges: ent.VodEdges{
			Channel: &ent.Channel{Name: "streamer"},
		},
	}

	require.NoError(t, ensureVideoNFO(zerolog.Nop(), video))

	data, err := os.ReadFile(nfoPath)
	require.NoError(t, err)
	require.Equal(t, "user-authored", string(data))
}

func TestEnsureVideoNFOSkipsMissingMedia(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	video := &ent.Vod{
		ID:        uuid.New(),
		Title:     "Missing archive",
		VideoPath: filepath.Join(dir, "missing.mp4"),
	}

	require.NoError(t, ensureVideoNFO(zerolog.Nop(), video))
	_, err := os.Stat(filepath.Join(dir, "missing.nfo"))
	require.ErrorIs(t, err, os.ErrNotExist)
}
