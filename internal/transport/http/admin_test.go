package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zibbp/ganymede/internal/admin"
	httpHandler "github.com/zibbp/ganymede/internal/transport/http"
)

type MockAdminService struct {
	mock.Mock
}

func (m *MockAdminService) GetStats(ctx context.Context) (admin.GetStatsResp, error) {
	args := m.Called(ctx)
	return args.Get(0).(admin.GetStatsResp), args.Error(1)
}

func (m *MockAdminService) GetInfo(ctx context.Context) (admin.InfoResp, error) {
	args := m.Called(ctx)
	return args.Get(0).(admin.InfoResp), args.Error(1)
}

func setupAdminHandler() *httpHandler.Handler {
	e := setupEcho()
	mockAdminService := new(MockAdminService)

	services := httpHandler.Services{
		AdminService: mockAdminService,
	}

	handler := &httpHandler.Handler{
		Server:  e,
		Service: services,
	}

	return handler
}

// TestGetStats is a test function for getting the ganymede stats.
func TestGetStats(t *testing.T) {
	handler := setupAdminHandler()
	e := handler.Server
	mockService := handler.Service.AdminService.(*MockAdminService)

	expected := admin.GetStatsResp{
		VodCount:     0,
		ChannelCount: 0,
	}

	mockService.On("GetStats", mock.Anything).Return(expected, nil)

	req := httptest.NewRequest(http.MethodPost, "/admin/stats", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, handler.GetStats(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		var response admin.GetStatsResp
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, expected, response)
	}

	mockService.AssertExpectations(t)
}

// TestGetInfo is a test function for getting the ganymede info.
func TestGetInfo(t *testing.T) {
	handler := setupAdminHandler()
	e := handler.Server
	mockService := handler.Service.AdminService.(*MockAdminService)

	expected := admin.InfoResp{
		CommitHash: "test",
		BuildTime:  "test",
		Uptime:     "test",
		ProgramVersions: admin.ProgramVersions{
			FFmpeg:           "test",
			TwitchDownloader: "test",
			ChatDownloader:   "test",
			Streamlink:       "test",
		},
	}

	mockService.On("GetInfo", mock.Anything).Return(expected, nil)

	req := httptest.NewRequest(http.MethodPost, "/admin/info", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, handler.GetInfo(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		var response admin.InfoResp
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, expected, response)
	}

	mockService.AssertExpectations(t)
}
