package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type ParsedChat struct {
	Streamer Streamer  `json:"streamer"`
	Comments []Comment `json:"comments"`
}

type Streamer struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
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

func OpenChatFile(path string) ([]LiveComment, error) {

	liveChatJsonFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open chat file: %v", err)
	}
	defer liveChatJsonFile.Close()
	byteValue, _ := io.ReadAll(liveChatJsonFile)

	var liveComments []LiveComment
	err = json.Unmarshal(byteValue, &liveComments)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal chat file: %v", err)
	}
	return liveComments, nil
}

func ConvertTwitchLiveChatToVodChat(path string, channelName string, vID string, vExtID string, cID int, chatStart time.Time) error {

	log.Debug().Msg("Converting Twitch Live Chat to Vod Chat")

	liveComments, err := OpenChatFile(path)
	if err != nil {
		return err
	}

	// BEGIN CONVERSION LIVE -> PARSED
	var parsedChat ParsedChat
	parsedChat.Streamer.Name = channelName
	parsedChat.Streamer.ID = cID

	var parsedComments []Comment

	// Create an initial comment to mark the start of the chat session
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
	parsedComments = append(parsedComments, initialComment)

	for _, liveComment := range liveComments {
		// Check if comment is empty
		if liveComment.Message == "" {
			continue
		}

		var parsedComment Comment

		// Get offset in seconds
		liveCommentUnix, err := microSecondToMillisecondUnix(liveComment.Timestamp)
		if err != nil {
			return fmt.Errorf("failed to convert live comment timestamp: %v", err)
		}
		// Use chat start time to get offset in seconds
		diff := liveCommentUnix.Sub(chatStart)
		parsedComment.ContentOffsetSeconds = diff.Seconds()

		parsedComment.ID = liveComment.MessageID
		parsedComment.Source = "chat"
		parsedComment.Commenter.DisplayName = liveComment.Author.DisplayName
		parsedComment.Commenter.ID = liveComment.Author.ID
		parsedComment.Commenter.IsModerator = liveComment.Author.IsModerator
		parsedComment.Commenter.IsSubscriber = liveComment.Author.IsSubscriber
		parsedComment.Commenter.IsTurbo = liveComment.Author.IsTurbo
		parsedComment.Commenter.Name = liveComment.Author.Name

		parsedComment.Message.Body = liveComment.Message
		parsedComment.Message.BitsSpent = 0
		firstFragment := Fragment{
			Text:     liveComment.Message,
			Emoticon: nil,
		}
		parsedComment.Message.Fragments = append(parsedComment.Message.Fragments, firstFragment)
		parsedComment.Message.UserBadges = []UserBadge{}
		parsedComment.Message.UserColor = liveComment.Colour
		parsedComment.Message.UserNoticeParams = UserNoticParams{
			MsgID: nil,
		}

		var emoteFragments []Fragment

		// Extract emotes and create fragments with positions
		if liveComment.Emotes != nil {
			for _, liveEmote := range liveComment.Emotes {
				for _, liveEmoteLocation := range liveEmote.Locations {
					var emoteFragment Fragment
					var emoticonFragment Emoticon
					emoteFragment.Emoticon = &emoticonFragment

					// Get position of emote in message
					emotePositions := strings.Split(liveEmoteLocation, "-")

					pos1, err := strconv.Atoi(emotePositions[0])
					if err != nil {
						return fmt.Errorf("failed to convert emote position: %v", err)
					}
					pos2, err := strconv.Atoi(emotePositions[1])
					if err != nil {
						return fmt.Errorf("failed to convert emote position: %v", err)
					}

					emoteFragment.Pos1 = pos1
					emoteFragment.Pos2 = pos2 + 1

					if pos2+1 > len(liveComment.Message) {
						log.Debug().Msgf("Message: %s -- has an out-of-bounds emote position, skipping.", liveComment.Message)
					} else {
						slicedEmote := liveComment.Message[pos1 : pos2+1]
						emoteFragment.Text = slicedEmote
						emoteFragment.Emoticon.EmoticonID = liveEmote.ID
						emoteFragment.Emoticon.EmoticonSetID = ""

						emoteFragments = append(emoteFragments, emoteFragment)

					}

				}
			}
		}

		// Sort emoteFragments by position ascending
		sort.Slice(emoteFragments, func(i, j int) bool {
			return emoteFragments[i].Pos1 < emoteFragments[j].Pos1
		})

		var formattedEmoteFragments []Fragment

		// Remove emote fragments from message fragments
		for i, emoteFragment := range emoteFragments {
			if i == 0 {
				fragmentText := parsedComment.Message.Body[:emoteFragment.Pos1]
				fragment := Fragment{
					Text:     fragmentText,
					Emoticon: nil,
				}
				formattedEmoteFragments = append(formattedEmoteFragments, fragment)
				formattedEmoteFragments = append(formattedEmoteFragments, emoteFragment)
			} else {
				fragmentText := parsedComment.Message.Body[emoteFragments[i-1].Pos2:emoteFragment.Pos1]
				fragment := Fragment{
					Text:     fragmentText,
					Emoticon: nil,
				}
				formattedEmoteFragments = append(formattedEmoteFragments, fragment)
				formattedEmoteFragments = append(formattedEmoteFragments, emoteFragment)
			}
		}

		// Check if last fragment is an emoticon
		if len(formattedEmoteFragments) > 0 {
			lastItem := len(formattedEmoteFragments) - 1
			if formattedEmoteFragments[lastItem].Emoticon.EmoticonID != "" {
				fragmentText := parsedComment.Message.Body[formattedEmoteFragments[lastItem].Pos2:]
				fragment := Fragment{
					Text:     fragmentText,
					Emoticon: nil,
				}
				formattedEmoteFragments = append(formattedEmoteFragments, fragment)
			}
		}

		// If message has emote fragments
		if len(formattedEmoteFragments) > 0 {
			parsedComment.Message.Fragments = formattedEmoteFragments

		}

		// User badges
		if (liveComment.Author.Badges != nil) && (len(liveComment.Author.Badges) > 0) {
			for _, liveBadge := range liveComment.Author.Badges {
				userBadge := UserBadge{
					ID:      liveBadge.Name,
					Version: fmt.Sprintf("%v", liveBadge.Version),
				}
				parsedComment.Message.UserBadges = append(parsedComment.Message.UserBadges, userBadge)
			}
		}

		// Some users don't have a display name color set
		if parsedComment.Message.UserColor == "" {
			parsedComment.Message.UserColor = "#a65ee8"
		}

		// Push it
		parsedComments = append(parsedComments, parsedComment)
	}

	parsedChat.Comments = parsedComments

	err = writeParsedChat(parsedChat, vID, vExtID)
	if err != nil {
		return err
	}
	return nil
}

func writeParsedChat(parsedChat ParsedChat, vID string, vExtID string) error {
	data, err := json.Marshal(parsedChat)
	if err != nil {
		return fmt.Errorf("failed to marshal parsed comments: %v", err)
	}
	err = os.WriteFile(fmt.Sprintf("/tmp/%s_%s-chat-convert.json", vExtID, vID), data, 0644)
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
