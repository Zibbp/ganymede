package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/enttest"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/utils"
)

var (
	channelJSON = `{
		"name": "test_channel",
		"display_name": "Test Channel",
		"image_path": "/vods/test_channel/test_channel.jpg"
		}`
)

func TestCreateChannel(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	h := &Handler{
		Server: echo.New(),
		Service: Services{
			ChannelService: channel.NewService(&database.Database{Client: client}),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/channels", strings.NewReader(channelJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)

	if assert.NoError(t, h.CreateChannel(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)

		// Check response body
		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)
		assert.Equal(t, "test_channel", response["name"])
	}
}
