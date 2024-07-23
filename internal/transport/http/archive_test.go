package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/archive"
	httpHandler "github.com/zibbp/ganymede/internal/transport/http"

	"github.com/zibbp/ganymede/internal/utils"
)

type MockArchiveService struct {
	mock.Mock
}

func (m *MockArchiveService) ArchiveChannel(ctx context.Context, channelName string) (*ent.Channel, error) {
	args := m.Called(ctx, channelName)
	return args.Get(0).(*ent.Channel), args.Error(1)
}

func (m *MockArchiveService) ArchiveVideo(ctx context.Context, input archive.ArchiveVideoInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockArchiveService) ArchiveLivestream(ctx context.Context, input archive.ArchiveVideoInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func setupEcho() *echo.Echo {
	e := echo.New()
	e.Validator = &utils.CustomValidator{Validator: validator.New()}
	return e
}

func setupArchiveHandler() *httpHandler.Handler {
	e := setupEcho()
	mockArchiveService := new(MockArchiveService)

	services := httpHandler.Services{
		ArchiveService: mockArchiveService,
	}

	handler := &httpHandler.Handler{
		Server:  e,
		Service: services,
	}

	return handler
}

// TestArchiveChannel is a test function for archiving a channel.
//
// It tests the functionality of archiving a channel by sending a POST request with the channel name and verifying the response.
func TestArchiveChannel(t *testing.T) {
	handler := setupArchiveHandler()
	e := handler.Server
	mockService := handler.Service.ArchiveService.(*MockArchiveService)

	channelName := "test_channel"
	mockChannel := &ent.Channel{Name: channelName}

	mockService.On("ArchiveChannel", mock.Anything, channelName).Return(mockChannel, nil)

	reqBody, _ := json.Marshal(httpHandler.ArchiveChannelRequest{ChannelName: channelName})
	req := httptest.NewRequest(http.MethodPost, "/archive/channel", bytes.NewBuffer(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, handler.ArchiveChannel(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		var responseChannel ent.Channel
		json.Unmarshal(rec.Body.Bytes(), &responseChannel)
		assert.Equal(t, mockChannel.Name, responseChannel.Name)
	}

	mockService.AssertExpectations(t)
}

func TestArchiveVideo(t *testing.T) {
	handler := setupArchiveHandler()
	e := handler.Server
	mockService := handler.Service.ArchiveService.(*MockArchiveService)

	// test archive video
	archiveVideoBody := httpHandler.ArchiveVideoRequest{
		VideoId:     "123456789",
		Quality:     "best",
		ArchiveChat: true,
		RenderChat:  false,
	}

	expectedInput := archive.ArchiveVideoInput{
		VideoId:     "123456789",
		Quality:     "best",
		ArchiveChat: true,
		RenderChat:  false,
	}

	mockService.On("ArchiveVideo", mock.Anything, expectedInput).Return(nil)

	reqBody, _ := json.Marshal(archiveVideoBody)
	req := httptest.NewRequest(http.MethodPost, "/archive/video", bytes.NewBuffer(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, handler.ArchiveVideo(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// test invalid archive video
	invalidArchiveVideoBody := httpHandler.ArchiveVideoRequest{
		VideoId:     "123456789",
		ChannelId:   "123456789",
		Quality:     "best",
		ArchiveChat: true,
		RenderChat:  false,
	}

	reqBody, _ = json.Marshal(invalidArchiveVideoBody)
	req = httptest.NewRequest(http.MethodPost, "/archive/video", bytes.NewBuffer(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if assert.Error(t, handler.ArchiveVideo(c)) {
	}

	mockService.AssertExpectations(t)
}

func TestArchiveLivestream(t *testing.T) {
	handler := setupArchiveHandler()
	e := handler.Server
	mockService := handler.Service.ArchiveService.(*MockArchiveService)

	channelId := uuid.New()

	// test archive livestream
	archiveLivestreamBody := httpHandler.ArchiveVideoRequest{
		ChannelId:   channelId.String(),
		Quality:     "best",
		ArchiveChat: true,
		RenderChat:  false,
	}

	expectedInput := archive.ArchiveVideoInput{
		ChannelId:   channelId,
		Quality:     "best",
		ArchiveChat: true,
		RenderChat:  false,
	}

	mockService.On("ArchiveLivestream", mock.Anything, expectedInput).Return(nil)

	reqBody, _ := json.Marshal(archiveLivestreamBody)
	req := httptest.NewRequest(http.MethodPost, "/archive/livestream", bytes.NewBuffer(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, handler.ArchiveVideo(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	mockService.AssertExpectations(t)
}
