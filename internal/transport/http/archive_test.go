package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/enttest"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/queue"
	httpHandler "github.com/zibbp/ganymede/internal/transport/http"
	"github.com/zibbp/ganymede/internal/twitch"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
)

var (
	// The following are used for testing.
	testArchiveChannelJson = `{
		"channel_name": "test"
		}`
)

type ServiceFuncMock struct{}

func (m ServiceFuncMock) GetUserByLogin(login string) (twitch.Channel, error) {
	return twitch.Channel{
		ID:              "123",
		Login:           "test",
		DisplayName:     "test",
		ProfileImageURL: "https://raw.githubusercontent.com/Zibbp/ganymede/main/.github/ganymede-logo.png",
	}, nil
}

// * TestArchiveChannel tests the archiving of a twitch channel functionality.
// Test fetches a mock channel, creates a db entry, and downloads the channel image.
func TestArchiveTwitchChannel(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	twitch.API = ServiceFuncMock{}

	twitchService := twitch.NewService()
	vodService := vod.NewService(&database.Database{Client: client})
	channelService := channel.NewService(&database.Database{Client: client})
	queueService := queue.NewService(&database.Database{Client: client}, vodService, channelService)

	archiveService := archive.NewService(&database.Database{Client: client}, twitchService, channelService, vodService, queueService)

	h := &httpHandler.Handler{
		Server: echo.New(),
		Service: httpHandler.Services{
			ArchiveService: archiveService,
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
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "test", response["name"])

		// Check channel folder was created
		_, err = os.Stat("/vods/test")
		assert.NoError(t, err)

		// Check channel image was downloaded
		_, err = os.Stat("/vods/test/profile.png")
		assert.NoError(t, err)
	}
}
