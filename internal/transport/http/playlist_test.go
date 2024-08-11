package http_test

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"net/http/httptest"
// 	"strings"
// 	"testing"

// 	"github.com/go-playground/validator/v10"
// 	"github.com/labstack/echo/v4"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/zibbp/ganymede/ent"
// 	"github.com/zibbp/ganymede/ent/enttest"
// 	entPlaylist "github.com/zibbp/ganymede/ent/playlist"
// 	"github.com/zibbp/ganymede/internal/database"
// 	"github.com/zibbp/ganymede/internal/playlist"
// 	httpHandler "github.com/zibbp/ganymede/internal/transport/http"
// 	"github.com/zibbp/ganymede/internal/utils"
// )

// var (
// 	createPlaylistTestJson = `{
// 		"name": "test_playlist",
// 		"description": "test_description"
// 	}`
// )

// // * TestCreatePlaylist tests the CreatePlaylist function
// // Creates a new playlist
// func TestCreatePlaylist(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			PlaylistService: playlist.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

// 	req := httptest.NewRequest(http.MethodPost, "/api/v1/playlist", strings.NewReader(createPlaylistTestJson))
// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)

// 	if assert.NoError(t, h.CreatePlaylist(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check response body
// 		var response map[string]interface{}
// 		err := json.Unmarshal(rec.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, "test_playlist", response["name"])
// 	}
// }

// // * TestAddVodToPlaylist tests the AddVodToPlaylist function
// // Adds a vod to a playlist
// func TestAddVodToPlaylist(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	// Create a playlist
// 	dbPlaylist, err := client.Playlist.Create().SetName("test_playlist").SetDescription("test_description").Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Create a channel
// 	dbChannel, err := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Create a vod
// 	dbVod, err := client.Vod.Create().SetTitle("test vod").SetExtID("123").SetWebThumbnailPath("").SetVideoPath("").SetChannel(dbChannel).Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			PlaylistService: playlist.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

// 	addVodToPlaylistJson := `{
// 		"vod_id": "` + dbVod.ID.String() + `"
// 	}`

// 	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/playlist/%s", dbVod.ID.String()), strings.NewReader(addVodToPlaylistJson))
// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)
// 	c.SetParamNames("id")
// 	c.SetParamValues(dbPlaylist.ID.String())

// 	if assert.NoError(t, h.AddVodToPlaylist(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check response body
// 		// response will be a string
// 		var response string
// 		err := json.Unmarshal(rec.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, "ok", response)
// 	}
// }

// // * TestGetPlaylists tests the GetPlaylists function
// // Gets all playlists
// func TestGetPlaylists(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			PlaylistService: playlist.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

// 	// Create a playlist
// 	_, err := client.Playlist.Create().SetName("test_playlist").SetDescription("test_description").Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	req := httptest.NewRequest(http.MethodGet, "/api/v1/playlist", nil)
// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)

// 	if assert.NoError(t, h.GetPlaylists(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check response body
// 		var response []map[string]interface{}
// 		err := json.Unmarshal(rec.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, "test_playlist", response[0]["name"])
// 	}
// }

// // * TestGetPlaylist tests the GetPlaylist function
// // Gets a playlist
// func TestGetPlaylist(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			PlaylistService: playlist.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

// 	// Create a playlist
// 	dbPlaylist, err := client.Playlist.Create().SetName("test_playlist").SetDescription("test_description").Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/playlist/%s", dbPlaylist.ID.String()), nil)
// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)
// 	c.SetParamNames("id")
// 	c.SetParamValues(dbPlaylist.ID.String())

// 	if assert.NoError(t, h.GetPlaylist(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check response body
// 		var response map[string]interface{}
// 		err := json.Unmarshal(rec.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, "test_playlist", response["name"])
// 	}
// }

// // * TestUpdatePlaylist tests the UpdatePlaylist function
// // Update a playlist
// func TestUpdatePlaylist(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			PlaylistService: playlist.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

// 	// Create a playlist
// 	dbPlaylist, err := client.Playlist.Create().SetName("test_playlist").SetDescription("test_description").Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	updatePlaylistJson := `{
// 		"name": "test_playlist_updated",
// 		"description": "test_description_updated"
// 	}`

// 	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/playlist/%s", dbPlaylist.ID.String()), strings.NewReader(updatePlaylistJson))
// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)
// 	c.SetParamNames("id")
// 	c.SetParamValues(dbPlaylist.ID.String())

// 	if assert.NoError(t, h.UpdatePlaylist(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check response body
// 		var response map[string]interface{}
// 		err := json.Unmarshal(rec.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, "test_playlist_updated", response["name"])
// 	}
// }

// // * TestDeletePlaylist tests the DeletePlaylist function
// // Delete a playlist
// func TestDeletePlaylist(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			PlaylistService: playlist.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

// 	// Create a playlist
// 	dbPlaylist, err := client.Playlist.Create().SetName("test_playlist").SetDescription("test_description").Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/playlist/%s", dbPlaylist.ID.String()), nil)
// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)
// 	c.SetParamNames("id")
// 	c.SetParamValues(dbPlaylist.ID.String())

// 	if assert.NoError(t, h.DeletePlaylist(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check if playlist is deleted
// 		dbPlaylists, err := client.Playlist.Query().All(context.Background())
// 		assert.NoError(t, err)
// 		assert.Equal(t, 0, len(dbPlaylists))
// 	}
// }

// // * TestDeleteVodFromPlaylist tests the DeleteVodFromPlaylist function
// // Delete a vod from a playlist
// func TestDeleteVodFromPlaylist(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	// Create a playlist
// 	dbPlaylist, err := client.Playlist.Create().SetName("test_playlist").SetDescription("test_description").Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Create a channel
// 	dbChannel, err := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Create a vod
// 	dbVod, err := client.Vod.Create().SetTitle("test vod").SetExtID("123").SetWebThumbnailPath("").SetVideoPath("").SetChannel(dbChannel).Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	// Add vod to playlist
// 	_, err = client.Playlist.UpdateOne(dbPlaylist).AddVods(dbVod).Save(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			PlaylistService: playlist.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

// 	deletVodFromPlaylistJson := `{
// 		"vod_id": "` + dbVod.ID.String() + `"
// 	}`

// 	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/playlist/%s", dbPlaylist.ID.String()), strings.NewReader(deletVodFromPlaylistJson))
// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)
// 	c.SetParamNames("id")
// 	c.SetParamValues(dbPlaylist.ID.String())

// 	if assert.NoError(t, h.DeleteVodFromPlaylist(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check response body
// 		// response will be a string
// 		var response string
// 		err := json.Unmarshal(rec.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, "ok", response)

// 		// Check if vod is deleted from playlist
// 		dbPlaylist, err := client.Playlist.Query().Where(entPlaylist.ID(dbPlaylist.ID)).Only(context.Background())
// 		assert.NoError(t, err)
// 		assert.Equal(t, 0, len(dbPlaylist.Edges.Vods))
// 	}
// }
