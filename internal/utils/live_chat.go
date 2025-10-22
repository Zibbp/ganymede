package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

type LiveComment struct {
	ActionType string `json:"action_type"`
	Author     struct {
		Badges []struct {
			ClickAction string `json:"click_action"`
			ClickURL    string `json:"click_url"`
			Description string `json:"description"`
			Icons       []struct {
				Height int    `json:"height"`
				ID     string `json:"id"`
				URL    string `json:"url"`
				Width  int    `json:"width"`
			} `json:"icons"`
			ID      string      `json:"id"`
			Name    string      `json:"name"`
			Title   string      `json:"title"`
			Version interface{} `json:"version"`
		} `json:"badges"`
		DisplayName  string `json:"display_name"`
		ID           string `json:"id"`
		IsModerator  bool   `json:"is_moderator"`
		IsSubscriber bool   `json:"is_subscriber"`
		IsTurbo      bool   `json:"is_turbo"`
		Name         string `json:"name"`
	} `json:"author"`
	ChannelID   string `json:"channel_id"`
	ClientNonce string `json:"client_nonce"`
	Colour      string `json:"colour"`
	Emotes      []struct {
		ID     string `json:"id"`
		Images []struct {
			Height int    `json:"height"`
			ID     string `json:"id"`
			URL    string `json:"url"`
			Width  int    `json:"width"`
		} `json:"images"`
		Locations []string `json:"locations"`
		Name      string   `json:"name"`
	} `json:"emotes"`
	Flags            string `json:"flags"`
	IsFirstMessage   bool   `json:"is_first_message"`
	Message          string `json:"message"`
	MessageID        string `json:"message_id"`
	MessageType      string `json:"message_type"`
	ReturningChatter string `json:"returning_chatter"`
	Timestamp        int64  `json:"timestamp"`
	UserType         string `json:"user_type"`
}

type KickWebSocketRawMsg struct {
	Event   string `json:"event"`
	Data    string `json:"data"`
	Channel string `json:"channel"`
}

type KickWebSocketChatMessage struct {
	ID         string    `json:"id"`
	ChatroomID int       `json:"chatroom_id"`
	Content    string    `json:"content"`
	Type       string    `json:"type"`
	CreatedAt  time.Time `json:"created_at"`
	Sender     struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Slug     string `json:"slug"`
		Identity struct {
			Color  string        `json:"color"`
			Badges []interface{} `json:"badges"`
		} `json:"identity"`
	} `json:"sender"`
	Metadata struct {
		MessageRef string `json:"message_ref"`
	} `json:"metadata"`
}

func OpenTwitchLiveChatFile(path string) ([]LiveComment, error) {

	liveChatJsonFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open chat file: %v", err)
	}
	defer func() {
		if err := liveChatJsonFile.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing chat file")
		}
	}()
	byteValue, _ := io.ReadAll(liveChatJsonFile)

	var liveComments []LiveComment
	err = json.Unmarshal(byteValue, &liveComments)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal chat file: %v", err)
	}
	return liveComments, nil
}

func OpenKickLiveChatFile(path string) ([]KickWebSocketChatMessage, error) {
	liveChatJsonFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open chat file: %v", err)
	}
	defer func() {
		if err := liveChatJsonFile.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing chat file")
		}
	}()
	byteValue, _ := io.ReadAll(liveChatJsonFile)

	var liveComments []KickWebSocketRawMsg
	err = json.Unmarshal(byteValue, &liveComments)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal chat file: %v", err)
	}
	var liveCommentsParsed []KickWebSocketChatMessage
	for _, msg := range liveComments {
		if msg.Event == "App\\Events\\ChatMessageEvent" {
			var chatMessage KickWebSocketChatMessage
			err = json.Unmarshal([]byte(msg.Data), &chatMessage)
			if err != nil {
				log.Debug().Err(err).Msg("failed to unmarshal chat message")
				continue
			}
			liveCommentsParsed = append(liveCommentsParsed, chatMessage)
		}
	}

	return liveCommentsParsed, nil
}
