package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type BTTVEmote struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	ImageType ImageType `json:"imageType"`
	UserID    UserID    `json:"userId"`
}

type ImageType string

type UserID string

type BTTVChannelEmotes struct {
	ID            string        `json:"id"`
	Bots          []interface{} `json:"bots"`
	Avatar        string        `json:"avatar"`
	ChannelEmotes []BTTVEmote   `json:"channelEmotes"`
	SharedEmotes  []BTTVEmote   `json:"sharedEmotes"`
}

func GetBTTVGlobalEmotes() ([]*GanymedeEmote, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.betterttv.net/3/cached/emotes/global", nil)
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
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var bttvGlobalEmotes []BTTVEmote
	err = json.Unmarshal(body, &bttvGlobalEmotes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	var emotes []*GanymedeEmote
	for _, emote := range bttvGlobalEmotes {
		emotes = append(emotes, &GanymedeEmote{
			ID:     emote.ID,
			Name:   emote.Code,
			URL:    fmt.Sprintf("https://cdn.betterttv.net/emote/%s/1x", emote.ID),
			Type:   "third_party",
			Source: "bttv",
		})
	}

	return emotes, nil
}

func GetBTTVChannelEmotes(channelId int64) ([]*GanymedeEmote, error) {
	stringChannelId := fmt.Sprintf("%d", channelId)
	client := &http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.betterttv.net/3/cached/users/twitch/%s", stringChannelId), nil)
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
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var bttvChannelEmotes BTTVChannelEmotes
	err = json.Unmarshal(body, &bttvChannelEmotes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	var emotes []*GanymedeEmote
	for _, emote := range bttvChannelEmotes.ChannelEmotes {
		emotes = append(emotes, &GanymedeEmote{
			ID:     emote.ID,
			Name:   emote.Code,
			URL:    fmt.Sprintf("https://cdn.betterttv.net/emote/%s/1x", emote.ID),
			Type:   "third_party",
			Source: "bttv",
		})
	}
	for _, emote := range bttvChannelEmotes.SharedEmotes {
		emotes = append(emotes, &GanymedeEmote{
			ID:     emote.ID,
			Name:   emote.Code,
			URL:    fmt.Sprintf("https://cdn.betterttv.net/emote/%s/1x", emote.ID),
			Type:   "third_party",
			Source: "bttv",
		})
	}

	return emotes, nil
}
