package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type SevenTVGlobalEmotes struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Tags       []interface{}  `json:"tags"`
	Immutable  bool           `json:"immutable"`
	Privileged bool           `json:"privileged"`
	Emotes     []SevenTVEmote `json:"emotes"`
	Capacity   int64          `json:"capacity"`
	Owner      Owner          `json:"owner"`
}

type SevenTVEmote struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Flags     int64    `json:"flags"`
	Timestamp int64    `json:"timestamp"`
	ActorID   *ActorID `json:"actor_id"`
	Data      Data     `json:"data"`
}

type Data struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Flags     int64  `json:"flags"`
	Lifecycle int64  `json:"lifecycle"`
	Listed    bool   `json:"listed"`
	Animated  bool   `json:"animated"`
	Owner     *Owner `json:"owner,omitempty"`
	Host      Host   `json:"host"`
}

type Host struct {
	URL   string `json:"url"`
	Files []File `json:"files"`
}

type File struct {
	Name       Name          `json:"name"`
	StaticName StaticName    `json:"static_name"`
	Width      int64         `json:"width"`
	Height     int64         `json:"height"`
	Size       int64         `json:"size"`
	Format     SevenTVFormat `json:"format"`
}

type Owner struct {
	ID          string      `json:"id"`
	Username    string      `json:"username"`
	DisplayName string      `json:"display_name"`
	AvatarURL   string      `json:"avatar_url"`
	Style       Style       `json:"style"`
	Roles       []Role      `json:"roles"`
	Connections interface{} `json:"connections"`
}

type Style struct {
	Color int64       `json:"color"`
	Paint interface{} `json:"paint"`
}

type ActorID string

type SevenTVFormat string

type Name string

type StaticName string

type Role string

type SevenTVChannelEmotes struct {
	ID            string   `json:"id"`
	Platform      string   `json:"platform"`
	Username      string   `json:"username"`
	DisplayName   string   `json:"display_name"`
	LinkedAt      int64    `json:"linked_at"`
	EmoteCapacity int64    `json:"emote_capacity"`
	EmoteSet      EmoteSet `json:"emote_set"`
	User          User     `json:"user"`
}

type Connection struct {
	ID            string   `json:"id"`
	Platform      string   `json:"platform"`
	Username      string   `json:"username"`
	DisplayName   string   `json:"display_name"`
	LinkedAt      int64    `json:"linked_at"`
	EmoteCapacity int64    `json:"emote_capacity"`
	EmoteSet      EmoteSet `json:"emote_set"`
}

type User struct {
	ID          string       `json:"id"`
	Username    string       `json:"username"`
	DisplayName string       `json:"display_name"`
	AvatarURL   string       `json:"avatar_url"`
	Style       Style        `json:"style"`
	Roles       []Role       `json:"roles"`
	Connections []Connection `json:"connections"`
	CreatedAt   *int64       `json:"createdAt,omitempty"`
	Biography   *string      `json:"biography,omitempty"`
}

type EmoteData struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Flags     int64  `json:"flags"`
	Lifecycle int64  `json:"lifecycle"`
	Listed    bool   `json:"listed"`
	Animated  bool   `json:"animated"`
	Owner     User   `json:"owner"`
	Host      Host   `json:"host"`
}

type Emote struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Flags     int64     `json:"flags"`
	Timestamp int64     `json:"timestamp"`
	ActorID   *ID       `json:"actor_id"`
	Data      EmoteData `json:"data"`
}

type EmoteSet struct {
	ID         ID            `json:"id"`
	Name       string        `json:"name"`
	Tags       []interface{} `json:"tags"`
	Immutable  bool          `json:"immutable"`
	Privileged bool          `json:"privileged"`
	Emotes     []Emote       `json:"emotes,omitempty"`
	Capacity   int64         `json:"capacity"`
	Owner      *User         `json:"owner"`
}

func Get7TVGlobalEmotes() ([]*GanymedeEmote, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://7tv.io/v3/emote-sets/global", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get global emotes: %v", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %v", err)
	}

	var emotes SevenTVGlobalEmotes
	err = json.Unmarshal(body, &emotes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal emotes: %v", err)
	}

	var ganymedeEmotes []*GanymedeEmote
	for _, emote := range emotes.Emotes {
		ganymedeEmotes = append(ganymedeEmotes, &GanymedeEmote{
			ID:     emote.ID,
			Name:   emote.Name,
			URL:    fmt.Sprintf("https:%s/1x.webp", emote.Data.Host.URL),
			Type:   "third_party",
			Source: "7tv",
			Width:  emote.Data.Host.Files[0].Width,
			Height: emote.Data.Host.Files[0].Height,
		})
	}

	return ganymedeEmotes, nil
}

func Get7TVChannelEmotes(channelId int64) ([]*GanymedeEmote, error) {
	stringChannelId := fmt.Sprintf("%d", channelId)

	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://7tv.io/v3/users/twitch/%s", stringChannelId), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel emotes: %v", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %v", err)
	}

	var emotes SevenTVChannelEmotes
	err = json.Unmarshal(body, &emotes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal emotes: %v", err)
	}

	var ganymedeEmotes []*GanymedeEmote
	for _, emote := range emotes.EmoteSet.Emotes {
		ganymedeEmotes = append(ganymedeEmotes, &GanymedeEmote{
			ID:     emote.ID,
			Name:   emote.Name,
			URL:    fmt.Sprintf("https:%s/1x.webp", emote.Data.Host.URL),
			Type:   "third_party",
			Source: "7tv",
			Width:  emote.Data.Host.Files[0].Width,
			Height: emote.Data.Host.Files[0].Height,
		})
	}

	return ganymedeEmotes, nil
}
