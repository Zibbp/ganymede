package http_test

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"net/http/httptest"
// 	"strings"
// 	"testing"
// 	"time"

// 	"github.com/go-playground/validator/v10"
// 	"github.com/labstack/echo/v4"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/zibbp/ganymede/ent"
// 	"github.com/zibbp/ganymede/ent/enttest"
// 	"github.com/zibbp/ganymede/internal/database"
// 	httpHandler "github.com/zibbp/ganymede/internal/transport/http"
// 	"github.com/zibbp/ganymede/internal/utils"
// 	"github.com/zibbp/ganymede/internal/vod"
// )

// // * TestCreateVod tests the CreateVod function
// // Creates a vod
// func TestCreateVod(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			VodService: vod.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

// 	// Create a channel
// 	dbChannel, err := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	createVodJson := `{
// 		"channel_id": "` + dbChannel.ID.String() + `",
// 		"ext_id": "123456789",
// 		"platform": "twitch",
// 		"type": "archive",
// 		"title": "Test Vod",
// 		"duration": 6520,
// 		"views": 520,
// 		"resolution": "source",
// 		"thumbnail_path": "/vods/test/123456789/123456789-thumbnail.jpg",
// 		"web_thumbnail_path": "/vods/test/123456789/123456789-web_thumbnail.jpg",
// 		"video_path": "/vods/test/123456789/123456789-video.mp4",
// 		"chat_path": "/vods/test/123456789/123456789-chat.json",
// 		"chat_video_path": "/vods/test/123456789/123456789-chat.mp4",
// 		"info_path": "/vods/test/123456789/123456789-info.json",
// 		"streamed_at": "2023-02-02T20:07:51.594Z"
// 	}`

// 	req := httptest.NewRequest(http.MethodPost, "/api/v1/vod", strings.NewReader(createVodJson))
// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)

// 	if assert.NoError(t, h.CreateVod(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check response body
// 		var response map[string]interface{}
// 		err := json.Unmarshal(rec.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, "123456789", response["ext_id"])

// 	}
// }

// // * TestGetVods tests the GetVods function
// // Gets all vods
// func TestGetVods(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			VodService: vod.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

// 	// Create a channel
// 	dbChannel, err := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Create a vod
// 	dbVod, err := client.Vod.Create().SetChannel(dbChannel).SetExtID("123456789").SetPlatform("twitch").SetType("archive").SetTitle("Test Vod").SetDuration(6520).SetViews(520).SetResolution("source").SetThumbnailPath("/vods/test/123456789/123456789-thumbnail.jpg").SetWebThumbnailPath("/vods/test/123456789/123456789-web_thumbnail.jpg").SetVideoPath("/vods/test/123456789/123456789-video.mp4").SetChatPath("/vods/test/123456789/123456789-chat.json").SetChatVideoPath("/vods/test/123456789/123456789-chat.mp4").SetInfoPath("/vods/test/123456789/123456789-info.json").SetStreamedAt(time.Now()).Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	req := httptest.NewRequest(http.MethodGet, "/api/v1/vod", nil)
// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)

// 	if assert.NoError(t, h.GetVods(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check response body
// 		var response []map[string]interface{}
// 		err := json.Unmarshal(rec.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, dbVod.ID.String(), response[0]["id"])

// 	}
// }

// // * TestGetVod tests the GetVod function
// // Gets a vod
// func TestGetVod(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			VodService: vod.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

// 	// Create a channel
// 	dbChannel, err := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Create a vod
// 	dbVod, err := client.Vod.Create().SetChannel(dbChannel).SetExtID("123456789").SetPlatform("twitch").SetType("archive").SetTitle("Test Vod").SetDuration(6520).SetViews(520).SetResolution("source").SetThumbnailPath("/vods/test/123456789/123456789-thumbnail.jpg").SetWebThumbnailPath("/vods/test/123456789/123456789-web_thumbnail.jpg").SetVideoPath("/vods/test/123456789/123456789-video.mp4").SetChatPath("/vods/test/123456789/123456789-chat.json").SetChatVideoPath("/vods/test/123456789/123456789-chat.mp4").SetInfoPath("/vods/test/123456789/123456789-info.json").SetStreamedAt(time.Now()).Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vod/%s", dbVod.ID.String()), nil)
// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)
// 	c.SetParamNames("id")
// 	c.SetParamValues(dbVod.ID.String())

// 	if assert.NoError(t, h.GetVod(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check response body
// 		var response map[string]interface{}
// 		err := json.Unmarshal(rec.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, dbVod.ID.String(), response["id"])

// 	}
// }

// // * TestDeleteVod tests the DeleteVod function
// // Deletes a vod
// func TestDeleteVod(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			VodService: vod.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

// 	// Create a channel
// 	dbChannel, err := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Create a vod
// 	dbVod, err := client.Vod.Create().SetChannel(dbChannel).SetExtID("123456789").SetPlatform("twitch").SetType("archive").SetTitle("Test Vod").SetDuration(6520).SetViews(520).SetResolution("source").SetThumbnailPath("/vods/test/123456789/123456789-thumbnail.jpg").SetWebThumbnailPath("/vods/test/123456789/123456789-web_thumbnail.jpg").SetVideoPath("/vods/test/123456789/123456789-video.mp4").SetChatPath("/vods/test/123456789/123456789-chat.json").SetChatVideoPath("/vods/test/123456789/123456789-chat.mp4").SetInfoPath("/vods/test/123456789/123456789-info.json").SetStreamedAt(time.Now()).Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/vod/%s", dbVod.ID.String()), nil)
// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)
// 	c.SetParamNames("id")
// 	c.SetParamValues(dbVod.ID.String())

// 	if assert.NoError(t, h.DeleteVod(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check if vod is deleted
// 		vods, err := client.Vod.Query().All(context.Background())
// 		assert.NoError(t, err)
// 		assert.Equal(t, 0, len(vods))
// 	}
// }

// // * TestUpdateVod tests the UpdateVod function
// // Updates a vod
// func TestUpdateVod(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			VodService: vod.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

// 	// Create a channel
// 	dbChannel, err := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Create a vod
// 	dbVod, err := client.Vod.Create().SetChannel(dbChannel).SetExtID("123456789").SetPlatform("twitch").SetType("archive").SetTitle("Test Vod").SetDuration(6520).SetViews(520).SetResolution("source").SetThumbnailPath("/vods/test/123456789/123456789-thumbnail.jpg").SetWebThumbnailPath("/vods/test/123456789/123456789-web_thumbnail.jpg").SetVideoPath("/vods/test/123456789/123456789-video.mp4").SetChatPath("/vods/test/123456789/123456789-chat.json").SetChatVideoPath("/vods/test/123456789/123456789-chat.mp4").SetInfoPath("/vods/test/123456789/123456789-info.json").SetStreamedAt(time.Now()).Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	updateVodJson := `{
// 		"channel_id": "` + dbChannel.ID.String() + `",
// 		"ext_id": "123456789",
// 		"platform": "twitch",
// 		"type": "archive",
// 		"title": "Updated Test Vod",
// 		"duration": 6520,
// 		"views": 520,
// 		"resolution": "source",
// 		"thumbnail_path": "/vods/test/123456789/123456789-thumbnail.jpg",
// 		"web_thumbnail_path": "/vods/test/123456789/123456789-web_thumbnail.jpg",
// 		"video_path": "/vods/test/123456789/123456789-video.mp4",
// 		"chat_path": "/vods/test/123456789/123456789-chat.json",
// 		"chat_video_path": "/vods/test/123456789/123456789-chat.mp4",
// 		"info_path": "/vods/test/123456789/123456789-info.json",
// 		"streamed_at": "2023-02-02T20:07:51.594Z"
// 	}`

// 	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/vod/%s", dbVod.ID.String()), strings.NewReader(updateVodJson))
// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)
// 	c.SetParamNames("id")
// 	c.SetParamValues(dbVod.ID.String())

// 	if assert.NoError(t, h.UpdateVod(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check response body
// 		var response map[string]interface{}
// 		err := json.Unmarshal(rec.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, "Updated Test Vod", response["title"])

// 	}
// }

// // * TestSearchVods tests the SearchVods function
// // Searches for vods
// func TestSearchVods(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			VodService: vod.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

// 	// Create a channel
// 	dbChannel, err := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Create a vod
// 	_, err = client.Vod.Create().SetChannel(dbChannel).SetExtID("123456789").SetPlatform("twitch").SetType("archive").SetTitle("Test Vod").SetDuration(6520).SetViews(520).SetResolution("source").SetThumbnailPath("/vods/test/123456789/123456789-thumbnail.jpg").SetWebThumbnailPath("/vods/test/123456789/123456789-web_thumbnail.jpg").SetVideoPath("/vods/test/123456789/123456789-video.mp4").SetChatPath("/vods/test/123456789/123456789-chat.json").SetChatVideoPath("/vods/test/123456789/123456789-chat.mp4").SetInfoPath("/vods/test/123456789/123456789-info.json").SetStreamedAt(time.Now()).Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vod/search/?q=%s&limit=%s&offset=%s", "test", "20", "1"), nil)
// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)
// 	c.SetParamNames("q", "limit", "offset")
// 	c.SetParamValues("test", "20", "1")

// 	if assert.NoError(t, h.SearchVods(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check response body
// 		var response map[string]interface{}
// 		err := json.Unmarshal(rec.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, float64(1), response["total_count"])
// 	}
// }

// // * TestGetVodPlaylists tests the GetVodPlaylists function
// // Gets a vod's playlists
// func TestGetVodPlaylists(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			VodService: vod.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

// 	// Create a channel
// 	dbChannel, err := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Create a vod
// 	dbVod, err := client.Vod.Create().SetChannel(dbChannel).SetExtID("123456789").SetPlatform("twitch").SetType("archive").SetTitle("Test Vod").SetDuration(6520).SetViews(520).SetResolution("source").SetThumbnailPath("/vods/test/123456789/123456789-thumbnail.jpg").SetWebThumbnailPath("/vods/test/123456789/123456789-web_thumbnail.jpg").SetVideoPath("/vods/test/123456789/123456789-video.mp4").SetChatPath("/vods/test/123456789/123456789-chat.json").SetChatVideoPath("/vods/test/123456789/123456789-chat.mp4").SetInfoPath("/vods/test/123456789/123456789-info.json").SetStreamedAt(time.Now()).Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Create a playlist
// 	dbPlaylist, err := client.Playlist.Create().SetName("test_playlist").SetDescription("Test Playlist").Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Add vod to playlist
// 	_, err = client.Playlist.UpdateOne(dbPlaylist).AddVods(dbVod).Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vod/%s/playlist", dbVod.ID.String()), nil)
// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)
// 	c.SetParamNames("id")
// 	c.SetParamValues(dbVod.ID.String())

// 	if assert.NoError(t, h.GetVodPlaylists(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check response body
// 		var response []map[string]interface{}
// 		err := json.Unmarshal(rec.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, dbPlaylist.ID.String(), response[0]["id"])
// 	}
// }

// // * TestGetVodsPagination tests the GetVodsPagination function
// // Gets a paginated list of vods
// func TestGetVodsPagination(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			VodService: vod.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

// 	// Create a channel
// 	dbChannel, err := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Create a vod
// 	_, err = client.Vod.Create().SetChannel(dbChannel).SetExtID("123456789").SetPlatform("twitch").SetType("archive").SetTitle("Test Vod").SetDuration(6520).SetViews(520).SetResolution("source").SetThumbnailPath("/vods/test/123456789/123456789-thumbnail.jpg").SetWebThumbnailPath("/vods/test/123456789/123456789-web_thumbnail.jpg").SetVideoPath("/vods/test/123456789/123456789-video.mp4").SetChatPath("/vods/test/123456789/123456789-chat.json").SetChatVideoPath("/vods/test/123456789/123456789-chat.mp4").SetInfoPath("/vods/test/123456789/123456789-info.json").SetStreamedAt(time.Now()).Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	dbVod, err := client.Vod.Create().SetChannel(dbChannel).SetExtID("987654321").SetPlatform("twitch").SetType("highlight").SetTitle("Test Vod 2").SetDuration(6520).SetViews(520).SetResolution("source").SetThumbnailPath("/vods/test/123456789/123456789-thumbnail.jpg").SetWebThumbnailPath("/vods/test/123456789/123456789-web_thumbnail.jpg").SetVideoPath("/vods/test/123456789/123456789-video.mp4").SetChatPath("/vods/test/123456789/123456789-chat.json").SetChatVideoPath("/vods/test/123456789/123456789-chat.mp4").SetInfoPath("/vods/test/123456789/123456789-info.json").SetStreamedAt(time.Now()).Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vod/paginate?limit=%s&offset=%s&channel_id=%s", "20", "0", dbChannel.ID.String()), nil)
// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)
// 	c.SetParamNames("limit", "offset", "channel_id")
// 	c.SetParamValues("20", "0", dbChannel.ID.String())

// 	if assert.NoError(t, h.GetVodsPagination(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check response body
// 		var response map[string]interface{}
// 		err := json.Unmarshal(rec.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, float64(0), response["offset"])
// 		assert.Equal(t, float64(20), response["limit"])
// 		assert.Equal(t, float64(2), response["total_count"])
// 		assert.Equal(t, float64(1), response["pages"])
// 		assert.Equal(t, dbVod.ID.String(), response["data"].([]interface{})[0].(map[string]interface{})["id"])

// 	}
// }
