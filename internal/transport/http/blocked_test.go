package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zibbp/ganymede/ent"
	httpHandler "github.com/zibbp/ganymede/internal/transport/http"
)

type MockBlockedVideoService struct {
	mock.Mock
}

func (m *MockBlockedVideoService) IsVideoBlocked(ctx context.Context, id string) (bool, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockBlockedVideoService) CreateBlockedVideo(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockBlockedVideoService) DeleteBlockedVideo(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockBlockedVideoService) GetBlockedVideos(ctx context.Context) ([]*ent.BlockedVideos, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*ent.BlockedVideos), args.Error(1)
}

func setupBlockedVideoHandler() *httpHandler.Handler {
	e := setupEcho()

	MockBlockedVideoService := new(MockBlockedVideoService)

	services := httpHandler.Services{
		BlockedVideoService: MockBlockedVideoService,
	}

	handler := &httpHandler.Handler{
		Server:  e,
		Service: services,
	}

	return handler
}

func TestIsVideoBlocked(t *testing.T) {
	handler := setupBlockedVideoHandler()
	e := handler.Server
	mockService := handler.Service.BlockedVideoService.(*MockBlockedVideoService)

	mockService.On("IsVideoBlocked", mock.Anything, mock.Anything).Return(true, nil)

	req := httptest.NewRequest(http.MethodGet, "/blocked-video/123", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, handler.IsVideoBlocked(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		mockService.AssertExpectations(t)
	}
}

func TestCreateBlockedVideo(t *testing.T) {
	handler := setupBlockedVideoHandler()
	e := handler.Server
	mockService := handler.Service.BlockedVideoService.(*MockBlockedVideoService)

	mockService.On("CreateBlockedVideo", mock.Anything, mock.Anything).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/blocked-video/123", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, handler.CreateBlockedVideo(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		mockService.AssertExpectations(t)
	}
}

func TestDeleteBlockedVideo(t *testing.T) {
	handler := setupBlockedVideoHandler()
	e := handler.Server
	mockService := handler.Service.BlockedVideoService.(*MockBlockedVideoService)

	mockService.On("DeleteBlockedVideo", mock.Anything, mock.Anything).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/blocked-video/123", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, handler.DeleteBlockedVideo(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		mockService.AssertExpectations(t)
	}
}

func TestGetBlockedVideos(t *testing.T) {
	handler := setupBlockedVideoHandler()
	e := handler.Server
	mockService := handler.Service.BlockedVideoService.(*MockBlockedVideoService)

	mockService.On("GetBlockedVideos", mock.Anything).Return([]*ent.BlockedVideos{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/blocked", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, handler.GetBlockedVideos(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		mockService.AssertExpectations(t)
	}
}
