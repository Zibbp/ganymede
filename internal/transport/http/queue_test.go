package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/enttest"
	entQueue "github.com/zibbp/ganymede/ent/queue"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/queue"
	httpHandler "github.com/zibbp/ganymede/internal/transport/http"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
)

// * TestCreateQueueItem tests the CreateQueueItem function
// Creates a new queue item
func TestCreateQueueItem(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	vodService := vod.NewService(&database.Database{Client: client})
	channelService := channel.NewService(&database.Database{Client: client})

	h := &httpHandler.Handler{
		Server: echo.New(),
		Service: httpHandler.Services{
			QueueService: queue.NewService(&database.Database{Client: client}, vodService, channelService),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	// Create a channel
	dbChannel, err := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Create a vod
	dbVod, err := client.Vod.Create().SetTitle("test vod").SetExtID("123").SetWebThumbnailPath("").SetVideoPath("").SetChannel(dbChannel).Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	createQueueItemJson := `{
		"vod_id": "` + dbVod.ID.String() + `"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/queue", strings.NewReader(createQueueItemJson))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)

	if assert.NoError(t, h.CreateQueueItem(c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)

		// Check response body
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)

		/// Check if the queue item was created
		queueItem, err := client.Queue.Query().Where(entQueue.HasVodWith(entVod.ID(dbVod.ID))).WithVod().Only(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, dbVod.ID, queueItem.Edges.Vod.ID)

	}
}

// * TestGetQueueItems tests the GetQueueItems function
// Gets all queue items
func TestGetQueueItems(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	vodService := vod.NewService(&database.Database{Client: client})
	channelService := channel.NewService(&database.Database{Client: client})

	h := &httpHandler.Handler{
		Server: echo.New(),
		Service: httpHandler.Services{
			QueueService: queue.NewService(&database.Database{Client: client}, vodService, channelService),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	// Create a channel
	dbChannel, err := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Create a vod
	dbVod, err := client.Vod.Create().SetTitle("test vod").SetExtID("123").SetWebThumbnailPath("").SetVideoPath("").SetChannel(dbChannel).Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Create a queue item
	dbQueue, err := client.Queue.Create().SetVod(dbVod).Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/queue", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)

	if assert.NoError(t, h.GetQueueItems(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)

		// Check response body
		var response []map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(response))
		assert.Equal(t, dbQueue.ID.String(), response[0]["id"])

	}
}

// * TestGetQueueItem tests the GetQueueItem function
// Gets all queue items
func TestGetQueueItem(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	vodService := vod.NewService(&database.Database{Client: client})
	channelService := channel.NewService(&database.Database{Client: client})

	h := &httpHandler.Handler{
		Server: echo.New(),
		Service: httpHandler.Services{
			QueueService: queue.NewService(&database.Database{Client: client}, vodService, channelService),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	// Create a channel
	dbChannel, err := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Create a vod
	dbVod, err := client.Vod.Create().SetTitle("test vod").SetExtID("123").SetWebThumbnailPath("").SetVideoPath("").SetChannel(dbChannel).Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Create a queue item
	dbQueue, err := client.Queue.Create().SetVod(dbVod).Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/queue/%s", dbQueue.ID.String()), nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(dbQueue.ID.String())

	if assert.NoError(t, h.GetQueueItem(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)

		// Check response body
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, dbQueue.ID.String(), response["id"])

	}
}

// * TestUpdateQueueItem tests the UpdateQueueItem function
// Updates a queue item
func TestUpdateQueueItem(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	vodService := vod.NewService(&database.Database{Client: client})
	channelService := channel.NewService(&database.Database{Client: client})

	h := &httpHandler.Handler{
		Server: echo.New(),
		Service: httpHandler.Services{
			QueueService: queue.NewService(&database.Database{Client: client}, vodService, channelService),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	// Create a channel
	dbChannel, err := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Create a vod
	dbVod, err := client.Vod.Create().SetTitle("test vod").SetExtID("123").SetWebThumbnailPath("").SetVideoPath("").SetChannel(dbChannel).Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Create a queue item
	dbQueue, err := client.Queue.Create().SetVod(dbVod).Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	updateQueueItemJson := `{
		"processing": false,
		"task_vod_create_folder": "success",
		"task_vod_download_thumbnail": "success",
		"task_vod_save_info": "success",
		"task_video_download": "success",
		"task_video_move": "success",
		"task_chat_download": "success",
		"task_chat_render": "success",
		"task_chat_move": "success",
		"task_video_convert": "success",
		"task_chat_convert": "success"
	}`

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/queue/%s", dbQueue.ID.String()), strings.NewReader(updateQueueItemJson))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(dbQueue.ID.String())

	if assert.NoError(t, h.UpdateQueueItem(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)

		// Check response body
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["task_vod_create_folder"])

	}
}

// * TestDeleteQueueItem tests the DeleteQueueItem function
// Deletes a queue item
func TestDeleteQueueItem(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	vodService := vod.NewService(&database.Database{Client: client})
	channelService := channel.NewService(&database.Database{Client: client})

	h := &httpHandler.Handler{
		Server: echo.New(),
		Service: httpHandler.Services{
			QueueService: queue.NewService(&database.Database{Client: client}, vodService, channelService),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	// Create a channel
	dbChannel, err := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Create a vod
	dbVod, err := client.Vod.Create().SetTitle("test vod").SetExtID("123").SetWebThumbnailPath("").SetVideoPath("").SetChannel(dbChannel).Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Create a queue item
	dbQueue, err := client.Queue.Create().SetVod(dbVod).Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/queue/%s", dbQueue.ID.String()), nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(dbQueue.ID.String())

	if assert.NoError(t, h.DeleteQueueItem(c)) {
		assert.Equal(t, http.StatusNoContent, rec.Code)

		// Check if queue item was deleted
		queueItem, err := client.Queue.Get(context.Background(), dbQueue.ID)
		assert.Error(t, err)
		assert.Nil(t, queueItem)

	}
}

// * TestReadQueueLogFile tests the ReadQueueLogFile function
// Deletes a queue item
func TestReadQueueLogFile(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	vodService := vod.NewService(&database.Database{Client: client})
	channelService := channel.NewService(&database.Database{Client: client})

	h := &httpHandler.Handler{
		Server: echo.New(),
		Service: httpHandler.Services{
			QueueService: queue.NewService(&database.Database{Client: client}, vodService, channelService),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	// Create a channel
	dbChannel, err := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Create a vod
	dbVod, err := client.Vod.Create().SetTitle("test vod").SetExtID("123").SetWebThumbnailPath("").SetVideoPath("").SetChannel(dbChannel).Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Create a queue item
	dbQueue, err := client.Queue.Create().SetVod(dbVod).Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Create log folder
	err = os.MkdirAll("/logs", 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create log file
	logFile, err := os.Create(fmt.Sprintf("/logs/%s-%s.log", dbVod.ID.String(), "video"))
	if err != nil {
		t.Fatal(err)
	}

	// Write to log file
	_, err = logFile.WriteString("test log")
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/queue/%s/tail?type=%s", dbQueue.ID.String(), "video"), nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)
	c.SetParamNames("id", "type")
	c.SetParamValues(dbQueue.ID.String(), "video")

	if assert.NoError(t, h.ReadQueueLogFile(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)

		// Check response body
		var response string
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "test log", response)

	}
}
