package nfo

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSidecarPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mediaPath string
		want      string
		wantError bool
	}{
		{
			name:      "MP4",
			mediaPath: "/archive/channel/video.mp4",
			want:      "/archive/channel/video.nfo",
		},
		{
			name:      "HLS playlist",
			mediaPath: "/archive/channel/video.m3u8",
			want:      "/archive/channel/video.nfo",
		},
		{
			name:      "case insensitive",
			mediaPath: "/archive/channel/video.MP4",
			want:      "/archive/channel/video.nfo",
		},
		{
			name:      "unsupported",
			mediaPath: "/archive/channel/video.mkv",
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := SidecarPath(test.mediaPath)
			if test.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestMarshalMovie(t *testing.T) {
	t.Parallel()

	data, err := MarshalMovie(MovieMetadata{
		Title:      `A stream & "friends"`,
		Premiered:  time.Date(2026, time.July, 30, 23, 30, 0, 0, time.FixedZone("UTC-2", -2*60*60)),
		Studio:     "Streamer",
		Genres:     []string{"Just Chatting", "Games"},
		Platform:   "Twitch",
		ExternalID: "123456789",
	})
	require.NoError(t, err)
	require.Contains(t, string(data), "&amp;")

	var got movie
	require.NoError(t, xml.Unmarshal(data, &got))
	require.Equal(t, "A stream & \"friends\"", got.Title)
	require.Equal(t, "2026-07-31", got.Premiered)
	require.Equal(t, "Streamer", got.Studio)
	require.Equal(t, []string{"Just Chatting", "Games"}, got.Genres)
	require.NotNil(t, got.UniqueID)
	require.Equal(t, "twitch", got.UniqueID.Type)
	require.True(t, got.UniqueID.Default)
	require.Equal(t, "123456789", got.UniqueID.Value)
}

func TestCreateMovieIfMissingPreservesExistingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "video.nfo")
	require.NoError(t, os.WriteFile(path, []byte("custom metadata"), 0o600))

	created, err := CreateMovieIfMissing(path, MovieMetadata{Title: "Replacement"})
	require.NoError(t, err)
	require.False(t, created)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "custom metadata", string(data))
}

func TestCreateMovieIfMissingCreatesSidecarOnce(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "video.nfo")
	metadata := MovieMetadata{Title: "New archive"}

	created, err := CreateMovieIfMissing(path, metadata)
	require.NoError(t, err)
	require.True(t, created)

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm())

	want, err := MarshalMovie(metadata)
	require.NoError(t, err)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, want, got)

	created, err = CreateMovieIfMissing(path, MovieMetadata{Title: "Different"})
	require.NoError(t, err)
	require.False(t, created)
	got, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
