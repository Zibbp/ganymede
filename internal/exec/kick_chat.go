package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/utils"
)

const (
	kickChatMessageEvent   = `App\Events\ChatMessageEvent`
	kickPusherVersion      = "8.4.0"
	kickChatHeartbeatCheck = 5 * time.Second
	kickChatPongTimeout    = 30 * time.Second
)

var kickEmoteTokenRe = regexp.MustCompile(`\[emote:(\d+):([^\]]*)\]`)

type kickPusherFrame struct {
	Event   string         `json:"event"`
	Data    kickPusherData `json:"data"`
	Channel string         `json:"channel,omitempty"`
}

// kickPusherData is an event payload, which Pusher sends either as a JSON
// string (chat) or as a plain JSON value (errors).
type kickPusherData string

func (d *kickPusherData) UnmarshalJSON(raw []byte) error {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		*d = kickPusherData(s)
		return nil
	}
	*d = kickPusherData(raw)
	return nil
}

type kickChatBadge struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Count int    `json:"count"`
}

type kickChatBadgeV2 struct {
	Name      string `json:"name"`
	BadgeType string `json:"badge_type"`
	ImageURL  string `json:"image_url"`
}

type kickChatMessage struct {
	ID         string `json:"id"`
	ChatroomID int64  `json:"chatroom_id"`
	Content    string `json:"content"`
	Type       string `json:"type"`
	CreatedAt  string `json:"created_at"`
	Sender     struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Slug     string `json:"slug"`
		Identity struct {
			Color    string            `json:"color"`
			Badges   []kickChatBadge   `json:"badges"`
			BadgesV2 []kickChatBadgeV2 `json:"badges_v2"`
		} `json:"identity"`
	} `json:"sender"`
	Metadata *struct {
		OriginalSender *struct {
			ID       json.Number `json:"id"`
			Username string      `json:"username"`
		} `json:"original_sender"`
		OriginalMessage *struct {
			ID      string `json:"id"`
			Content string `json:"content"`
		} `json:"original_message"`
	} `json:"metadata"`
}

// kickBadgeVersion distinguishes badge images sharing a name (e.g. levels).
func kickBadgeVersion(imageURL string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(imageURL))
	return strconv.FormatUint(uint64(h.Sum32()), 36)
}

// replaceKickEmoteTokens turns "[emote:ID:name]" tokens into the emote name
// and returns Twitch-style "start-end" (inclusive, byte offset) locations.
func replaceKickEmoteTokens(content string) (string, []utils.LiveCommentEmote) {
	matches := kickEmoteTokenRe.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, nil
	}
	var b strings.Builder
	byID := map[string]int{}
	var emotes []utils.LiveCommentEmote
	last := 0
	for _, m := range matches {
		b.WriteString(content[last:m[0]])
		id := content[m[2]:m[3]]
		name := content[m[4]:m[5]]
		if name == "" {
			name = id
		}
		start := b.Len()
		b.WriteString(name)
		loc := fmt.Sprintf("%d-%d", start, start+len(name)-1)
		if idx, ok := byID[id]; ok {
			emotes[idx].Locations = append(emotes[idx].Locations, loc)
		} else {
			byID[id] = len(emotes)
			emotes = append(emotes, utils.LiveCommentEmote{
				ID:        id,
				Name:      name,
				Locations: []string{loc},
				Images:    []utils.LiveCommentEmoteImage{{ID: id, URL: platform.KickEmoteURL(id)}},
			})
		}
		last = m[1]
	}
	b.WriteString(content[last:])
	return b.String(), emotes
}

func convertKickChatMessage(msg kickChatMessage, receivedAt time.Time) utils.LiveComment {
	body, emotes := replaceKickEmoteTokens(msg.Content)
	ts := receivedAt
	if t, err := time.Parse(time.RFC3339, msg.CreatedAt); err == nil {
		ts = t
	}
	comment := utils.LiveComment{
		ActionType:  "add_chat_message",
		ChannelID:   strconv.FormatInt(msg.ChatroomID, 10),
		ClientNonce: msg.ID,
		Colour:      msg.Sender.Identity.Color,
		Message:     body,
		MessageID:   msg.ID,
		MessageType: "text",
		Timestamp:   ts.UnixMicro(),
		Emotes:      emotes,
	}
	comment.Author.ID = strconv.FormatInt(msg.Sender.ID, 10)
	comment.Author.Name = msg.Sender.Slug
	comment.Author.DisplayName = msg.Sender.Username
	if comment.Author.Name == "" {
		comment.Author.Name = strings.ToLower(msg.Sender.Username)
	}

	for _, b := range msg.Sender.Identity.Badges {
		switch b.Type {
		case "moderator", "broadcaster":
			comment.Author.IsModerator = true
		case "subscriber", "founder":
			comment.Author.IsSubscriber = true
		}
	}
	if len(msg.Sender.Identity.BadgesV2) > 0 {
		for _, b := range msg.Sender.Identity.BadgesV2 {
			if b.Name == "" || b.ImageURL == "" {
				continue
			}
			// One badge name per image: Kick reuses names like "level" for
			// different images, and the playback API keeps one version of
			// non-Twitch-special badges.
			name := b.Name + "_" + kickBadgeVersion(b.ImageURL)
			comment.Author.Badges = append(comment.Author.Badges, utils.LiveCommentBadge{
				ID:      name,
				Name:    name,
				Title:   b.Name,
				Version: "1",
				Icons:   []utils.LiveCommentBadgeIcon{{ID: name, URL: b.ImageURL}},
			})
		}
	} else {
		for _, b := range msg.Sender.Identity.Badges {
			version := "1"
			if b.Count > 0 {
				version = strconv.Itoa(b.Count)
			}
			comment.Author.Badges = append(comment.Author.Badges, utils.LiveCommentBadge{
				ID: b.Type, Name: b.Type, Title: b.Text, Version: version,
			})
		}
	}

	if msg.Type == "reply" && msg.Metadata != nil && msg.Metadata.OriginalMessage != nil {
		reply := &utils.LiveCommentReply{
			ParentMsgID:   msg.Metadata.OriginalMessage.ID,
			ParentMsgBody: msg.Metadata.OriginalMessage.Content,
		}
		if msg.Metadata.OriginalSender != nil {
			reply.ParentUserID = msg.Metadata.OriginalSender.ID.String()
			reply.ParentDisplayName = msg.Metadata.OriginalSender.Username
			reply.ParentUserLogin = strings.ToLower(msg.Metadata.OriginalSender.Username)
		}
		reply.ParentMsgBody, _ = replaceKickEmoteTokens(reply.ParentMsgBody)
		comment.Reply = reply
	}
	return comment
}

// SaveKickLiveChatToFile records a Kick channel's chat over Kick's Pusher
// websocket into the same live chat JSON format used for Twitch, until ctx is
// cancelled. Connection drops are retried indefinitely with backoff because the
// capture lifetime, not the chat connection, bounds the job.
func SaveKickLiveChatToFile(ctx context.Context, slug, filename string) error {
	const (
		initialBackoff = 2 * time.Second
		maxBackoff     = 1 * time.Minute
	)
	logger := log.With().Str("channel", slug).Str("platform", "kick").Str("chat_file", filename).Logger()

	writer, err := newLiveChatJSONArrayWriter(filename)
	if err != nil {
		return fmt.Errorf("failed to initialize live chat writer: %w", err)
	}
	defer func() {
		if err := writer.Finalize(); err != nil {
			logger.Error().Err(err).Msg("failed to finalize live chat writer")
		}
	}()

	// Merge pending messages into the primary JSON on the same cadence as the
	// Twitch recorder, so the main chat file stays current during long streams.
	flushDone := make(chan struct{})
	defer close(flushDone)
	go func() {
		ticker := time.NewTicker(liveChatMainFlushInterval)
		defer ticker.Stop()
		for {
			select {
			case <-flushDone:
				return
			case <-ticker.C:
				if err := writer.Flush(); err != nil {
					logger.Error().Err(err).Msg("error flushing pending kick chat messages to primary JSON")
				}
			}
		}
	}()

	kc := platform.NewKickConnection()
	var saved atomic.Int64
	backoff := initialBackoff
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		connectedAt := time.Now()
		err := runKickChatSession(ctx, kc, slug, writer, &saved, logger)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// A session that stayed up for a while resets the backoff.
		if time.Since(connectedAt) > 5*time.Minute {
			backoff = initialBackoff
		}
		logger.Warn().Err(err).Dur("retry_in", backoff).Int64("messages_saved", saved.Load()).Msg("kick chat connection lost; reconnecting")
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, maxBackoff)
	}
}

func runKickChatSession(ctx context.Context, kc *platform.KickConnection, slug string, writer *liveChatJSONArrayWriter, saved *atomic.Int64, logger zerolog.Logger) error {
	ch, err := kc.GetKickChannel(ctx, slug)
	if err != nil {
		return err
	}
	if ch.Chatroom.ID == 0 {
		return fmt.Errorf("kick channel %s has no chatroom", slug)
	}
	appKey, cluster := kc.KickRealtimeCredentials(ctx, ch.ID)
	wsURL := fmt.Sprintf("wss://ws-%s.pusher.com/app/%s?protocol=7&client=js&version=%s&flash=false", cluster, appKey, kickPusherVersion)

	header := http.Header{}
	header.Set("User-Agent", utils.ChromeUserAgent)
	header.Set("Origin", "https://kick.com")
	dialer := websocket.Dialer{HandshakeTimeout: 20 * time.Second}
	conn, _, err := dialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return fmt.Errorf("dial kick chat: %w", err)
	}
	defer conn.Close() //nolint:errcheck

	var writeMu sync.Mutex
	writeFrame := func(event string, data any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		return conn.WriteJSON(map[string]any{"event": event, "data": data})
	}

	var activityTimeout atomic.Int64
	activityTimeout.Store(int64(120 * time.Second))
	var lastRead atomic.Int64
	lastRead.Store(time.Now().UnixNano())

	// Closing the connection is the only reliable way to stop a blocked read
	// on cancellation. While idle, ping before the read deadline expires,
	// because a timed-out gorilla read cannot be resumed.
	sessionDone := make(chan struct{})
	defer close(sessionDone)
	go func() {
		ticker := time.NewTicker(kickChatHeartbeatCheck)
		defer ticker.Stop()
		var lastPing time.Time
		for {
			select {
			case <-ctx.Done():
				_ = conn.Close()
				return
			case <-sessionDone:
				return
			case <-ticker.C:
				timeout := time.Duration(activityTimeout.Load())
				idle := time.Since(time.Unix(0, lastRead.Load()))
				if idle >= timeout && time.Since(lastPing) >= timeout {
					if err := writeFrame("pusher:ping", map[string]any{}); err != nil {
						logger.Debug().Err(err).Msg("failed to send pusher ping")
					}
					lastPing = time.Now()
				}
			}
		}
	}()

	subscribed := false
	for {
		// A pong or any other frame must arrive within the grace period after a ping.
		_ = conn.SetReadDeadline(time.Now().Add(time.Duration(activityTimeout.Load()) + kickChatPongTimeout))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("read kick chat: %w", err)
		}
		lastRead.Store(time.Now().UnixNano())

		var frame kickPusherFrame
		if err := json.Unmarshal(raw, &frame); err != nil {
			continue
		}
		switch frame.Event {
		case "pusher:connection_established":
			var est struct {
				ActivityTimeout int `json:"activity_timeout"`
			}
			if json.Unmarshal([]byte(frame.Data), &est) == nil && est.ActivityTimeout > 0 {
				activityTimeout.Store(int64(time.Duration(est.ActivityTimeout) * time.Second))
			}
			channel := fmt.Sprintf("chatrooms.%d.v2", ch.Chatroom.ID)
			if err := writeFrame("pusher:subscribe", map[string]any{"auth": "", "channel": channel}); err != nil {
				return fmt.Errorf("subscribe kick chat: %w", err)
			}
		case "pusher_internal:subscription_succeeded":
			if !subscribed {
				subscribed = true
				logger.Info().Int64("chatroom_id", ch.Chatroom.ID).Msgf("connected to %s kick chat room", slug)
			}
		case "pusher:ping":
			if err := writeFrame("pusher:pong", map[string]any{}); err != nil {
				return fmt.Errorf("send pusher pong: %w", err)
			}
		case "pusher:error":
			return fmt.Errorf("pusher error: %s", frame.Data)
		case kickChatMessageEvent:
			var msg kickChatMessage
			if err := json.Unmarshal([]byte(frame.Data), &msg); err != nil {
				logger.Debug().Err(err).Msg("skipping undecodable kick chat message")
				continue
			}
			if err := writer.Append(convertKickChatMessage(msg, time.Now())); err != nil {
				logger.Error().Err(err).Str("message_id", msg.ID).Msg("error saving kick chat message")
				continue
			}
			if n := saved.Add(1); n == 1 || n%500 == 0 {
				logger.Debug().Int64("messages_saved", n).Msg("kick chat messages saved")
			}
		}
	}
}
