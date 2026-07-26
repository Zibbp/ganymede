package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

	liveChatJSONFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open chat file: %w", err)
	}
	defer func() {
		if err := liveChatJSONFile.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing chat file")
		}
	}()

	outputMode := os.FileMode(0o644)
	if info, err := os.Stat(outPath); err == nil {
		outputMode = info.Mode()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat output chat file: %w", err)
	}

	outputDir := filepath.Dir(outPath)
	tempFile, err := os.CreateTemp(outputDir, filepath.Base(outPath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary output chat file: %w", err)
	}
	tempPath := tempFile.Name()
	tempFileOpen := true
	defer func() {
		if tempFileOpen {
			if err := tempFile.Close(); err != nil {
				log.Debug().Err(err).Msg("error closing temporary output chat file")
			}
		}
		if err := os.Remove(tempPath); err != nil && !os.IsNotExist(err) {
			log.Debug().Err(err).Msg("error removing temporary output chat file")
		}
	}()

	streamerJSON, err := json.Marshal(Streamer{Name: channelName, ID: channelID})
	if err != nil {
		return fmt.Errorf("failed to marshal streamer metadata: %w", err)
	}
	videoIDJSON, err := json.Marshal(previousVideoID) // we don't know the video (vod) id at this point
	if err != nil {
		return fmt.Errorf("failed to marshal video metadata: %w", err)
	}

	if _, err := fmt.Fprintf(tempFile, `{"streamer":%s,"video":{"id":%s,"start":0,"end":`, streamerJSON, videoIDJSON); err != nil {
		return fmt.Errorf("failed to write TDL chat header: %w", err)
	}

	videoEndOffset, err := tempFile.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("failed to locate TDL video end field: %w", err)
	}
	const videoEndFieldWidth = 20
	if _, err := fmt.Fprintf(tempFile, "%*d", videoEndFieldWidth, 0); err != nil {
		return fmt.Errorf("failed to write TDL video end placeholder: %w", err)
	}
	if _, err := io.WriteString(tempFile, `},"comments":[`); err != nil {
		return fmt.Errorf("failed to write TDL comments header: %w", err)
	}

	outputWriter := bufio.NewWriter(tempFile)
	firstComment := true
	initialComment := initialTDLComment()
	if err := writeTDLComment(outputWriter, initialComment, &firstComment); err != nil {
		return err
	}
	videoEnd := int64(initialComment.ContentOffsetSeconds)

	err = streamLiveComments(liveChatJSONFile, func(liveComment LiveComment) error {
		tdlComment, include, err := convertLiveCommentToTDLComment(liveComment, chatStartTime)
		if err != nil {
			return err
		}
		if !include {
			return nil
		}

		if err := writeTDLComment(outputWriter, tdlComment, &firstComment); err != nil {
			return err
		}
		videoEnd = int64(tdlComment.ContentOffsetSeconds)
		return nil
	})
	if err != nil {
		return err
	}

	if _, err := io.WriteString(outputWriter, `]}`); err != nil {
		return fmt.Errorf("failed to write TDL chat footer: %w", err)
	}
	if err := outputWriter.Flush(); err != nil {
		return fmt.Errorf("failed to flush TDL chat output: %w", err)
	}

	videoEndJSON := strconv.FormatInt(videoEnd, 10)
	if len(videoEndJSON) > videoEndFieldWidth {
		return fmt.Errorf("TDL video end value %q exceeds reserved field width", videoEndJSON)
	}
	if _, err := tempFile.Seek(videoEndOffset, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to TDL video end field: %w", err)
	}
	if _, err := io.WriteString(tempFile, strings.Repeat(" ", videoEndFieldWidth-len(videoEndJSON))+videoEndJSON); err != nil {
		return fmt.Errorf("failed to write TDL video end field: %w", err)
	}

	if err := tempFile.Chmod(outputMode); err != nil {
		return fmt.Errorf("failed to set output chat file permissions: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync output chat file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close output chat file: %w", err)
	}
	tempFileOpen = false

	if err := os.Rename(tempPath, outPath); err != nil {
		return fmt.Errorf("failed to atomically replace output chat file: %w", err)
	}

	return nil
}

func streamLiveComments(reader io.Reader, handle func(LiveComment) error) error {
	decoder := json.NewDecoder(reader)

	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to unmarshal chat file: %w", err)
	}
	openingDelimiter, ok := token.(json.Delim)
	if !ok || openingDelimiter != '[' {
		return fmt.Errorf("failed to unmarshal chat file: expected a JSON array")
	}

	for decoder.More() {
		var liveComment LiveComment
		if err := decoder.Decode(&liveComment); err != nil {
			return fmt.Errorf("failed to unmarshal chat file: %w", err)
		}
		if err := handle(liveComment); err != nil {
			return err
		}
	}

	token, err = decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to unmarshal chat file: %w", err)
	}
	closingDelimiter, ok := token.(json.Delim)
	if !ok || closingDelimiter != ']' {
		return fmt.Errorf("failed to unmarshal chat file: expected the end of a JSON array")
	}

	if token, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("failed to unmarshal chat file: unexpected trailing JSON token %v", token)
		}
		return fmt.Errorf("failed to unmarshal chat file: %w", err)
	}

	return nil
}

func initialTDLComment() Comment {
	return Comment{
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
	}
}

func convertLiveCommentToTDLComment(liveComment LiveComment, chatStartTime time.Time) (Comment, bool, error) {
	if liveComment.Message == "" {
		return Comment{}, false, nil
	}

	liveCommentUnix, err := microSecondToMillisecondUnix(liveComment.Timestamp)
	if err != nil {
		return Comment{}, false, fmt.Errorf("failed to convert live comment timestamp: %v", err)
	}
	diff := liveCommentUnix.Sub(chatStartTime)

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
		highlightString := "highlighted-message"
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

	tdlComment.Message.Fragments = append(tdlComment.Message.Fragments, Fragment{
		Text:     liveComment.Message,
		Emoticon: nil,
	})

	messageIsOffset := false
	emoteFragments := []Fragment{}
	if liveComment.Emotes != nil {
		for _, liveCommentEmote := range liveComment.Emotes {
			for i, liveCommentEmoteLocation := range liveCommentEmote.Locations {
				var pos1, pos2 int
				var emoteFragment Fragment
				emotePositions := strings.Split(liveCommentEmoteLocation, "-")
				pos1, err = strconv.Atoi(emotePositions[0])
				if err != nil {
					return Comment{}, false, fmt.Errorf("failed to convert emote position: %v", err)
				}
				chatPos2, err := strconv.Atoi(emotePositions[1])
				if err != nil {
					return Comment{}, false, fmt.Errorf("failed to convert emote position: %v", err)
				}
				pos2 = pos1 + len(liveCommentEmote.Name)

				var slicedEmote string
				if pos1 < 0 || pos2 > len(liveComment.Message) {
					if pos1 < 0 || chatPos2 > len(liveComment.Message) {
						log.Error().Str("message_id", liveComment.MessageID).Msg("emote position out of bounds, skipping emote")
						continue
					}
					log.Warn().Str("message_id", liveComment.MessageID).Msg("emote position out of bounds, using default chat pos2 instead")
					slicedEmote = liveComment.Message[pos1:chatPos2]
				} else {
					slicedEmote = liveComment.Message[pos1:pos2]
				}

				if slicedEmote != liveCommentEmote.Name || messageIsOffset {
					log.Debug().Str("message_id", liveComment.MessageID).Msg("emote position mismatch detected while converting chat")
					messageIsOffset = true

					var found bool
					pos1, pos2, found = findSubstringPositions(liveComment.Message, liveCommentEmote.Name, i+1)
					if !found {
						log.Warn().Str("message_id", liveComment.MessageID).Msg("unable to extract emote positions from message, skpping")
						continue
					}
					slicedEmote = liveComment.Message[pos1:pos2]
				}

				emoteFragment = Fragment{
					Pos1: pos1,
					Pos2: pos2,
					Text: slicedEmote,
					Emoticon: &Emoticon{
						EmoticonID:    liveCommentEmote.ID,
						EmoticonSetID: "",
					},
				}
				emoteFragments = append(emoteFragments, emoteFragment)
			}
		}
	}

	sort.Slice(emoteFragments, func(i, j int) bool {
		return emoteFragments[i].Pos1 < emoteFragments[j].Pos1
	})

	formattedEmoteFragments := []Fragment{}
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

	if len(formattedEmoteFragments) > 0 {
		tdlComment.Message.Fragments = formattedEmoteFragments
	}

	if len(liveComment.Author.Badges) > 0 {
		for _, liveCommentBadge := range liveComment.Author.Badges {
			liveCommentUserBadge := UserBadge{
				ID:      liveCommentBadge.Name,
				Version: fmt.Sprintf("%v", liveCommentBadge.Version),
			}
			tdlComment.Message.UserBadges = append(tdlComment.Message.UserBadges, liveCommentUserBadge)
		}
	}

	if tdlComment.Message.UserColor == "" {
		tdlComment.Message.UserColor = "#a65ee8"
	}

	return tdlComment, true, nil
}

func writeTDLComment(writer io.Writer, comment Comment, first *bool) error {
	if !*first {
		if _, err := io.WriteString(writer, ","); err != nil {
			return fmt.Errorf("failed to write parsed comment separator: %w", err)
		}
	}

	data, err := json.Marshal(comment)
	if err != nil {
		return fmt.Errorf("failed to marshal parsed comment: %w", err)
	}
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("failed to write parsed comment: %w", err)
	}
	*first = false
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
