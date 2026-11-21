package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/internal/utils"
)

func TestReplaceKickEmoteTokens(t *testing.T) {
	body, emotes := replaceKickEmoteTokens("hi [emote:37226:KEKW] x [emote:1:a] [emote:37226:KEKW]")
	require.Equal(t, "hi KEKW x a KEKW", body)
	require.Len(t, emotes, 2)
	require.Equal(t, "37226", emotes[0].ID)
	require.Equal(t, []string{"3-6", "12-15"}, emotes[0].Locations)
	require.Equal(t, []string{"10-10"}, emotes[1].Locations)
	for _, e := range emotes {
		for _, loc := range e.Locations {
			var a, b int
			_, err := fmtSscanLoc(loc, &a, &b)
			require.NoError(t, err)
			require.Equal(t, e.Name, body[a:b+1])
		}
	}

	plain, none := replaceKickEmoteTokens("no emotes here")
	require.Equal(t, "no emotes here", plain)
	require.Empty(t, none)
}

func TestReplaceKickEmoteTokensMultibyte(t *testing.T) {
	body, emotes := replaceKickEmoteTokens("🥥 [emote:5:pog]")
	require.Equal(t, "🥥 pog", body)
	require.Equal(t, []string{"5-7"}, emotes[0].Locations)
	require.Equal(t, "pog", body[5:8])
}

const kickSampleMessage = `{"id":"25a07e86-0f34-482d-a83d-03a7d7b3e399","chatroom_id":668,"content":"like brake [emote:5748004:collectiblesGoldenNAHH]","type":"message","created_at":"2026-11-20T03:03:23+00:00","sender":{"id":113930042,"username":"ExodiaKa","slug":"exodiaka","identity":{"color":"#FFAE76","badges":[{"type":"subscriber","text":"Subscriber","count":1}],"badges_v2":[{"name":"Flyby","badge_type":"global","image_url":"https://ext.cdn.kick.com/chat/badges/Flyby.png"}]}}}`

func TestConvertKickChatMessage(t *testing.T) {
	var msg kickChatMessage
	require.NoError(t, json.Unmarshal([]byte(kickSampleMessage), &msg))
	c := convertKickChatMessage(msg, time.Now())

	require.Equal(t, "like brake collectiblesGoldenNAHH", c.Message)
	require.Equal(t, "exodiaka", c.Author.Name)
	require.Equal(t, "ExodiaKa", c.Author.DisplayName)
	require.Equal(t, "113930042", c.Author.ID)
	require.True(t, c.Author.IsSubscriber)
	require.Equal(t, "#FFAE76", c.Colour)
	require.Equal(t, time.Date(2026, 11, 20, 3, 3, 23, 0, time.UTC).UnixMicro(), c.Timestamp)
	require.Len(t, c.Author.Badges, 1)
	require.Equal(t, "Flyby_"+kickBadgeVersion("https://ext.cdn.kick.com/chat/badges/Flyby.png"), c.Author.Badges[0].Name)
	require.Equal(t, "Flyby", c.Author.Badges[0].Title)
	require.Equal(t, "1", c.Author.Badges[0].Version)
	require.Equal(t, "https://ext.cdn.kick.com/chat/badges/Flyby.png", c.Author.Badges[0].Icons[0].URL)
}

func TestConvertKickChatMessageReply(t *testing.T) {
	raw := `{"id":"r1","chatroom_id":1,"content":"yes","type":"reply","created_at":"2026-11-20T03:03:23+00:00","sender":{"id":2,"username":"Bob","slug":"bob","identity":{"color":"","badges":[{"type":"moderator","text":"Moderator"}],"badges_v2":[]}},"metadata":{"original_sender":{"id":"7","username":"Alice"},"original_message":{"id":"m0","content":"ok? [emote:9:pog]"}}}`
	var msg kickChatMessage
	require.NoError(t, json.Unmarshal([]byte(raw), &msg))
	c := convertKickChatMessage(msg, time.Now())
	require.True(t, c.Author.IsModerator)
	require.NotNil(t, c.Reply)
	require.Equal(t, "m0", c.Reply.ParentMsgID)
	require.Equal(t, "Alice", c.Reply.ParentDisplayName)
	require.Equal(t, "ok? pog", c.Reply.ParentMsgBody)
	require.Equal(t, "moderator", c.Author.Badges[0].Name)
}

// Kick comments must survive the shared live-chat -> archive converter with
// emote fragments aligned to the message text.
func TestKickChatThroughTDLConverter(t *testing.T) {
	dir := t.TempDir()
	var msg kickChatMessage
	require.NoError(t, json.Unmarshal([]byte(kickSampleMessage), &msg))
	start := time.Date(2026, 11, 20, 3, 0, 0, 0, time.UTC)
	c := convertKickChatMessage(msg, start)

	livePath := filepath.Join(dir, "live.json")
	data, err := json.Marshal([]utils.LiveComment{c})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(livePath, data, 0o644))

	outPath := filepath.Join(dir, "convert.json")
	require.NoError(t, utils.ConvertTwitchLiveChatToTDLChat(livePath, outPath, "xqc", "vid", "123", 668, start, "", 0))

	var converted struct {
		Comments []struct {
			ContentOffsetSeconds float64 `json:"content_offset_seconds"`
			Message              struct {
				Body      string `json:"body"`
				Fragments []struct {
					Text     string `json:"text"`
					Emoticon *struct {
						EmoticonID string `json:"emoticon_id"`
					} `json:"emoticon"`
				} `json:"fragments"`
			} `json:"message"`
		} `json:"comments"`
	}
	out, err := os.ReadFile(outPath)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(out, &converted))
	// The converter prepends a synthetic "Initial chat message" comment.
	require.Len(t, converted.Comments, 2)
	cm := converted.Comments[1]
	require.Equal(t, "like brake collectiblesGoldenNAHH", cm.Message.Body)
	require.InDelta(t, 203, cm.ContentOffsetSeconds, 1)
	var emoteFrag bool
	for _, f := range cm.Message.Fragments {
		if f.Emoticon != nil && f.Emoticon.EmoticonID == "5748004" {
			require.Equal(t, "collectiblesGoldenNAHH", f.Text)
			emoteFrag = true
		}
	}
	require.True(t, emoteFrag, "expected an emote fragment")
}

func fmtSscanLoc(loc string, a, b *int) (int, error) {
	return fmt.Sscanf(loc, "%d-%d", a, b)
}

func TestFetchKickAssets(t *testing.T) {
	t.Parallel()
	png := []byte("\x89PNG\r\n\x1a\n")
	var flaky atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			_, _ = w.Write(png)
		case "/gone":
			w.WriteHeader(http.StatusNotFound)
		case "/flaky":
			if flaky.Add(1) == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			_, _ = w.Write(png)
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer srv.Close()

	assets, err := fetchKickAssets(context.Background(), []string{srv.URL + "/ok", srv.URL + "/gone", srv.URL + "/flaky"})
	require.NoError(t, err)
	assert.Contains(t, assets, srv.URL+"/ok")
	assert.Contains(t, assets, srv.URL+"/flaky", "transient failure should be retried")
	assert.NotContains(t, assets, srv.URL+"/gone")

	_, err = fetchKickAssets(context.Background(), []string{srv.URL + "/ok", srv.URL + "/down"})
	assert.Error(t, err, "persistent transient failure must fail the embed")
}
