package exec

import (
	"fmt"
	"testing"
	"time"

	twitchIRC "github.com/gempir/go-twitch-irc/v4"
)

func TestConvertToLiveCommentIncludesReply(t *testing.T) {
	message := twitchIRC.PrivateMessage{
		User: twitchIRC.User{
			ID:          "269899575",
			Name:        "fletchercodes",
			DisplayName: "FletcherCodes",
			Color:       "#DAA520",
			Badges:      map[string]int{"subscriber": 12},
		},
		Tags: map[string]string{
			"flags":             "",
			"returning-chatter": "0",
			"user-type":         "",
			"subscriber":        "1",
		},
		RoomID:         "408892348",
		ID:             "6efffc70-27a1-4637-9111-44e5104bb7da",
		Time:           time.Unix(1_551_473_087, 761_000_000),
		Message:        "Chew your food slower... it's healthier",
		Bits:           100,
		Action:         true,
		FirstMessage:   true,
		CustomRewardID: "reward-id",
		Reply: &twitchIRC.Reply{
			ParentMsgID:       "b34ccfc7-4977-403a-8a94-33c6bac34fb8",
			ParentUserID:      "71601484",
			ParentUserLogin:   "yannismate",
			ParentDisplayName: "Yannismate",
			ParentMsgBody:     "This message contains special chars !:",
		},
	}

	comment := convertToLiveComment(message)

	if comment.Reply == nil {
		t.Fatal("expected reply metadata")
	}
	if comment.Reply.ParentMsgID != message.Reply.ParentMsgID {
		t.Fatalf("expected parent message ID %q, got %q", message.Reply.ParentMsgID, comment.Reply.ParentMsgID)
	}
	if comment.Reply.ParentMsgBody != message.Reply.ParentMsgBody {
		t.Fatalf("expected parent body %q, got %q", message.Reply.ParentMsgBody, comment.Reply.ParentMsgBody)
	}
	if comment.BitsSpent != 100 {
		t.Fatalf("expected bits spent to be copied, got %d", comment.BitsSpent)
	}
	if !comment.IsAction {
		t.Fatal("expected action flag to be copied")
	}
	if !comment.IsFirstMessage {
		t.Fatal("expected first message flag to be copied")
	}
	if comment.CustomRewardID != "reward-id" {
		t.Fatalf("expected custom reward ID, got %q", comment.CustomRewardID)
	}
}

func TestConvertUserNoticeToLiveCommentResubWithUserMessage(t *testing.T) {
	raw := `@badges=moderator/1,subscriber/24;color=#1FD2FF;display-name=Karl_Kons;emotes=28087:0-6;flags=;id=7c95beea-a7ac-4c10-9e0a-d7dbf163c038;login=karl_kons;mod=1;msg-id=resub;msg-param-months=34;msg-param-sub-plan-name=look\sat\sthose\sshitty\semotes,\srip\s$5\sLUL;msg-param-sub-plan=1000;room-id=11148817;subscriber=1;system-msg=Karl_Kons\sjust\ssubscribed\swith\sa\sTier\s1\ssub.\sKarl_Kons\ssubscribed\sfor\s34\smonths\sin\sa\srow!;tmi-sent-ts=1540140252828;turbo=0;user-id=68706331;user-type=mod :tmi.twitch.tv USERNOTICE #pajlada :WutFace`
	message, ok := twitchIRC.ParseMessage(raw).(*twitchIRC.UserNoticeMessage)
	if !ok {
		t.Fatal("expected user notice message")
	}

	comment := convertUserNoticeToLiveComment(*message)

	expectedBody := "Karl_Kons just subscribed with a Tier 1 sub. Karl_Kons subscribed for 34 months in a row! WutFace"
	if comment.Message != expectedBody {
		t.Fatalf("expected body %q, got %q", expectedBody, comment.Message)
	}
	if comment.MessageType != "user_notice" {
		t.Fatalf("expected user_notice message type, got %q", comment.MessageType)
	}
	if comment.UserNoticeParams["msg-id"] != "resub" {
		t.Fatalf("expected msg-id resub, got %q", comment.UserNoticeParams["msg-id"])
	}
	if comment.UserNoticeParams["msg-param-months"] != "34" {
		t.Fatalf("expected msg-param-months 34, got %q", comment.UserNoticeParams["msg-param-months"])
	}
	if comment.UserNoticeParams["user-message"] != "WutFace" {
		t.Fatalf("expected user-message WutFace, got %q", comment.UserNoticeParams["user-message"])
	}
}

func TestConvertUserNoticeToLiveCommentRaidWithoutUserMessage(t *testing.T) {
	raw := `@badges=partner/1;color=#00FF7F;display-name=FletcherCodes;emotes=;flags=;id=7a61cd41-f049-466b-9654-43e5bfc554aa;login=fletchercodes;mod=0;msg-id=raid;msg-param-displayName=FletcherCodes;msg-param-login=fletchercodes;msg-param-profileImageURL=https://static-cdn.jtvnw.net/jtv_user_pictures/herr_currywurst-profile_image-e6c037c9d321b955-70x70.jpeg;msg-param-viewerCount=538;room-id=269899575;subscriber=0;system-msg=538\sraiders\sfrom\sFletcherCodes\shave\sjoined\n!;tmi-sent-ts=1551490358542;turbo=0;user-id=269899575;user-type= :tmi.twitch.tv USERNOTICE #clippyassistant`
	message, ok := twitchIRC.ParseMessage(raw).(*twitchIRC.UserNoticeMessage)
	if !ok {
		t.Fatal("expected user notice message")
	}

	comment := convertUserNoticeToLiveComment(*message)

	if comment.Message != "538 raiders from FletcherCodes have joined!" {
		t.Fatalf("unexpected raid message body: %q", comment.Message)
	}
	if comment.UserNoticeParams["msg-id"] != "raid" {
		t.Fatalf("expected msg-id raid, got %q", comment.UserNoticeParams["msg-id"])
	}
	if comment.UserNoticeParams["msg-param-viewerCount"] != "538" {
		t.Fatalf("expected viewer count 538, got %q", comment.UserNoticeParams["msg-param-viewerCount"])
	}
}

func TestConvertUserNoticeToLiveCommentShiftsUserMessageEmotes(t *testing.T) {
	raw := `@badges=;color=;display-name=FletcherCodes;emotes=64138:0-8;flags=;id=e4090aa9-8079-41ff-904d-64c7a2193ee0;login=fletchercodes;mod=0;msg-id=ritual;msg-param-ritual-name=new_chatter;room-id=408892348;subscriber=0;system-msg=@FletcherCodes\sis\snew\shere.\sSay\shello!;tmi-sent-ts=1551487438943;turbo=0;user-id=412636239;user-type= :tmi.twitch.tv USERNOTICE #clippyassistant :SeemsGood`
	message, ok := twitchIRC.ParseMessage(raw).(*twitchIRC.UserNoticeMessage)
	if !ok {
		t.Fatal("expected user notice message")
	}

	comment := convertUserNoticeToLiveComment(*message)

	if len(comment.Emotes) != 1 || len(comment.Emotes[0].Locations) != 1 {
		t.Fatalf("expected one emote location, got %#v", comment.Emotes)
	}

	offset := len(message.SystemMsg) + 1
	expectedLocation := fmt.Sprintf("%d-%d", offset, offset+len(message.Message)-1)
	if comment.Emotes[0].Locations[0] != expectedLocation {
		t.Fatalf("expected shifted emote location %q, got %q", expectedLocation, comment.Emotes[0].Locations[0])
	}
}
