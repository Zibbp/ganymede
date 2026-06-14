package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConvertTwitchLiveChatToTDLChatKeepsMessagesAndUserNotices(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "live-chat.json")
	outputPath := filepath.Join(tmpDir, "tdl-chat.json")
	chatStart := time.Unix(1_700_000_000, 0)

	normalComment := LiveComment{
		ActionType:  "add_chat_message",
		ChannelID:   "408892348",
		Colour:      "#1FD2FF",
		Message:     "hello chat",
		MessageID:   "normal-message-id",
		MessageType: "text",
		Timestamp:   chatStart.Add(2 * time.Second).UnixMicro(),
		BitsSpent:   50,
		IsAction:    true,
		Reply: &LiveCommentReply{
			ParentMsgID:       "parent-message-id",
			ParentUserID:      "222",
			ParentUserLogin:   "parentuser",
			ParentDisplayName: "ParentUser",
			ParentMsgBody:     "original message",
		},
	}
	normalComment.Author.DisplayName = "NormalUser"
	normalComment.Author.ID = "111"
	normalComment.Author.Name = "normaluser"
	normalComment.Author.IsSubscriber = true
	normalComment.Author.Badges = []LiveCommentBadge{
		{Name: "subscriber", Version: 12},
	}

	userNoticeComment := LiveComment{
		ActionType:  "add_chat_message",
		ChannelID:   "408892348",
		Colour:      "#00FF7F",
		Message:     "NormalUser just subscribed with a Tier 1 sub. Great stream!",
		MessageID:   "notice-message-id",
		MessageType: "user_notice",
		Timestamp:   chatStart.Add(5 * time.Second).UnixMicro(),
		UserNoticeParams: map[string]string{
			"msg-id":                  "resub",
			"system-msg":              "NormalUser just subscribed with a Tier 1 sub.",
			"msg-param-months":        "12",
			"msg-param-sub-plan":      "1000",
			"msg-param-sub-plan-name": "Channel Subscription",
		},
	}
	userNoticeComment.Author.DisplayName = "NormalUser"
	userNoticeComment.Author.ID = "111"
	userNoticeComment.Author.Name = "normaluser"

	input, err := json.Marshal([]LiveComment{normalComment, userNoticeComment})
	if err != nil {
		t.Fatalf("failed to marshal live comments: %v", err)
	}
	if err := os.WriteFile(inputPath, input, 0o644); err != nil {
		t.Fatalf("failed to write live comments: %v", err)
	}

	err = ConvertTwitchLiveChatToTDLChat(inputPath, outputPath, "clippyassistant", "video-id", "external-id", 408892348, chatStart, "previous-video-id")
	if err != nil {
		t.Fatalf("ConvertTwitchLiveChatToTDLChat returned error: %v", err)
	}

	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output chat: %v", err)
	}

	var chat TDLChat
	if err := json.Unmarshal(output, &chat); err != nil {
		t.Fatalf("failed to unmarshal output chat: %v", err)
	}

	if len(chat.Comments) != 3 {
		t.Fatalf("expected initial comment plus two live comments, got %d", len(chat.Comments))
	}

	convertedNormal := chat.Comments[1]
	if convertedNormal.ID != normalComment.MessageID {
		t.Fatalf("expected normal comment ID %q, got %q", normalComment.MessageID, convertedNormal.ID)
	}
	if convertedNormal.Message.Body != normalComment.Message {
		t.Fatalf("expected normal body %q, got %q", normalComment.Message, convertedNormal.Message.Body)
	}
	if convertedNormal.Message.BitsSpent != normalComment.BitsSpent {
		t.Fatalf("expected bits spent %d, got %d", normalComment.BitsSpent, convertedNormal.Message.BitsSpent)
	}
	if !convertedNormal.Message.IsAction {
		t.Fatal("expected action flag")
	}
	if convertedNormal.Message.Reply == nil {
		t.Fatal("expected reply metadata")
	}
	if convertedNormal.Message.Reply.ParentMsgID != "parent-message-id" {
		t.Fatalf("expected reply parent message ID, got %q", convertedNormal.Message.Reply.ParentMsgID)
	}
	if len(convertedNormal.Message.UserBadges) != 1 || convertedNormal.Message.UserBadges[0].ID != "subscriber" {
		t.Fatalf("expected subscriber badge, got %#v", convertedNormal.Message.UserBadges)
	}

	convertedNotice := chat.Comments[2]
	if convertedNotice.ID != userNoticeComment.MessageID {
		t.Fatalf("expected notice comment ID %q, got %q", userNoticeComment.MessageID, convertedNotice.ID)
	}
	if convertedNotice.Message.Body != userNoticeComment.Message {
		t.Fatalf("expected notice body %q, got %q", userNoticeComment.Message, convertedNotice.Message.Body)
	}
	if convertedNotice.Message.UserNoticeParams.MsgID == nil {
		t.Fatal("expected notice msg-id")
	}
	if *convertedNotice.Message.UserNoticeParams.MsgID != "resub" {
		t.Fatalf("expected notice msg-id resub, got %q", *convertedNotice.Message.UserNoticeParams.MsgID)
	}
	if convertedNotice.Message.UserNoticeParams.SystemMsg != userNoticeComment.UserNoticeParams["system-msg"] {
		t.Fatalf("expected notice system message, got %q", convertedNotice.Message.UserNoticeParams.SystemMsg)
	}
	if convertedNotice.Message.UserNoticeParams.Params["msg-param-months"] != "12" {
		t.Fatalf("expected notice params, got %#v", convertedNotice.Message.UserNoticeParams.Params)
	}
}

func TestEnrichTwitchChatMetadataFromLiveChat(t *testing.T) {
	tmpDir := t.TempDir()
	liveChatPath := filepath.Join(tmpDir, "live-chat.json")
	chatPath := filepath.Join(tmpDir, "chat.json")
	chatStart := time.Unix(1_700_000_000, 0)

	liveComments := []LiveComment{
		{
			Message:   "hello chat",
			MessageID: "normal-message-id",
			Timestamp: chatStart.UnixMicro(),
		},
		{
			Message:   "reply body",
			MessageID: "reply-message-id",
			Timestamp: chatStart.Add(time.Second).UnixMicro(),
			BitsSpent: 25,
			IsAction:  true,
			Reply: &LiveCommentReply{
				ParentMsgID:       "parent-id",
				ParentUserID:      "222",
				ParentUserLogin:   "parentuser",
				ParentDisplayName: "ParentUser",
				ParentMsgBody:     "parent body",
			},
		},
		{
			Message:   "Somebody subscribed!",
			MessageID: "notice-message-id",
			Timestamp: chatStart.Add(2 * time.Second).UnixMicro(),
			UserNoticeParams: map[string]string{
				"msg-id":           "sub",
				"system-msg":       "Somebody subscribed!",
				"msg-param-months": "1",
			},
		},
	}

	liveInput, err := json.Marshal(liveComments)
	if err != nil {
		t.Fatalf("failed to marshal live comments: %v", err)
	}
	if err := os.WriteFile(liveChatPath, liveInput, 0o644); err != nil {
		t.Fatalf("failed to write live chat: %v", err)
	}

	chatInput := []byte(`{
		"streamer":{"name":"channel","id":123},
		"comments":[
			{"_id":"normal-message-id","message":{"body":"hello chat","bits_spent":0,"is_action":false}},
			{"_id":"reply-message-id","message":{"body":"reply body","bits_spent":0,"is_action":false}},
			{"_id":"notice-message-id","message":{"body":"Somebody subscribed!","bits_spent":0,"is_action":false}}
		],
		"embeddedData":{"kept":true}
	}`)
	if err := os.WriteFile(chatPath, chatInput, 0o644); err != nil {
		t.Fatalf("failed to write chat: %v", err)
	}

	if err := EnrichTwitchChatMetadataFromLiveChat(liveChatPath, chatPath); err != nil {
		t.Fatalf("EnrichTwitchChatMetadataFromLiveChat returned error: %v", err)
	}

	output, err := os.ReadFile(chatPath)
	if err != nil {
		t.Fatalf("failed to read enriched chat: %v", err)
	}

	var enriched struct {
		Comments []struct {
			ID      string `json:"_id"`
			Message struct {
				BitsSpent        int                       `json:"bits_spent"`
				IsAction         bool                      `json:"is_action"`
				Reply            *finalChatReply           `json:"reply"`
				UserNoticeParams finalChatUserNoticeParams `json:"user_notice_params"`
			} `json:"message"`
		} `json:"comments"`
		EmbeddedData map[string]bool `json:"embeddedData"`
	}
	if err := json.Unmarshal(output, &enriched); err != nil {
		t.Fatalf("failed to unmarshal enriched chat: %v", err)
	}

	if !enriched.EmbeddedData["kept"] {
		t.Fatal("expected unrelated top-level fields to be preserved")
	}
	if enriched.Comments[0].Message.Reply != nil || enriched.Comments[0].Message.UserNoticeParams.MsgID != "" {
		t.Fatalf("expected untouched normal comment, got %#v", enriched.Comments[0].Message)
	}
	if enriched.Comments[1].Message.BitsSpent != 25 || !enriched.Comments[1].Message.IsAction {
		t.Fatalf("expected bits/action metadata, got %#v", enriched.Comments[1].Message)
	}
	if enriched.Comments[1].Message.Reply == nil || enriched.Comments[1].Message.Reply.ParentMsgID != "parent-id" {
		t.Fatalf("expected reply metadata, got %#v", enriched.Comments[1].Message.Reply)
	}
	if enriched.Comments[2].Message.UserNoticeParams.MsgID != "sub" {
		t.Fatalf("expected user notice msg ID, got %#v", enriched.Comments[2].Message.UserNoticeParams)
	}
	if enriched.Comments[2].Message.UserNoticeParams.Params["msg-param-months"] != "1" {
		t.Fatalf("expected user notice params, got %#v", enriched.Comments[2].Message.UserNoticeParams.Params)
	}
}
