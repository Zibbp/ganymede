package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/zibbp/ganymede/internal/platform"
)

type BTTVEmote struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	ImageType ImageType `json:"imageType"`
	UserID    UserID    `json:"userId"`
	Animated  bool      `json:"animated"`
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

func GetBTTVGlobalEmotes(ctx context.Context) ([]platform.Emote, error) {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.betterttv.net/3/cached/emotes/global", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get global emotes: %v", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("failed to close response body: %v\n", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var bttvGlobalEmotes []BTTVEmote
	err = json.Unmarshal(body, &bttvGlobalEmotes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	var emotes []platform.Emote
	for _, emote := range bttvGlobalEmotes {
		e := platform.Emote{
			ID:     emote.ID,
			Name:   emote.Code,
			URL:    fmt.Sprintf("https://cdn.betterttv.net/emote/%s/1x", emote.ID),
			Format: platform.EmoteFormatStatic,
			Type:   platform.EmoteTypeGlobal,
			Source: "bttv",
		}
		if emote.Animated {
			e.Format = platform.EmoteFormatAnimated
		}

		emotes = append(emotes, e)
	}

	return emotes, nil
}

func GetBTTVChannelEmotes(ctx context.Context, channelId string) ([]platform.Emote, error) {
	client := &http.Client{}

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://api.betterttv.net/3/cached/users/twitch/%s", channelId), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel emotes: %v", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("failed to close response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var bttvChannelEmotes BTTVChannelEmotes
	err = json.Unmarshal(body, &bttvChannelEmotes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	var emotes []platform.Emote
	for _, emote := range bttvChannelEmotes.ChannelEmotes {
		e := platform.Emote{
			ID:     emote.ID,
			Name:   emote.Code,
			URL:    fmt.Sprintf("https://cdn.betterttv.net/emote/%s/1x", emote.ID),
			Format: platform.EmoteFormatStatic,
			Type:   platform.EmoteTypeGlobal,
			Source: "bttv",
		}
		if emote.Animated {
			e.Format = platform.EmoteFormatAnimated
		}

		emotes = append(emotes, e)
	}
	for _, emote := range bttvChannelEmotes.SharedEmotes {
		e := platform.Emote{
			ID:     emote.ID,
			Name:   emote.Code,
			URL:    fmt.Sprintf("https://cdn.betterttv.net/emote/%s/1x", emote.ID),
			Format: platform.EmoteFormatStatic,
			Type:   platform.EmoteTypeGlobal,
			Source: "bttv",
		}
		if emote.Animated {
			e.Format = platform.EmoteFormatAnimated
		}

		emotes = append(emotes, e)
	}

	return emotes, nil
}
