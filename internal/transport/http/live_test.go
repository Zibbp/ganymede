package http_test

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
	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/ent"
	entChannel "github.com/zibbp/ganymede/ent/channel"
	"github.com/zibbp/ganymede/ent/enttest"
	entLive "github.com/zibbp/ganymede/ent/live"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/queue"
	httpHandler "github.com/zibbp/ganymede/internal/transport/http"
	"github.com/zibbp/ganymede/internal/twitch"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
)

var ()

// * TestAddLiveWatchedChannel tests the create watched channel
// Test creates a live watched channel
func TestAddLiveWatchedChannel(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	twitchService := twitch.NewService()
	vodService := vod.NewService(&database.Database{Client: client})
	channelService := channel.NewService(&database.Database{Client: client})
	queueService := queue.NewService(&database.Database{Client: client}, vodService, channelService)
	archiveService := archive.NewService(&database.Database{Client: client}, twitchService, channelService, vodService, queueService)

	h := &httpHandler.Handler{
		Server: echo.New(),
		Service: httpHandler.Services{
			LiveService: live.NewService(&database.Database{Client: client}, twitchService, archiveService),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	// Create a test channel
	testChannel := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").SaveX(context.Background())

	// Watched channel json
	liveWatchedChannelJson := `{"channel_id": "` + testChannel.ID.String() + `", "watch_live": true, "watch_vod": true, "download_archives": true, "download_highlights": true, "download_uploads": true, "resolution": "best", "archive_chat": true}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/live", strings.NewReader(liveWatchedChannelJson))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)

	if assert.NoError(t, h.AddLiveWatchedChannel(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)

		// Check response body
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Check database to ensure the live watched channel was created
		liveWatchedChannels := client.Live.Query().Where(entLive.HasChannelWith(entChannel.IDEQ(testChannel.ID))).AllX(context.Background())
		assert.Equal(t, 1, len(liveWatchedChannels))
	}
}

// * TestGetLiveWatchedChannels tests the get watched channels
// Test gets watched channels
func TestGetLiveWatchedChannels(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	twitchService := twitch.NewService()
	vodService := vod.NewService(&database.Database{Client: client})
	channelService := channel.NewService(&database.Database{Client: client})
	queueService := queue.NewService(&database.Database{Client: client}, vodService, channelService)
	archiveService := archive.NewService(&database.Database{Client: client}, twitchService, channelService, vodService, queueService)

	h := &httpHandler.Handler{
		Server: echo.New(),
		Service: httpHandler.Services{
			LiveService: live.NewService(&database.Database{Client: client}, twitchService, archiveService),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	// Create a test channel
	testChannel := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").SaveX(context.Background())

	// Create a live watched channel
	client.Live.Create().SetChannel(testChannel).SetWatchLive(true).SetWatchVod(true).SetDownloadArchives(true).SetDownloadHighlights(true).SetDownloadUploads(true).SetResolution("best").SetArchiveChat(true).SaveX(context.Background())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/live", nil)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)

	if assert.NoError(t, h.GetLiveWatchedChannels(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)

		// Check response body
		var response []map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Check database to ensure the live watched channel was created
		liveWatchedChannels := client.Live.Query().Where(entLive.HasChannelWith(entChannel.IDEQ(testChannel.ID))).AllX(context.Background())
		assert.Equal(t, 1, len(liveWatchedChannels))
	}
}

// * TestUpdateLiveWatchedChannel tests the update live watched channel
// Test updating a live watched channel
func TestUpdateLiveWatchedChannel(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	twitchService := twitch.NewService()
	vodService := vod.NewService(&database.Database{Client: client})
	channelService := channel.NewService(&database.Database{Client: client})
	queueService := queue.NewService(&database.Database{Client: client}, vodService, channelService)
	archiveService := archive.NewService(&database.Database{Client: client}, twitchService, channelService, vodService, queueService)

	h := &httpHandler.Handler{
		Server: echo.New(),
		Service: httpHandler.Services{
			LiveService: live.NewService(&database.Database{Client: client}, twitchService, archiveService),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	// Create a test channel
	testChannel := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").SaveX(context.Background())

	// Create a live watched channel
	liveWatchedChannel := client.Live.Create().SetChannel(testChannel).SetWatchLive(true).SetWatchVod(true).SetDownloadArchives(true).SetDownloadHighlights(true).SetDownloadUploads(true).SetResolution("best").SetArchiveChat(true).SaveX(context.Background())

	// Live watched channel json
	liveWatchedChannelJson := `{"channel_id": "` + testChannel.ID.String() + `", "watch_live": false, "watch_vod": false, "download_archives": false, "download_highlights": false, "download_uploads": false, "resolution": "720p60", "archive_chat": false}`

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/live/%s", liveWatchedChannel.ID.String()), strings.NewReader(liveWatchedChannelJson))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)

	// Set params
	c.SetParamNames("id")
	c.SetParamValues(liveWatchedChannel.ID.String())

	if assert.NoError(t, h.UpdateLiveWatchedChannel(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)

		// Check response body
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		// Check if the live watched channel was updated, the fields set to false will not be returned
		assert.Equal(t, "720p60", response["resolution"])
	}
}

// * TestDeleteLiveWatchedChannel tests the delete watched channel
// Test deletes a live watched channel
func TestDeleteLiveWatchedChannel(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	twitchService := twitch.NewService()
	vodService := vod.NewService(&database.Database{Client: client})
	channelService := channel.NewService(&database.Database{Client: client})
	queueService := queue.NewService(&database.Database{Client: client}, vodService, channelService)
	archiveService := archive.NewService(&database.Database{Client: client}, twitchService, channelService, vodService, queueService)

	h := &httpHandler.Handler{
		Server: echo.New(),
		Service: httpHandler.Services{
			LiveService: live.NewService(&database.Database{Client: client}, twitchService, archiveService),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	// Create a test channel
	testChannel := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").SaveX(context.Background())

	// Create a live watched channel
	liveWatchedChannel := client.Live.Create().SetChannel(testChannel).SetWatchLive(true).SetWatchVod(true).SetDownloadArchives(true).SetDownloadHighlights(true).SetDownloadUploads(true).SetResolution("best").SetArchiveChat(true).SaveX(context.Background())

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/live/"+liveWatchedChannel.ID.String(), nil)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)

	// Set params
	c.SetParamNames("id")
	c.SetParamValues(liveWatchedChannel.ID.String())

	if assert.NoError(t, h.DeleteLiveWatchedChannel(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)

		// Check if watched channel is deleted from database
		liveWatchedChannels := client.Live.Query().Where(entLive.HasChannelWith(entChannel.IDEQ(testChannel.ID))).AllX(context.Background())
		assert.Equal(t, 0, len(liveWatchedChannels))

	}
}
