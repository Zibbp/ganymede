package chat

import "encoding/json"

func UnmarshalChat(data []byte) (Chat, error) {
	var r Chat
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *Chat) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type Chat struct {
	Streamer Streamer   `json:"streamer"`
	Comments []Comment  `json:"comments"`
	Video    VideoClass `json:"video"`
	Emotes   Emotes     `json:"emotes"`
}

type ChatNoEmotes struct {
	Streamer Streamer   `json:"streamer"`
	Comments []Comment  `json:"comments"`
	Video    VideoClass `json:"video"`
}
type ChatOnlyEmotes struct {
	Streamer Streamer   `json:"streamer"`
	Video    VideoClass `json:"video"`
	Emotes   Emotes     `json:"emotes"`
}

type Comment struct {
	ID                   string      `json:"_id"`
	CreatedAt            string      `json:"created_at"`
	UpdatedAt            string      `json:"updated_at"`
	ChannelID            string      `json:"channel_id"`
	ContentType          ContentType `json:"content_type"`
	ContentID            string      `json:"content_id"`
	ContentOffsetSeconds float64     `json:"content_offset_seconds"`
	Commenter            Commenter   `json:"commenter"`
	Source               Source      `json:"source"`
	State                State       `json:"state"`
	Message              Message     `json:"message"`
	MoreReplies          bool        `json:"more_replies"`
}

type Commenter struct {
	DisplayName string  `json:"display_name"`
	ID          string  `json:"_id"`
	Name        string  `json:"name"`
	Type        Type    `json:"type"`
	Bio         *string `json:"bio"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	Logo        string  `json:"logo"`
}

type Message struct {
	Body             string            `json:"body"`
	BitsSpent        int64             `json:"bits_spent"`
	Fragments        []Fragment        `json:"fragments"`
	IsAction         bool              `json:"is_action"`
	UserBadges       []UserBadge       `json:"user_badges"`
	UserColor        *string           `json:"user_color"`
	UserNoticeParams UserNoticeParams  `json:"user_notice_params"`
	Emoticons        []EmoticonElement `json:"emoticons"`
}

type EmoticonElement struct {
	ID    string `json:"_id"`
	Begin int64  `json:"begin"`
	End   int64  `json:"end"`
}

type Fragment struct {
	Text     string            `json:"text"`
	Emoticon *FragmentEmoticon `json:"emoticon"`
}

type FragmentEmoticon struct {
	EmoticonID    string `json:"emoticon_id"`
	EmoticonSetID string `json:"emoticon_set_id"`
}

type UserBadge struct {
	ID      ID     `json:"_id"`
	Version string `json:"version"`
}

type UserNoticeParams struct {
	MsgID interface{} `json:"msg_id"`
}

type Streamer struct {
	Name string `json:"name"`
	ID   int64  `json:"id"`
}

type VideoClass struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type Type string

type ContentType string

type ID string

type Source string

type State string

type Emotes struct {
	ThirdParty []Party `json:"thirdParty"`
	FirstParty []Party `json:"firstParty"`
}

type Party struct {
	ID         string      `json:"id"`
	ImageScale int64       `json:"imageScale"`
	Data       string      `json:"data"`
	Name       string      `json:"name"`
	URL        interface{} `json:"url"`
	Width      int64       `json:"width"`
	Height     int64       `json:"height"`
}

type GanymedeEmotes struct {
	Emotes []GanymedeEmote `json:"emotes"`
}

type GanymedeEmote struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	URL    string            `json:"url"`
	Type   GanymedeEmoteType `json:"type"`
	Source string            `json:"source"`
	Width  int64             `json:"width"`
	Height int64             `json:"height"`
}

type GanymedeEmoteType string
