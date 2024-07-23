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
// 	_ "github.com/mattn/go-sqlite3"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/zibbp/ganymede/ent"
// 	entChannel "github.com/zibbp/ganymede/ent/channel"
// 	"github.com/zibbp/ganymede/ent/enttest"
// 	"github.com/zibbp/ganymede/internal/channel"
// 	"github.com/zibbp/ganymede/internal/database"
// 	httpHandler "github.com/zibbp/ganymede/internal/transport/http"
// 	"github.com/zibbp/ganymede/internal/utils"
// )

// var (
// 	channelJSON = `{
// 		"name": "test_channel",
// 		"display_name": "Test Channel",
// 		"image_path": "/vods/test_channel/test_channel.jpg"
// 		}`
// 	invalidChannelJSON = `{
// 		"name": "t",
// 		"display_name": "t",
// 		"image_path": "t"
// 		}`
// )

// // * TestCreateChannel tests the CreateChannel function
// // Test creates a new channel and checks if the response is correct
// func TestCreateChannel(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			ChannelService: channel.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

// 	req := httptest.NewRequest(http.MethodPost, "/api/v1/channels", strings.NewReader(channelJSON))
// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)

// 	if assert.NoError(t, h.CreateChannel(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check response body
// 		var response map[string]interface{}
// 		err := json.Unmarshal(rec.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, "test_channel", response["name"])
// 	}
// }

// // * TestCreateChannelInvalid tests the CreateChannel function
// // Test creates a new channel with invalid data and checks if the response is correct
// func TestCreateInvalidChannel(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			ChannelService: channel.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

// 	req := httptest.NewRequest(http.MethodPost, "/api/v1/channels", strings.NewReader(invalidChannelJSON))
// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)

// 	// Response should be 400, pass the test if it is
// 	if assert.Error(t, h.CreateChannel(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)
// 	}
// }

// // * TestGetChannels tests the GetChannel function
// // Test creates a new channel and checks if the response contains 1 channel
// func TestGetChannels(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			ChannelService: channel.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	// Create a channel
// 	client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").SaveX(context.Background())

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

// 	req := httptest.NewRequest(http.MethodGet, "/api/v1/channel", nil)
// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)

// 	if assert.NoError(t, h.GetChannels(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check response body
// 		var response []map[string]interface{}
// 		err := json.Unmarshal(rec.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, 1, len(response))
// 	}
// }

// // * TestGetChannel tests the GetChannel function
// // Test creates a new channel and checks if the response contains the correct channel
// func TestGetChannel(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			ChannelService: channel.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	// Create a channel
// 	testChannel := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").SaveX(context.Background())

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}
// 	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/channel/%s", testChannel.ID.String()), nil)

// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)

// 	// Set path parameters
// 	c.SetParamNames("id")
// 	c.SetParamValues(testChannel.ID.String())

// 	if assert.NoError(t, h.GetChannel(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check response body
// 		var response map[string]interface{}
// 		err := json.Unmarshal(rec.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, "test_channel", response["name"])
// 	}
// }

// // * TestDeleteChannel tests the DeleteChannel function
// // Test creates a new channel and deletes it and checks if the response is correct
// func TestDeleteChannel(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			ChannelService: channel.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	// Create a channel
// 	testChannel := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").SaveX(context.Background())

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}
// 	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/channel/%s", testChannel.ID.String()), nil)

// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)

// 	// Set path parameters
// 	c.SetParamNames("id")
// 	c.SetParamValues(testChannel.ID.String())

// 	if assert.NoError(t, h.DeleteChannel(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)
// 	}

// 	// Check if channel is deleted
// 	channel, err := client.Channel.Query().Where(entChannel.ID(testChannel.ID)).Only(context.Background())
// 	assert.Error(t, err)
// 	assert.Nil(t, channel)
// }

// // * TestUpdateChannel tests the UpdateChannel function
// // Test creates a new channel and updates it and checks if the response is correct
// func TestUpdateChannel(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			ChannelService: channel.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	// Create a channel
// 	testChannel := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").SaveX(context.Background())

// 	// Updated channel
// 	updatedJson := `{
// 		"name": "updated",
// 		"display_name": "updated",
// 		"image_path": "/vods/updated/updated.jpg"
// 		}`

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}
// 	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/channel/%s", testChannel.ID.String()), strings.NewReader(updatedJson))

// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)

// 	// Set path parameters
// 	c.SetParamNames("id")
// 	c.SetParamValues(testChannel.ID.String())

// 	if assert.NoError(t, h.UpdateChannel(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check response body
// 		var response map[string]interface{}
// 		err := json.Unmarshal(rec.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, "updated", response["name"])
// 		assert.Equal(t, "updated", response["display_name"])
// 		assert.Equal(t, "/vods/updated/updated.jpg", response["image_path"])
// 	}
// }

// // * TestGetChannelByName tests the GetChannelByName function
// // Test creates a new channel and checks if the response contains the correct channel
// func TestGetChannelByName(t *testing.T) {
// 	opts := []enttest.Option{
// 		enttest.WithOptions(ent.Log(t.Log)),
// 	}

// 	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
// 	defer client.Close()

// 	h := &httpHandler.Handler{
// 		Server: echo.New(),
// 		Service: httpHandler.Services{
// 			ChannelService: channel.NewService(&database.Database{Client: client}),
// 		},
// 	}

// 	// Create a channel
// 	testChannel := client.Channel.Create().SetName("test_channel").SetDisplayName("Test Channel").SetImagePath("/vods/test_channel/test_channel.jpg").SaveX(context.Background())

// 	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}
// 	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/channel/name/%s", testChannel.Name), nil)

// 	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
// 	rec := httptest.NewRecorder()
// 	c := h.Server.NewContext(req, rec)

// 	// Set path parameters
// 	c.SetParamNames("name")
// 	c.SetParamValues(testChannel.Name)

// 	if assert.NoError(t, h.GetChannelByName(c)) {
// 		assert.Equal(t, http.StatusOK, rec.Code)

// 		// Check response body
// 		var response map[string]interface{}
// 		err := json.Unmarshal(rec.Body.Bytes(), &response)
// 		assert.NoError(t, err)
// 		assert.Equal(t, "test_channel", response["name"])
// 	}
// }
