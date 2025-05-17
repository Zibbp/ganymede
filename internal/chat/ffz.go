package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/platform"
)

type FFZEmote struct {
	ID        int64     `json:"id"`
	User      FFZUser   `json:"user"`
	Code      string    `json:"code"`
	Images    FFZImages `json:"images"`
	ImageType ImageType `json:"imageType"`
	Animated  bool      `json:"animated"`
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

func GetFFZGlobalEmotes(ctx context.Context) ([]platform.Emote, error) {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.betterttv.net/3/cached/frankerfacez/emotes/global", nil)
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

	var ffzGlobalEmotes []FFZEmote
	err = json.Unmarshal(body, &ffzGlobalEmotes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	var emotes []platform.Emote
	for _, emote := range ffzGlobalEmotes {
		e := platform.Emote{
			ID:     strconv.FormatInt(emote.ID, 10),
			Name:   emote.Code,
			URL:    emote.Images.The1X,
			Format: platform.EmoteFormatStatic,
			Type:   platform.EmoteTypeGlobal,
			Source: "ffz",
		}
		if emote.Animated {
			e.Format = platform.EmoteFormatAnimated
		}

		emotes = append(emotes, e)
	}

	return emotes, nil
}

func GetFFZChannelEmotes(ctx context.Context, channelId string) ([]platform.Emote, error) {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://api.betterttv.net/3/cached/frankerfacez/users/twitch/%s", channelId), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get global emotes: %v", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Debug().Err(err).Msg("failed to close response body")
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var ffzChannelEmotes []FFZEmote
	err = json.Unmarshal(body, &ffzChannelEmotes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	var emotes []platform.Emote
	for _, emote := range ffzChannelEmotes {
		e := platform.Emote{
			ID:     strconv.FormatInt(emote.ID, 10),
			Name:   emote.Code,
			URL:    emote.Images.The1X,
			Format: platform.EmoteFormatStatic,
			Type:   platform.EmoteTypeGlobal,
			Source: "ffz",
		}
		if emote.Animated {
			e.Format = platform.EmoteFormatAnimated
		}

		emotes = append(emotes, e)
	}

	return emotes, nil
}
