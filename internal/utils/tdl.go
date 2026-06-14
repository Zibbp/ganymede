package utils

import (
	"bufio"
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
	UserBadges       []UserBadge     `json:"user_badges"`
	UserColor        string          `json:"user_color"`
	UserNoticeParams UserNoticParams `json:"user_notice_params"`
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
	MsgID *string `json:"msg-id"`
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

	liveChatJsonFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open chat file: %v", err)
	}
	defer func() {
		if err := liveChatJsonFile.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing chat file")
		}
	}()

	tdlChat := TDLChat{}

	tdlChat.Streamer.Name = channelName
	tdlChat.Streamer.ID = channelID
	tdlChat.Video.ID = previousVideoID // we don't know the video (vod) id at this point
	tdlChat.Video.Start = 0

	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create parsed comments file: %v", err)
	}
	defer func() {
		if err := outFile.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing parsed comments file")
		}
	}()

	writer := bufio.NewWriter(outFile)
	defer func() {
		if err := writer.Flush(); err != nil {
			log.Debug().Err(err).Msg("error flushing parsed comments file")
		}
	}()

	if _, err := fmt.Fprint(writer, `{"streamer":`); err != nil {
		return fmt.Errorf("failed to write parsed comments: %v", err)
	}
	if err := json.NewEncoder(writer).Encode(tdlChat.Streamer); err != nil {
		return fmt.Errorf("failed to encode streamer: %v", err)
	}
	if _, err := fmt.Fprint(writer, `,"comments":[`); err != nil {
		return fmt.Errorf("failed to write parsed comments: %v", err)
	}

	// create an initial comment to mark the start of chat
	initialComment := Comment{
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

	encoder := json.NewEncoder(writer)
	if err := encoder.Encode(initialComment); err != nil {
		return fmt.Errorf("failed to encode initial comment: %v", err)
	}

	decoder := json.NewDecoder(liveChatJsonFile)
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read chat file: %v", err)
	}
	if delim, ok := token.(json.Delim); !ok || delim != '[' {
		return fmt.Errorf("failed to read chat file: expected JSON array")
	}

	lastOffset := initialComment.ContentOffsetSeconds
	for decoder.More() {
		var liveComment LiveComment
		if err := decoder.Decode(&liveComment); err != nil {
			return fmt.Errorf("failed to decode live comment: %v", err)
		}

		tdlComment, ok, err := convertLiveCommentToTDLComment(liveComment, chatStartTime)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}

		if _, err := fmt.Fprint(writer, ","); err != nil {
			return fmt.Errorf("failed to write parsed comments: %v", err)
		}
		if err := encoder.Encode(tdlComment); err != nil {
			return fmt.Errorf("failed to encode comment: %v", err)
		}
		lastOffset = tdlComment.ContentOffsetSeconds
	}

	token, err = decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read chat file: %v", err)
	}
	if delim, ok := token.(json.Delim); !ok || delim != ']' {
		return fmt.Errorf("failed to read chat file: expected end of JSON array")
	}

	tdlChat.Video.End = int64(lastOffset)
	if _, err := fmt.Fprint(writer, `],"video":`); err != nil {
		return fmt.Errorf("failed to write parsed comments: %v", err)
	}
	if err := encoder.Encode(tdlChat.Video); err != nil {
		return fmt.Errorf("failed to encode video: %v", err)
	}
	if _, err := fmt.Fprint(writer, "}"); err != nil {
		return fmt.Errorf("failed to write parsed comments: %v", err)
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to write parsed comments: %v", err)
	}

	return nil

}

func convertLiveCommentToTDLComment(liveComment LiveComment, chatStartTime time.Time) (Comment, bool, error) {
	if liveComment.Message == "" {
		return Comment{}, false, nil
	}

	// get offset in seconds
	liveCommentUnix, err := microSecondToMillisecondUnix(liveComment.Timestamp)
	if err != nil {
		return Comment{}, false, fmt.Errorf("failed to convert live comment timestamp: %v", err)
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
			Body:       liveComment.Message,
			BitsSpent:  0,
			UserBadges: []UserBadge{},
			UserColor:  liveComment.Colour,
			UserNoticeParams: UserNoticParams{
				MsgID: nil,
			},
		},
	}

	if liveComment.MessageType == "highlighted_message" {
		var highlightString = "highlighted-message"
		tdlComment.Message.UserNoticeParams.MsgID = &highlightString
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
				if len(emotePositions) != 2 {
					log.Warn().Str("message_id", liveComment.MessageID).Msg("invalid emote position, skipping emote")
					continue
				}
				pos1, err := strconv.Atoi(emotePositions[0])
				if err != nil {
					return Comment{}, false, fmt.Errorf("failed to convert emote position: %v", err)
				}
				chatPos2, err := strconv.Atoi(emotePositions[1])
				if err != nil {
					return Comment{}, false, fmt.Errorf("failed to convert emote position: %v", err)
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
		if formattedEmoteFragments[lastItem].Emoticon != nil && formattedEmoteFragments[lastItem].Emoticon.EmoticonID != "" {
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

	return tdlComment, true, nil
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
