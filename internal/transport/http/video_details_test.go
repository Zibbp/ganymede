package http

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/utils"
)

type videoDetailsService struct {
	VodService
	video *ent.Vod
}

func (s videoDetailsService) GetVod(context.Context, uuid.UUID, bool, bool, bool, bool) (*ent.Vod, error) {
	return s.video, nil
}

func TestVideoDetailsLivePreviewAvailability(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		status utils.ArchiveStatus
		output string
		want   bool
	}{
		{"missing playlist", utils.ArchiveRunning, "missing", false},
		{"empty playlist", utils.ArchiveRunning, "empty", false},
		{"directory instead of playlist", utils.ArchiveRunning, "directory", false},
		{"recording with playlist", utils.ArchiveRunning, "playlist", true},
		{"finalizing with playlist", utils.ArchiveFinalizing, "playlist", true},
		{"completed with leftover playlist", utils.ArchiveCompleted, "playlist", false},
		{"failed with leftover playlist", utils.ArchiveFailed, "playlist", false},
		{"no preview directory", utils.ArchiveRunning, "no-path", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			video := &ent.Vod{ID: uuid.New(), ExtID: "123", Type: utils.Live, Status: tc.status, TmpVideoHlsPath: t.TempDir()}
			path := filepath.Join(video.TmpVideoHlsPath, "123-video.m3u8")
			switch tc.output {
			case "playlist":
				require.NoError(t, os.WriteFile(path, []byte("#EXTM3U\n#EXTINF:10,\n123_segment000001.ts\n"), 0600))
			case "empty":
				require.NoError(t, os.WriteFile(path, nil, 0600))
			case "directory":
				require.NoError(t, os.Mkdir(path, 0700))
			case "no-path":
				video.TmpVideoHlsPath = ""
			}
			require.Equal(t, tc.want, requestVideoDetails(t, video)["live_preview_available"])
		})
	}
}

func TestVideoDetailsDetectsNewLivePreview(t *testing.T) {
	t.Parallel()
	video := &ent.Vod{ID: uuid.New(), ExtID: "123", Type: utils.Live, Status: utils.ArchiveRunning, TmpVideoHlsPath: t.TempDir()}
	require.Equal(t, false, requestVideoDetails(t, video)["live_preview_available"])
	path := filepath.Join(video.TmpVideoHlsPath, "123-video.m3u8")
	require.NoError(t, os.WriteFile(path, []byte("#EXTM3U\n#EXTINF:10,\n123_segment000001.ts\n"), 0600))
	require.Equal(t, true, requestVideoDetails(t, video)["live_preview_available"])
	require.NoError(t, os.Remove(path))
	require.Equal(t, false, requestVideoDetails(t, video)["live_preview_available"])
}

func requestVideoDetails(t *testing.T, video *ent.Vod) map[string]any {
	t.Helper()
	handler := &Handler{Service: Services{VodService: videoDetailsService{video: video}}}
	recorder := httptest.NewRecorder()
	c := echo.New().NewContext(httptest.NewRequest("GET", "/api/v1/vod/"+video.ID.String(), nil), recorder)
	c.SetParamNames("id")
	c.SetParamValues(video.ID.String())
	require.NoError(t, handler.GetVod(c))
	require.Equal(t, 200, recorder.Code)
	var response struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, video.ID.String(), response.Data["id"])
	require.Equal(t, string(video.Status), response.Data["status"])
	return response.Data
}
