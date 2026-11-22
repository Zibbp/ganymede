package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/utils"
)

type lowStorageArchiveService struct{ ArchiveService }

func (lowStorageArchiveService) ArchiveLivestream(context.Context, archive.ArchiveVideoInput) (*archive.ArchiveResponse, error) {
	return nil, fmt.Errorf("create archive and enqueue first task: %w", archive.ErrTempDirLowSpace)
}

func (lowStorageArchiveService) ArchiveVideo(context.Context, archive.ArchiveVideoInput) (*archive.ArchiveResponse, error) {
	return nil, fmt.Errorf("create archive and enqueue first task: %w", archive.ErrTempDirLowSpace)
}

func TestArchiveLowStorage(t *testing.T) {
	server := echo.New()
	server.Validator = &utils.CustomValidator{Validator: validator.New()}
	handler := &Handler{Service: Services{ArchiveService: lowStorageArchiveService{}}}

	t.Run("livestream", func(t *testing.T) {
		body := `{"channel_id":"00000000-0000-0000-0000-000000000000","quality":"best"}`
		request := httptest.NewRequest(http.MethodPost, "/api/v1/archive/video", strings.NewReader(body))
		request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recorder := httptest.NewRecorder()
		require.NoError(t, handler.ArchiveVideo(server.NewContext(request, recorder)))
		require.Equal(t, http.StatusInsufficientStorage, recorder.Code)
		var response struct {
			Success bool
			Message string
		}
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
		require.False(t, response.Success)
		require.Contains(t, response.Message, archive.ErrTempDirLowSpace.Error())
	})

	t.Run("video", func(t *testing.T) {
		body := `{"video_id":"123456789","quality":"best"}`
		request := httptest.NewRequest(http.MethodPost, "/api/v1/archive/video", strings.NewReader(body))
		request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		recorder := httptest.NewRecorder()
		require.NoError(t, handler.ArchiveVideo(server.NewContext(request, recorder)))
		require.Equal(t, http.StatusInsufficientStorage, recorder.Code)
		var response struct {
			Success bool
			Message string
		}
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
		require.False(t, response.Success)
		require.Contains(t, response.Message, archive.ErrTempDirLowSpace.Error())
	})
}
