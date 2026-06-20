package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type TDLChat struct {
	Streamer Streamer  `json:"streamer"`
	Video    Video     `json:"video"`
	Comments []Comment `json:"comments"`
}

type Streamer struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

type Video struct {
	ID    string `json:"id"`
	Start int64  `json:"start"`
	End   int64  `json:"end"`
}

type Comment struct {
	ID                   string    `json:"_id"`
	Source               string    `json:"source"`
	ContentOffsetSeconds float64   `json:"content_offset_seconds"`
	Commenter            Commenter `json:"commenter"`
	Message              Message   `json:"message"`
}

type Commenter struct {
	DisplayName  string `json:"display_name"`
	ID           string `json:"id"`
	IsModerator  bool   `json:"is_moderator"`
	IsSubscriber bool   `json:"is_subscriber"`
	IsTurbo      bool   `json:"is_turbo"`
	Name         string `json:"name"`
}

type Message struct {
	Body             string          `json:"body"`
	BitsSpent        int             `json:"bits_spent"`
	Fragments        []Fragment      `json:"fragments"`
	IsAction         bool            `json:"is_action"`
	IsFirstMessage   bool            `json:"is_first_message,omitempty"`
	UserBadges       []UserBadge     `json:"user_badges"`
	UserColor        string          `json:"user_color"`
	UserNoticeParams UserNoticParams `json:"user_notice_params"`
	Reply            *ChatReply      `json:"reply,omitempty"`
}

type ChatReply struct {
	ParentMsgID       string `json:"parent_msg_id"`
	ParentUserID      string `json:"parent_user_id"`
	ParentUserLogin   string `json:"parent_user_login"`
	ParentDisplayName string `json:"parent_display_name"`
	ParentMsgBody     string `json:"parent_msg_body"`
}

type Fragment struct {
	Text     string    `json:"text"`
	Emoticon *Emoticon `json:"emoticon"`
	Pos1     int       `json:"pos1"`
	Pos2     int       `json:"pos2"`
}

type UserBadge struct {
	ID      string `json:"_id"`
	Version string `json:"version"`
}

type UserNoticParams struct {
	MsgID     *string           `json:"msg_id,omitempty"`
	SystemMsg string            `json:"system_msg,omitempty"`
	Params    map[string]string `json:"params,omitempty"`
}

type Emoticon struct {
	EmoticonID    string `json:"emoticon_id"`
	EmoticonSetID string `json:"emoticon_set_id"`
}

type LiveChat struct {
	Comments []LiveComment `json:"comments"`
}

func ConvertTwitchLiveChatToTDLChat(path string, outPath string, channelName string, videoID string, videoExternalID string, channelID int, chatStartTime time.Time, previousVideoID string) error {

	log.Debug().Str("chat_file", path).Msg("Converting live Twitch chat to TDL chat for rendering")

	liveComments, err := OpenLiveChatFile(path)
	if err != nil {
		return err
	}

	tdlChat := TDLChat{}

	tdlChat.Streamer.Name = channelName
	tdlChat.Streamer.ID = channelID
	tdlChat.Video.ID = previousVideoID // we don't know the video (vod) id at this point
	tdlChat.Video.Start = 0

	tdlComments := []Comment{}

	// create an initial comment to mark the start of chat
	tdlComments = append(tdlComments, Comment{
		ID:                   "546a5e6e-c820-4ad2-9421-9ba5b5bf37ea",
		Source:               "chat",
		ContentOffsetSeconds: 0,
		Commenter: Commenter{
			DisplayName:  "Ganymede",
			ID:           "222777213",
			IsModerator:  false,
			IsSubscriber: false,
			IsTurbo:      false,
			Name:         "ganymede",
		},
		Message: Message{
			Body:      "Initial chat message",
			BitsSpent: 0,
			Fragments: []Fragment{
				{
					Text:     "Initial chat message",
					Emoticon: nil,
					Pos1:     0,
					Pos2:     0,
				},
			},
			UserBadges: []UserBadge{},
			UserColor:  "#a65ee8",
			UserNoticeParams: UserNoticParams{
				MsgID: nil,
			},
		},
	})

	for _, liveComment := range liveComments {
		if liveComment.Message == "" {
			continue
		}

		// get offset in seconds
		liveCommentUnix, err := microSecondToMillisecondUnix(liveComment.Timestamp)
		if err != nil {
			return fmt.Errorf("failed to convert live comment timestamp: %v", err)
		}

		// use chat start time to get offset in seconds
		diff := liveCommentUnix.Sub(chatStartTime)

		// populate static variables
		tdlComment := Comment{
			ContentOffsetSeconds: diff.Seconds(),
			ID:                   liveComment.MessageID,
			Source:               "chat",
			Commenter: Commenter{
				ID:           liveComment.Author.ID,
				DisplayName:  liveComment.Author.DisplayName,
				Name:         liveComment.Author.Name,
				IsModerator:  liveComment.Author.IsModerator,
				IsSubscriber: liveComment.Author.IsSubscriber,
				IsTurbo:      liveComment.Author.IsTurbo,
			},
			Message: Message{
				Body:           liveComment.Message,
				BitsSpent:      liveComment.BitsSpent,
				IsAction:       liveComment.IsAction,
				IsFirstMessage: liveComment.IsFirstMessage,
				UserBadges:     []UserBadge{},
				UserColor:      liveComment.Colour,
				UserNoticeParams: UserNoticParams{
					MsgID: nil,
				},
			},
		}

		if liveComment.Reply != nil {
			tdlComment.Message.Reply = liveCommentReplyToChatReply(liveComment.Reply)
		}

		if liveComment.MessageType == "highlighted_message" {
			var highlightString = "highlighted-message"
			tdlComment.Message.UserNoticeParams.MsgID = &highlightString
		}
		if msgID, ok := liveComment.UserNoticeParams["msg-id"]; ok && msgID != "" {
			tdlComment.Message.UserNoticeParams.MsgID = &msgID
		}
		if systemMsg, ok := liveComment.UserNoticeParams["system-msg"]; ok {
			tdlComment.Message.UserNoticeParams.SystemMsg = systemMsg
		}
		if len(liveComment.UserNoticeParams) > 0 {
			tdlComment.Message.UserNoticeParams.Params = liveUserNoticeParams(liveComment.UserNoticeParams)
		}

		// create the first message fragment
		tdlComment.Message.Fragments = append(tdlComment.Message.Fragments, Fragment{
			Text:     liveComment.Message,
			Emoticon: nil,
		})

		// set default offset value for this live comment
		message_is_offset := false

		// parse emotes, creating fragments with positions
		emoteFragments := []Fragment{}
		if liveComment.Emotes != nil {
			for _, liveCommentEmote := range liveComment.Emotes {
				for i, liveCommentEmoteLocation := range liveCommentEmote.Locations {
					var pos1, pos2 int
					var emoteFragment Fragment
					// get position of emote in message
					emotePositions := strings.Split(liveCommentEmoteLocation, "-")
					pos1, err := strconv.Atoi(emotePositions[0])
					if err != nil {
						return fmt.Errorf("failed to convert emote position: %v", err)
					}
					chatPos2, err := strconv.Atoi(emotePositions[1])
					if err != nil {
						return fmt.Errorf("failed to convert emote position: %v", err)
					}
					pos2 = pos1 + len(liveCommentEmote.Name)

					var slicedEmote string

					// Check if pos1 and pos2 are within bounds
					if pos1 < 0 || pos2 > len(liveComment.Message) {
						// Check if pos1 is negative and pos2 is within bounds
						if pos1 < 0 || chatPos2 > len(liveComment.Message) {
							log.Error().Str("message_id", liveComment.MessageID).Msg("emote position out of bounds, skipping emote")
							continue
						}
						// If default chat pos2 is in bounds use it
						log.Warn().Str("message_id", liveComment.MessageID).Msg("emote position out of bounds, using default chat pos2 instead")
						slicedEmote = liveComment.Message[pos1:chatPos2]
					} else {
						slicedEmote = liveComment.Message[pos1:pos2]
					}

					// ensure that the sliced string equals the emote
					// sometimes the output of chat-downloader will not include a unicode character when calculating positions causing an offset in positions
					if slicedEmote != liveCommentEmote.Name || message_is_offset {
						log.Debug().Str("message_id", liveComment.MessageID).Msg("emote position mismatch detected while converting chat")
						message_is_offset = true

						// attempt to get emote position in comment message
						pos1, pos2, found := findSubstringPositions(liveComment.Message, liveCommentEmote.Name, i+1)
						if !found {
							log.Warn().Str("message_id", liveComment.MessageID).Msg("unable to extract emote positions from message, skpping")
							continue
						}
						slicedEmote = liveComment.Message[pos1:pos2]
						emoteFragment = Fragment{
							Pos1: pos1,
							Pos2: pos2,
							Text: slicedEmote,
							Emoticon: &Emoticon{
								EmoticonID:    liveCommentEmote.ID,
								EmoticonSetID: "",
							},
						}
					} else {
						emoteFragment = Fragment{
							Pos1: pos1,
							Pos2: pos2,
							Text: slicedEmote,
							Emoticon: &Emoticon{
								EmoticonID:    liveCommentEmote.ID,
								EmoticonSetID: "",
							},
						}
					}

					emoteFragments = append(emoteFragments, emoteFragment)
				}
			}
		}

		// sort emote fragments by position ascending
		sort.Slice(emoteFragments, func(i, j int) bool {
			return emoteFragments[i].Pos1 < emoteFragments[j].Pos1
		})

		formattedEmoteFragments := []Fragment{}

		// remove emote fragments from message fragments
		for i, emoteFragment := range emoteFragments {
			if i == 0 {
				fragmentText := tdlComment.Message.Body[:emoteFragment.Pos1]
				fragment := Fragment{
					Text:     fragmentText,
					Emoticon: nil,
				}
				formattedEmoteFragments = append(formattedEmoteFragments, fragment)
				formattedEmoteFragments = append(formattedEmoteFragments, emoteFragment)
			} else {
				if emoteFragment.Pos1 == 0 {
					log.Warn().Str("message_id", liveComment.MessageID).Msg("skipping invalid emote position")
					continue
				}
				fragmentText := tdlComment.Message.Body[emoteFragments[i-1].Pos2:emoteFragment.Pos1]
				fragment := Fragment{
					Text:     fragmentText,
					Emoticon: nil,
				}
				formattedEmoteFragments = append(formattedEmoteFragments, fragment)
				formattedEmoteFragments = append(formattedEmoteFragments, emoteFragment)
			}
		}

		// check if last fragment is an emoticon
		if len(formattedEmoteFragments) > 0 {
			lastItem := len(formattedEmoteFragments) - 1
			if formattedEmoteFragments[lastItem].Emoticon.EmoticonID != "" {
				fragmentText := tdlComment.Message.Body[formattedEmoteFragments[lastItem].Pos2:]
				fragment := Fragment{
					Text:     fragmentText,
					Emoticon: nil,
				}
				formattedEmoteFragments = append(formattedEmoteFragments, fragment)
			}
		}

		// ensure message has emote fragments
		if len(formattedEmoteFragments) > 0 {
			tdlComment.Message.Fragments = formattedEmoteFragments
		}

		// user badges
		if len(liveComment.Author.Badges) > 0 {
			for _, liveCommentBadge := range liveComment.Author.Badges {
				liveCommentUserBadge := UserBadge{
					ID:      liveCommentBadge.Name,
					Version: fmt.Sprintf("%v", liveCommentBadge.Version),
				}
				tdlComment.Message.UserBadges = append(tdlComment.Message.UserBadges, liveCommentUserBadge)
			}
		}

		// ensure user has a display name color
		if tdlComment.Message.UserColor == "" {
			tdlComment.Message.UserColor = "#a65ee8"
		}

		tdlComments = append(tdlComments, tdlComment)
	}

	tdlChat.Comments = tdlComments

	// get last comment offset and set as video end
	lastComment := tdlChat.Comments[len(tdlChat.Comments)-1]
	tdlChat.Video.End = int64(lastComment.ContentOffsetSeconds)

	// write chat
	err = writeTDLChat(tdlChat, outPath)
	if err != nil {
		return err
	}

	return nil

}

func writeTDLChat(parsedChat TDLChat, outPath string) error {
	data, err := json.Marshal(parsedChat)
	if err != nil {
		return fmt.Errorf("failed to marshal parsed comments: %v", err)
	}
	err = os.WriteFile(outPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write parsed comments: %v", err)
	}
	return nil
}

type liveChatMetadata struct {
	BitsSpent        int
	IsAction         bool
	IsFirstMessage   bool
	Reply            *LiveCommentReply
	UserNoticeParams map[string]string
}

type finalChatReply struct {
	ParentMsgID       string `json:"parent_msg_id"`
	ParentUserID      string `json:"parent_user_id"`
	ParentUserLogin   string `json:"parent_user_login"`
	ParentDisplayName string `json:"parent_display_name"`
	ParentMsgBody     string `json:"parent_msg_body"`
}

type finalChatUserNoticeParams struct {
	MsgID     string            `json:"msg_id,omitempty"`
	SystemMsg string            `json:"system_msg,omitempty"`
	Params    map[string]string `json:"params,omitempty"`
}

func EnrichTwitchChatMetadataFromLiveChat(liveChatPath string, chatPath string) error {
	liveComments, err := OpenLiveChatFile(liveChatPath)
	if err != nil {
		return err
	}

	metadataByID := make(map[string]liveChatMetadata)
	for _, liveComment := range liveComments {
		if liveComment.MessageID == "" {
			continue
		}

		userNoticeParams := liveComment.UserNoticeParams
		if liveComment.MessageType == "highlighted_message" && len(userNoticeParams) == 0 {
			userNoticeParams = map[string]string{"msg-id": "highlighted-message"}
		}

		metadata := liveChatMetadata{
			BitsSpent:        liveComment.BitsSpent,
			IsAction:         liveComment.IsAction,
			IsFirstMessage:   liveComment.IsFirstMessage,
			Reply:            liveComment.Reply,
			UserNoticeParams: userNoticeParams,
		}

		if metadata.BitsSpent == 0 && !metadata.IsAction && !metadata.IsFirstMessage && metadata.Reply == nil && len(metadata.UserNoticeParams) == 0 {
			continue
		}

		metadataByID[liveComment.MessageID] = metadata
	}

	if len(metadataByID) == 0 {
		return nil
	}

	data, err := os.ReadFile(chatPath)
	if err != nil {
		return fmt.Errorf("failed to read chat file for metadata enrichment: %w", err)
	}

	var chatData map[string]interface{}
	if err := json.Unmarshal(data, &chatData); err != nil {
		return fmt.Errorf("failed to unmarshal chat file for metadata enrichment: %w", err)
	}

	rawComments, ok := chatData["comments"].([]interface{})
	if !ok {
		return fmt.Errorf("failed to enrich chat metadata: comments field missing or invalid")
	}

	enrichedCount := 0
	for _, rawComment := range rawComments {
		comment, ok := rawComment.(map[string]interface{})
		if !ok {
			continue
		}

		id, ok := comment["_id"].(string)
		if !ok || id == "" {
			continue
		}

		metadata, ok := metadataByID[id]
		if !ok {
			continue
		}

		message, ok := comment["message"].(map[string]interface{})
		if !ok {
			continue
		}

		if metadata.BitsSpent > 0 {
			message["bits_spent"] = metadata.BitsSpent
		}
		if metadata.IsAction {
			message["is_action"] = true
		}
		if metadata.IsFirstMessage {
			message["is_first_message"] = true
		}
		if metadata.Reply != nil {
			message["reply"] = finalChatReply{
				ParentMsgID:       metadata.Reply.ParentMsgID,
				ParentUserID:      metadata.Reply.ParentUserID,
				ParentUserLogin:   metadata.Reply.ParentUserLogin,
				ParentDisplayName: metadata.Reply.ParentDisplayName,
				ParentMsgBody:     metadata.Reply.ParentMsgBody,
			}
		}
		if len(metadata.UserNoticeParams) > 0 {
			message["user_notice_params"] = finalUserNoticeParams(metadata.UserNoticeParams)
		}

		enrichedCount++
	}

	output, err := json.Marshal(chatData)
	if err != nil {
		return fmt.Errorf("failed to marshal enriched chat metadata: %w", err)
	}
	if err := os.WriteFile(chatPath, output, 0o644); err != nil {
		return fmt.Errorf("failed to write enriched chat metadata: %w", err)
	}

	log.Debug().
		Str("live_chat_file", liveChatPath).
		Str("chat_file", chatPath).
		Int("enriched_comments", enrichedCount).
		Msg("enriched Twitch chat metadata")

	return nil
}

func liveCommentReplyToChatReply(reply *LiveCommentReply) *ChatReply {
	return &ChatReply{
		ParentMsgID:       reply.ParentMsgID,
		ParentUserID:      reply.ParentUserID,
		ParentUserLogin:   reply.ParentUserLogin,
		ParentDisplayName: reply.ParentDisplayName,
		ParentMsgBody:     reply.ParentMsgBody,
	}
}

func liveUserNoticeParams(params map[string]string) map[string]string {
	noticeParams := make(map[string]string, len(params))
	for key, value := range params {
		if key == "msg-id" || key == "system-msg" {
			continue
		}
		noticeParams[key] = value
	}
	if len(noticeParams) == 0 {
		return nil
	}
	return noticeParams
}

func finalUserNoticeParams(params map[string]string) finalChatUserNoticeParams {
	return finalChatUserNoticeParams{
		MsgID:     params["msg-id"],
		SystemMsg: params["system-msg"],
		Params:    liveUserNoticeParams(params),
	}
}

func microSecondToMillisecondUnix(t int64) (time.Time, error) {
	sT := strconv.FormatInt(t, 10)
	fST := sT[:len(sT)-3]
	iFST, err := strconv.ParseInt(fST, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	unixTimeUTC := time.Unix(iFST/int64(1000), (iFST%int64(1000))*int64(1000000))
	return unixTimeUTC, nil
}
