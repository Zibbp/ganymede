package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type TwitchVersion map[string]TwitchItem

type TwitchBadeResp struct {
	BadgeSets map[string]TwitchBadge `json:"badge_sets"`
}

type TwitchBadge map[string]TwitchVersion

type TwitchItem struct {
	ImageUrl1X  string `json:"image_url_1x"`
	ImageUrl2X  string `json:"image_url_2x"`
	ImageUrl4X  string `json:"image_url_4x"`
	Description string `json:"description"`
	Title       string `json:"title"`
	ClickAction string `json:"click_action"`
	ClickUrl    string `json:"click_url"`
}

type BadgeResp struct {
	Badges []GanymedeBadge `json:"badges"`
}

type GanymedeBadge struct {
	Version     string `json:"version"`
	Name        string `json:"name"`
	ImageUrl1X  string `json:"image_url_1x"`
	ImageUrl2X  string `json:"image_url_2x"`
	ImageUrl4X  string `json:"image_url_4x"`
	Description string `json:"description"`
	Title       string `json:"title"`
	ClickAction string `json:"click_action"`
	ClickUrl    string `json:"click_url"`
}

func GetTwitchGlobalBadges() (*BadgeResp, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://badges.twitch.tv/v1/badges/global/display", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var twitchBadgeResp TwitchBadeResp
	if err := json.Unmarshal(body, &twitchBadgeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	var badgeResp BadgeResp

	for k, v := range twitchBadgeResp.BadgeSets {
		for _, v := range v {
			for version, v := range v {
				badge := GanymedeBadge{
					Version:     version,
					Name:        k,
					ImageUrl1X:  v.ImageUrl1X,
					ImageUrl2X:  v.ImageUrl2X,
					ImageUrl4X:  v.ImageUrl4X,
					Description: v.Description,
					Title:       v.Title,
					ClickAction: v.ClickAction,
					ClickUrl:    v.ClickUrl,
				}
				badgeResp.Badges = append(badgeResp.Badges, badge)
			}
		}
	}

	return &badgeResp, nil
}

func GetTwitchChannelBadges(channelId int64) (*BadgeResp, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://badges.twitch.tv/v1/badges/channels/%d/display", channelId), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var twitchBadgeResp TwitchBadeResp
	if err := json.Unmarshal(body, &twitchBadgeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	var badgeResp BadgeResp

	for k, v := range twitchBadgeResp.BadgeSets {
		for _, v := range v {
			for version, v := range v {
				badge := GanymedeBadge{
					Version:     version,
					Name:        k,
					ImageUrl1X:  v.ImageUrl1X,
					ImageUrl2X:  v.ImageUrl2X,
					ImageUrl4X:  v.ImageUrl4X,
					Description: v.Description,
					Title:       v.Title,
					ClickAction: v.ClickAction,
					ClickUrl:    v.ClickUrl,
				}
				badgeResp.Badges = append(badgeResp.Badges, badge)
			}
		}
	}

	return &badgeResp, nil
}
