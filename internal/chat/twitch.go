package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type TwitchGlobalEmotes struct {
	Data     []TwitchGlobalEmote `json:"data"`
	Template string              `json:"template"`
}

type TwitchGlobalEmote struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Images    Images      `json:"images"`
	Format    []Format    `json:"format"`
	Scale     []string    `json:"scale"`
	ThemeMode []ThemeMode `json:"theme_mode"`
}

type Images struct {
	URL1X string `json:"url_1x"`
	URL2X string `json:"url_2x"`
	URL4X string `json:"url_4x"`
}

type Format string

const (
	Static Format = "static"
)

type ThemeMode string

const (
	Dark  ThemeMode = "dark"
	Light ThemeMode = "light"
)

func GetTwitchGlobalEmotes() ([]*GanymedeEmote, error) {
	accessToken := os.Getenv("TWITCH_ACCESS_TOKEN")
	clientId := os.Getenv("TWITCH_CLIENT_ID")
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.twitch.tv/helix/chat/emotes/global", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Client-ID", clientId)
	req.Header.Add("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get global emotes: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get global emotes: %v", resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var twitchGlobalEmotes TwitchGlobalEmotes
	err = json.Unmarshal(body, &twitchGlobalEmotes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	var emotes []*GanymedeEmote
	for _, emote := range twitchGlobalEmotes.Data {
		// convert string to *string
		emotes = append(emotes, &GanymedeEmote{
			ID:   emote.ID,
			Name: emote.Name,
			URL:  emote.Images.URL1X,
			Type: "twitch",
		})
	}

	return emotes, nil
}

func GetTwitchChannelEmotes(channelId int64) ([]*GanymedeEmote, error) {
	accessToken := os.Getenv("TWITCH_ACCESS_TOKEN")
	clientId := os.Getenv("TWITCH_CLIENT_ID")
	stringChannelId := fmt.Sprintf("%d", channelId)
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.twitch.tv/helix/chat/emotes?broadcaster_id="+stringChannelId, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Client-ID", clientId)
	req.Header.Add("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel emotes: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get channel emotes: %v", resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var twitchChannelEmotes TwitchGlobalEmotes
	err = json.Unmarshal(body, &twitchChannelEmotes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	var emotes []*GanymedeEmote
	for _, emote := range twitchChannelEmotes.Data {
		emotes = append(emotes, &GanymedeEmote{
			ID:   emote.ID,
			Name: emote.Name,
			URL:  emote.Images.URL1X,
			Type: "twitch",
		})
	}

	return emotes, nil
}
