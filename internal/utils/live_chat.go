package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/rs/zerolog/log"
)

type LiveCommentBadgeIcon struct {
	Height int    `json:"height"`
	ID     string `json:"id"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
}

type LiveCommentBadge struct {
	ClickAction string                 `json:"click_action"`
	ClickURL    string                 `json:"click_url"`
	Description string                 `json:"description"`
	Icons       []LiveCommentBadgeIcon `json:"icons"`
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Title       string                 `json:"title"`
	Version     interface{}            `json:"version"`
}

type LiveCommentEmoteImage struct {
	Height int    `json:"height"`
	ID     string `json:"id"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
}

type LiveCommentEmote struct {
	ID        string                  `json:"id"`
	Images    []LiveCommentEmoteImage `json:"images"`
	Locations []string                `json:"locations"`
	Name      string                  `json:"name"`
}

type LiveCommentReply struct {
	ParentMsgID       string `json:"parent_msg_id"`
	ParentUserID      string `json:"parent_user_id"`
	ParentUserLogin   string `json:"parent_user_login"`
	ParentDisplayName string `json:"parent_display_name"`
	ParentMsgBody     string `json:"parent_msg_body"`
}

type LiveComment struct {
	ActionType string `json:"action_type"`
	Author     struct {
		Badges       []LiveCommentBadge `json:"badges"`
		DisplayName  string             `json:"display_name"`
		ID           string             `json:"id"`
		IsModerator  bool               `json:"is_moderator"`
		IsSubscriber bool               `json:"is_subscriber"`
		IsTurbo      bool               `json:"is_turbo"`
		Name         string             `json:"name"`
	} `json:"author"`
	ChannelID        string             `json:"channel_id"`
	ClientNonce      string             `json:"client_nonce"`
	Colour           string             `json:"colour"`
	Emotes           []LiveCommentEmote `json:"emotes"`
	Flags            string             `json:"flags"`
	IsFirstMessage   bool               `json:"is_first_message"`
	Message          string             `json:"message"`
	MessageID        string             `json:"message_id"`
	MessageType      string             `json:"message_type"`
	ReturningChatter string             `json:"returning_chatter"`
	Timestamp        int64              `json:"timestamp"`
	UserType         string             `json:"user_type"`
	BitsSpent        int                `json:"bits_spent,omitempty"`
	IsAction         bool               `json:"is_action,omitempty"`
	CustomRewardID   string             `json:"custom_reward_id,omitempty"`
	Reply            *LiveCommentReply  `json:"reply,omitempty"`
	UserNoticeParams map[string]string  `json:"user_notice_params,omitempty"`
}

func OpenLiveChatFile(path string) ([]LiveComment, error) {

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
