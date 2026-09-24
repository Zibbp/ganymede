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
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/utils"
)

type conflictingLiveArchiveService struct{ ArchiveService }

func (conflictingLiveArchiveService) ArchiveLivestream(context.Context, archive.ArchiveVideoInput) (*archive.ArchiveResponse, error) {
	return nil, fmt.Errorf("create archive and enqueue first task: %w", archive.ErrActiveLiveArchive)
}

func TestArchiveLiveConflict(t *testing.T) {
	server := echo.New()
	server.Validator = &utils.CustomValidator{Validator: validator.New()}
	body := fmt.Sprintf(`{"channel_id":%q,"quality":"best"}`, uuid.New().String())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/archive/video", strings.NewReader(body))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	handler := &Handler{Service: Services{ArchiveService: conflictingLiveArchiveService{}}}
	require.NoError(t, handler.ArchiveVideo(server.NewContext(request, recorder)))
	require.Equal(t, http.StatusConflict, recorder.Code)
	var response struct {
		Success bool
		Message string
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Equal(t, archive.ErrActiveLiveArchive.Error(), response.Message)
}
