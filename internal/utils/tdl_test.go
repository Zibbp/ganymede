package utils

import (
	"encoding/json"
	"fmt"
	"io"
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
		ActionType:     "add_chat_message",
		ChannelID:      "408892348",
		Colour:         "#1FD2FF",
		Message:        "hello chat",
		MessageID:      "normal-message-id",
		MessageType:    "text",
		Timestamp:      chatStart.Add(2 * time.Second).UnixMicro(),
		BitsSpent:      50,
		IsAction:       true,
		IsFirstMessage: true,
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
	if !convertedNormal.Message.IsFirstMessage {
		t.Fatal("expected first message flag")
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

func TestStreamLiveCommentsProcessesCommentBeforeEOF(t *testing.T) {
	firstComment := LiveComment{
		Message:   "first",
		MessageID: "first-message-id",
	}
	secondComment := LiveComment{
		Message:   "second",
		MessageID: "second-message-id",
	}

	firstJSON, err := json.Marshal(firstComment)
	if err != nil {
		t.Fatalf("failed to marshal first comment: %v", err)
	}
	secondJSON, err := json.Marshal(secondComment)
	if err != nil {
		t.Fatalf("failed to marshal second comment: %v", err)
	}

	reader, writer := io.Pipe()
	firstProcessed := make(chan struct{})
	producerErr := make(chan error, 1)

	go func() {
		if _, err := fmt.Fprintf(writer, "[%s,", firstJSON); err != nil {
			producerErr <- err
			return
		}

		select {
		case <-firstProcessed:
		case <-time.After(2 * time.Second):
			err := fmt.Errorf("first comment was not processed before EOF")
			_ = writer.CloseWithError(err)
			producerErr <- err
			return
		}

		if _, err := fmt.Fprintf(writer, "%s]", secondJSON); err != nil {
			producerErr <- err
			return
		}
		producerErr <- writer.Close()
	}()

	var messageIDs []string
	err = streamLiveComments(reader, func(comment LiveComment) error {
		messageIDs = append(messageIDs, comment.MessageID)
		if len(messageIDs) == 1 {
			close(firstProcessed)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("streamLiveComments returned error: %v", err)
	}
	if err := <-producerErr; err != nil {
		t.Fatalf("failed to produce streamed input: %v", err)
	}

	if len(messageIDs) != 2 {
		t.Fatalf("expected two streamed comments, got %d", len(messageIDs))
	}
	if messageIDs[0] != firstComment.MessageID || messageIDs[1] != secondComment.MessageID {
		t.Fatalf("unexpected streamed comment order: %#v", messageIDs)
	}
}

func TestConvertTwitchLiveChatToTDLChatStreamsLargeInput(t *testing.T) {
	const commentCount = 5_000

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "large-live-chat.json")
	outputPath := filepath.Join(tmpDir, "large-tdl-chat.json")
	chatStart := time.Unix(1_700_000_000, 0)

	inputFile, err := os.Create(inputPath)
	if err != nil {
		t.Fatalf("failed to create input chat: %v", err)
	}
	if _, err := inputFile.WriteString("["); err != nil {
		_ = inputFile.Close()
		t.Fatalf("failed to write input chat header: %v", err)
	}
	for i := 0; i < commentCount; i++ {
		comment := LiveComment{
			Message:   fmt.Sprintf("message-%d", i),
			MessageID: fmt.Sprintf("message-id-%d", i),
			Timestamp: chatStart.Add(time.Duration(i+1) * time.Second).UnixMicro(),
		}
		comment.Author.DisplayName = "StreamedUser"
		comment.Author.ID = "streamed-user-id"
		comment.Author.Name = "streameduser"

		data, err := json.Marshal(comment)
		if err != nil {
			_ = inputFile.Close()
			t.Fatalf("failed to marshal input comment %d: %v", i, err)
		}
		if i > 0 {
			if _, err := inputFile.WriteString(","); err != nil {
				_ = inputFile.Close()
				t.Fatalf("failed to write input separator: %v", err)
			}
		}
		if _, err := inputFile.Write(data); err != nil {
			_ = inputFile.Close()
			t.Fatalf("failed to write input comment %d: %v", i, err)
		}
	}
	if _, err := inputFile.WriteString("]"); err != nil {
		_ = inputFile.Close()
		t.Fatalf("failed to write input chat footer: %v", err)
	}
	if err := inputFile.Close(); err != nil {
		t.Fatalf("failed to close input chat: %v", err)
	}

	if err := ConvertTwitchLiveChatToTDLChat(inputPath, outputPath, "channel", "video-id", "external-id", 123, chatStart, "previous-video-id"); err != nil {
		t.Fatalf("ConvertTwitchLiveChatToTDLChat returned error: %v", err)
	}

	outputFile, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("failed to open output chat: %v", err)
	}
	defer outputFile.Close() //nolint:errcheck

	decoder := json.NewDecoder(outputFile)
	token, err := decoder.Token()
	if err != nil {
		t.Fatalf("failed to read output object: %v", err)
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		t.Fatalf("expected output object, got %v", token)
	}

	var video Video
	convertedCount := 0
	lastMessageID := ""
	for decoder.More() {
		fieldToken, err := decoder.Token()
		if err != nil {
			t.Fatalf("failed to read output field: %v", err)
		}
		field, ok := fieldToken.(string)
		if !ok {
			t.Fatalf("expected output field name, got %v", fieldToken)
		}

		switch field {
		case "video":
			if err := decoder.Decode(&video); err != nil {
				t.Fatalf("failed to decode video metadata: %v", err)
			}
		case "comments":
			token, err := decoder.Token()
			if err != nil {
				t.Fatalf("failed to read comments array: %v", err)
			}
			if delimiter, ok := token.(json.Delim); !ok || delimiter != '[' {
				t.Fatalf("expected comments array, got %v", token)
			}
			for decoder.More() {
				var comment Comment
				if err := decoder.Decode(&comment); err != nil {
					t.Fatalf("failed to decode streamed comment: %v", err)
				}
				convertedCount++
				lastMessageID = comment.ID
			}
			if _, err := decoder.Token(); err != nil {
				t.Fatalf("failed to read end of comments array: %v", err)
			}
		default:
			var ignored json.RawMessage
			if err := decoder.Decode(&ignored); err != nil {
				t.Fatalf("failed to skip output field %q: %v", field, err)
			}
		}
	}
	if _, err := decoder.Token(); err != nil {
		t.Fatalf("failed to read end of output object: %v", err)
	}

	if convertedCount != commentCount+1 {
		t.Fatalf("expected initial comment plus %d live comments, got %d", commentCount, convertedCount)
	}
	expectedLastMessageID := fmt.Sprintf("message-id-%d", commentCount-1)
	if lastMessageID != expectedLastMessageID {
		t.Fatalf("expected last message ID %q, got %q", expectedLastMessageID, lastMessageID)
	}
	if video.End != commentCount {
		t.Fatalf("expected video end %d, got %d", commentCount, video.End)
	}
}

func TestConvertTwitchLiveChatToTDLChatDoesNotReplaceOutputOnInvalidInput(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "invalid-live-chat.json")
	outputPath := filepath.Join(tmpDir, "existing-tdl-chat.json")
	originalOutput := []byte(`{"existing":true}`)

	commentJSON, err := json.Marshal(LiveComment{
		Message:   "valid first message",
		MessageID: "first-message-id",
		Timestamp: time.Now().UnixMicro(),
	})
	if err != nil {
		t.Fatalf("failed to marshal input comment: %v", err)
	}
	if err := os.WriteFile(inputPath, append(append([]byte("["), commentJSON...), ','), 0o644); err != nil {
		t.Fatalf("failed to write invalid input chat: %v", err)
	}
	if err := os.WriteFile(outputPath, originalOutput, 0o640); err != nil {
		t.Fatalf("failed to write existing output chat: %v", err)
	}

	err = ConvertTwitchLiveChatToTDLChat(inputPath, outputPath, "channel", "video-id", "external-id", 123, time.Now(), "previous-video-id")
	if err == nil {
		t.Fatal("expected conversion error for truncated input")
	}

	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read existing output chat: %v", err)
	}
	if string(output) != string(originalOutput) {
		t.Fatalf("expected existing output to remain unchanged, got %q", output)
	}

	tempFiles, err := filepath.Glob(outputPath + ".tmp-*")
	if err != nil {
		t.Fatalf("failed to inspect temporary output files: %v", err)
	}
	if len(tempFiles) != 0 {
		t.Fatalf("expected temporary output files to be cleaned up, got %#v", tempFiles)
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
			Message:        "reply body",
			MessageID:      "reply-message-id",
			Timestamp:      chatStart.Add(time.Second).UnixMicro(),
			BitsSpent:      25,
			IsAction:       true,
			IsFirstMessage: true,
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
				IsFirstMessage   bool                      `json:"is_first_message"`
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
	if len(enriched.Comments) < 3 {
		t.Fatalf("expected at least 3 enriched comments, got %d", len(enriched.Comments))
	}
	if enriched.Comments[0].Message.Reply != nil || enriched.Comments[0].Message.UserNoticeParams.MsgID != "" {
		t.Fatalf("expected untouched normal comment, got %#v", enriched.Comments[0].Message)
	}
	if enriched.Comments[1].Message.BitsSpent != 25 || !enriched.Comments[1].Message.IsAction {
		t.Fatalf("expected bits/action metadata, got %#v", enriched.Comments[1].Message)
	}
	if !enriched.Comments[1].Message.IsFirstMessage {
		t.Fatalf("expected first-message metadata, got %#v", enriched.Comments[1].Message)
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
