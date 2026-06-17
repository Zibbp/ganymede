package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConvertTwitchLiveChatToTDLChatStreamsValidOutput(t *testing.T) {
	t.Parallel()

	chatStartTime := time.Unix(1700000000, 0).UTC()
	liveComments := []LiveComment{
		{
			Message:   "",
			Timestamp: chatStartTime.UnixMicro(),
		},
		{
			Colour:      "#123456",
			Message:     "hello Kappa",
			MessageID:   "message-1",
			MessageType: "highlighted_message",
			Timestamp:   chatStartTime.Add(2 * time.Second).UnixMicro(),
		},
	}
	liveComments[1].Author.ID = "author-1"
	liveComments[1].Author.DisplayName = "Display"
	liveComments[1].Author.Name = "display"
	liveComments[1].Author.IsModerator = true
	liveComments[1].Emotes = append(liveComments[1].Emotes, struct {
		ID     string `json:"id"`
		Images []struct {
			Height int    `json:"height"`
			ID     string `json:"id"`
			URL    string `json:"url"`
			Width  int    `json:"width"`
		} `json:"images"`
		Locations []string `json:"locations"`
		Name      string   `json:"name"`
	}{
		ID:        "25",
		Locations: []string{"6-10"},
		Name:      "Kappa",
	})

	tempDir := t.TempDir()
	inPath := filepath.Join(tempDir, "live-chat.json")
	outPath := filepath.Join(tempDir, "tdl-chat.json")

	input, err := json.Marshal(liveComments)
	if err != nil {
		t.Fatalf("marshal live comments: %v", err)
	}
	if err := os.WriteFile(inPath, input, 0644); err != nil {
		t.Fatalf("write live chat: %v", err)
	}

	err = ConvertTwitchLiveChatToTDLChat(inPath, outPath, "channel", "video-id", "external-id", 123, chatStartTime, "previous-video-id")
	if err != nil {
		t.Fatalf("convert live chat: %v", err)
	}

	output, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read tdl chat: %v", err)
	}

	var tdlChat TDLChat
	if err := json.Unmarshal(output, &tdlChat); err != nil {
		t.Fatalf("unmarshal tdl chat: %v", err)
	}

	if tdlChat.Streamer.Name != "channel" || tdlChat.Streamer.ID != 123 {
		t.Fatalf("unexpected streamer: %+v", tdlChat.Streamer)
	}
	if tdlChat.Video.ID != "previous-video-id" || tdlChat.Video.End != 2 {
		t.Fatalf("unexpected video: %+v", tdlChat.Video)
	}
	if len(tdlChat.Comments) != 2 {
		t.Fatalf("expected initial and converted comments, got %d", len(tdlChat.Comments))
	}

	converted := tdlChat.Comments[1]
	if converted.ID != "message-1" || converted.ContentOffsetSeconds != 2 {
		t.Fatalf("unexpected converted comment: %+v", converted)
	}
	if converted.Message.UserNoticeParams.MsgID == nil || *converted.Message.UserNoticeParams.MsgID != "highlighted-message" {
		t.Fatalf("expected highlighted message notice params: %+v", converted.Message.UserNoticeParams)
	}
	if len(converted.Message.Fragments) != 3 {
		t.Fatalf("expected text/emote/text fragments, got %+v", converted.Message.Fragments)
	}
	if converted.Message.Fragments[1].Emoticon == nil || converted.Message.Fragments[1].Emoticon.EmoticonID != "25" {
		t.Fatalf("expected emote fragment, got %+v", converted.Message.Fragments[1])
	}
}
