package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	httpHandler "github.com/zibbp/ganymede/internal/transport/http"
)

type MockBlockedVodService struct {
	mock.Mock
}

func (m *MockBlockedVodService) IsVodBlocked(ctx context.Context, id string) (bool, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockBlockedVodService) CreateBlockedVod(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockBlockedVodService) DeleteBlockedVod(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockBlockedVodService) GetBlockedVods(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func setupBlockedVodHandler() *httpHandler.Handler {
	e := setupEcho()

	MockBlockedVodService := new(MockBlockedVodService)

	services := httpHandler.Services{
		BlockedVodService: MockBlockedVodService,
	}

	handler := &httpHandler.Handler{
		Server:  e,
		Service: services,
	}

	return handler
}

func TestIsVodBlocked(t *testing.T) {
	handler := setupBlockedVodHandler()
	e := handler.Server
	mockService := handler.Service.BlockedVodService.(*MockBlockedVodService)

	mockService.On("IsVodBlocked", mock.Anything, mock.Anything).Return(true, nil)

	req := httptest.NewRequest(http.MethodGet, "/blocked/123", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, handler.IsVodBlocked(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		mockService.AssertExpectations(t)
	}
}

func TestCreateBlockedVod(t *testing.T) {
	handler := setupBlockedVodHandler()
	e := handler.Server
	mockService := handler.Service.BlockedVodService.(*MockBlockedVodService)

	mockService.On("CreateBlockedVod", mock.Anything, mock.Anything).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/blocked/123", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, handler.CreateBlockedVod(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		mockService.AssertExpectations(t)
	}
}

func TestDeleteBlockedVod(t *testing.T) {
	handler := setupBlockedVodHandler()
	e := handler.Server
	mockService := handler.Service.BlockedVodService.(*MockBlockedVodService)

	mockService.On("DeleteBlockedVod", mock.Anything, mock.Anything).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/blocked/123", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, handler.DeleteBlockedVod(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		mockService.AssertExpectations(t)
	}
}

func TestGetBlockedVods(t *testing.T) {
	handler := setupBlockedVodHandler()
	e := handler.Server
	mockService := handler.Service.BlockedVodService.(*MockBlockedVodService)

	mockService.On("GetBlockedVods", mock.Anything).Return([]string{"123"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/blocked", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, handler.GetBlockedVods(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		mockService.AssertExpectations(t)
	}
}
