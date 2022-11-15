package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type FFZEmote struct {
	ID        int64     `json:"id"`
	User      FFZUser   `json:"user"`
	Code      string    `json:"code"`
	Images    FFZImages `json:"images"`
	ImageType ImageType `json:"imageType"`
}

type FFZImages struct {
	The1X string  `json:"1x"`
	The2X *string `json:"2x"`
	The4X *string `json:"4x"`
}

type FFZUser struct {
	ID          int64       `json:"id"`
	Name        Name        `json:"name"`
	DisplayName DisplayName `json:"displayName"`
}

type FFZImageType string

type DisplayName string

func GetFFZGlobalEmotes() ([]*GanymedeEmote, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.betterttv.net/3/cached/frankerfacez/emotes/global", nil)
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

	var ffzGlobalEmotes []FFZEmote
	err = json.Unmarshal(body, &ffzGlobalEmotes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	var emotes []*GanymedeEmote
	for _, emote := range ffzGlobalEmotes {
		emotes = append(emotes, &GanymedeEmote{
			ID:     strconv.FormatInt(emote.ID, 10),
			Name:   emote.Code,
			URL:    emote.Images.The1X,
			Type:   "third_party",
			Source: "ffz",
		})
	}

	return emotes, nil
}

func GetFFZChannelEmotes(channelId int64) ([]*GanymedeEmote, error) {
	stringChannelId := fmt.Sprintf("%d", channelId)
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.betterttv.net/3/cached/frankerfacez/users/twitch/%s", stringChannelId), nil)
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

	var ffzChannelEmotes []FFZEmote
	err = json.Unmarshal(body, &ffzChannelEmotes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	var emotes []*GanymedeEmote
	for _, emote := range ffzChannelEmotes {
		emotes = append(emotes, &GanymedeEmote{
			ID:     strconv.FormatInt(emote.ID, 10),
			Name:   emote.Code,
			URL:    emote.Images.The1X,
			Type:   "third_party",
			Source: "ffz",
		})
	}

	return emotes, nil
}
