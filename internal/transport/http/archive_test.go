package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/enttest"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/queue"
	"github.com/zibbp/ganymede/internal/twitch"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
)

var (
	testArchiveChannelJson = `{
		"channel_name": "staysafetv"
	}`
)

func TestArchiveTwitchChannel(t *testing.T) {
	// Load environment variables (for local testing)
	_ = godotenv.Load("../../../.env.dev")

	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	twitchService := twitch.NewService()
	channelService := channel.NewService(&database.Database{Client: client})
	vodService := vod.NewService(&database.Database{Client: client})
	queueService := queue.NewService(&database.Database{Client: client}, vodService, channelService)

	// Authenticate with Twitch
	twitch.Authenticate()

	h := &Handler{
		Server: echo.New(),
		Service: Services{
			ArchiveService: archive.NewService(&database.Database{Client: client}, twitchService, channelService, vodService, queueService),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/archive/channel", strings.NewReader(testArchiveChannelJson))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)

	if assert.NoError(t, h.ArchiveTwitchChannel(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)

		// Check response body
		var response map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &response)
		assert.Equal(t, "staysafetv", response["name"])
	}

}
