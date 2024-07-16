package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zibbp/ganymede/internal/chapter"
	"github.com/zibbp/ganymede/internal/dto"
	"github.com/zibbp/ganymede/internal/utils"
)

// GetVideo implements the Platform interface to get video information from Twitch. Optional parameters are chapters and muted segments. These use the undocumented Twitch GraphQL API.
func (c *TwitchConnection) GetVideo(ctx context.Context, id string, withChapters bool, withMutedSegments bool) (*VideoInfo, error) {
	queryParams := map[string]string{"id": id}
	body, err := c.twitchMakeHTTPRequest("GET", "videos", queryParams, nil)
	if err != nil {
		return nil, err
	}

	var videoResponse TwitchGetVideosResponse
	err = json.Unmarshal(body, &videoResponse)
	if err != nil {
		return nil, err
	}

	if len(videoResponse.Data) == 0 {
		return nil, fmt.Errorf("video not found")
	}

	info := VideoInfo{
		ID:           videoResponse.Data[0].ID,
		StreamID:     videoResponse.Data[0].StreamID,
		UserID:       videoResponse.Data[0].UserID,
		UserLogin:    videoResponse.Data[0].UserLogin,
		UserName:     videoResponse.Data[0].UserName,
		Title:        videoResponse.Data[0].Title,
		Description:  videoResponse.Data[0].Description,
		CreatedAt:    videoResponse.Data[0].CreatedAt,
		PublishedAt:  videoResponse.Data[0].PublishedAt,
		URL:          videoResponse.Data[0].URL,
		ThumbnailURL: videoResponse.Data[0].ThumbnailURL,
		Viewable:     videoResponse.Data[0].Viewable,
		ViewCount:    videoResponse.Data[0].ViewCount,
		Language:     videoResponse.Data[0].Language,
		Type:         videoResponse.Data[0].Type,
		Duration:     videoResponse.Data[0].Duration,
	}

	// get chapters
	if withChapters {
		gqlChapters, err := c.TwitchGQLGetChapters(info.ID)
		if err != nil {
			return nil, err
		}

		parsedDuration, err := time.ParseDuration(info.Duration)
		if err != nil {
			return &info, fmt.Errorf("error parsing duration: %v", err)
		}

		var chapters []chapter.Chapter
		convertedChapters, err := convertTwitchChaptersToChapters(gqlChapters, int(parsedDuration.Seconds()))
		if err != nil {
			return &info, err
		}
		chapters = append(chapters, convertedChapters...)
		info.Chapters = chapters
	}

	// get muted segments
	if withMutedSegments {
		gqlMutedSegments, err := c.TwitchGQLGetMutedSegments(info.ID)
		if err != nil {
			return nil, err
		}

		var mutedSegments []MutedSegment

		for _, segment := range gqlMutedSegments {
			mutedSegment := MutedSegment{
				Duration: segment.Duration,
				Offset:   segment.Offset,
			}
			mutedSegments = append(mutedSegments, mutedSegment)
		}
		info.MutedSegments = mutedSegments
	}

	return &info, nil
}

func (c *TwitchConnection) GetLiveStream(ctx context.Context, channelName string) (*LiveStreamInfo, error) {
	queryParams := map[string]string{"user_login": channelName}
	body, err := c.twitchMakeHTTPRequest("GET", "streams", queryParams, nil)
	if err != nil {
		return nil, err
	}

	var resp TwitchLiveStreamsRepsponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no streams found")
	}

	info := LiveStreamInfo{
		ID:           resp.Data[0].ID,
		UserID:       resp.Data[0].UserID,
		UserLogin:    resp.Data[0].UserLogin,
		UserName:     resp.Data[0].UserName,
		GameID:       resp.Data[0].GameID,
		GameName:     resp.Data[0].GameName,
		Type:         resp.Data[0].Type,
		Title:        resp.Data[0].Title,
		ViewerCount:  resp.Data[0].ViewerCount,
		StartedAt:    resp.Data[0].StartedAt,
		Language:     resp.Data[0].Language,
		ThumbnailURL: resp.Data[0].ThumbnailURL,
	}

	return &info, nil
}

func (c *TwitchConnection) GetLiveStreams(ctx context.Context, channelNames []string) ([]LiveStreamInfo, error) {
	queryParams := map[string]string{}

	for _, channelName := range channelNames {
		queryParams["user_login"] = channelName
	}

	body, err := c.twitchMakeHTTPRequest("GET", "streams", queryParams, nil)
	if err != nil {
		return nil, err
	}

	var resp TwitchLiveStreamsRepsponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, &ErrorNoStreamsFound{}
	}

	streams := make([]LiveStreamInfo, 0, len(resp.Data))
	for _, stream := range resp.Data {
		streams = append(streams, LiveStreamInfo{
			ID:           stream.ID,
			UserID:       stream.UserID,
			UserLogin:    stream.UserLogin,
			UserName:     stream.UserName,
			GameID:       stream.GameID,
			GameName:     stream.GameName,
			Type:         stream.Type,
			Title:        stream.Title,
			ViewerCount:  stream.ViewerCount,
			StartedAt:    stream.StartedAt,
			Language:     stream.Language,
			ThumbnailURL: stream.ThumbnailURL,
		})
	}

	return streams, nil
}

func (c *TwitchConnection) GetChannel(ctx context.Context, channelName string) (*ChannelInfo, error) {
	queryParams := map[string]string{"login": channelName}
	body, err := c.twitchMakeHTTPRequest("GET", "users", queryParams, nil)
	if err != nil {
		return nil, err
	}

	var resp TwitchChannelResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("channel not found")
	}

	info := ChannelInfo{
		ID:              resp.Data[0].ID,
		Login:           resp.Data[0].Login,
		DisplayName:     resp.Data[0].DisplayName,
		Type:            resp.Data[0].Type,
		BroadcasterType: resp.Data[0].BroadcasterType,
		Description:     resp.Data[0].Description,
		ProfileImageURL: resp.Data[0].ProfileImageURL,
		OfflineImageURL: resp.Data[0].OfflineImageURL,
		ViewCount:       resp.Data[0].ViewCount,
		CreatedAt:       resp.Data[0].CreatedAt,
	}

	return &info, nil
}

func (c *TwitchConnection) GetVideos(ctx context.Context, channelId string, videoType VideoType) ([]VideoInfo, error) {
	queryParams := map[string]string{"user_id": channelId, "first": "100", "type": string(videoType)}
	body, err := c.twitchMakeHTTPRequest("GET", "videos", queryParams, nil)
	if err != nil {
		return nil, err
	}

	var resp TwitchGetVideosResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	var videos []TwitchVideoInfo
	videos = append(videos, resp.Data...)

	// pagination
	cursor := resp.Pagination.Cursor
	for cursor != "" {
		queryParams["after"] = cursor
		body, err = c.twitchMakeHTTPRequest("GET", "videos", queryParams, nil)
		if err != nil {
			return nil, err
		}
		var resp TwitchGetVideosResponse
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return nil, err
		}
		videos = append(videos, resp.Data...)
		cursor = resp.Pagination.Cursor
	}

	var info []VideoInfo
	for _, video := range videos {
		info = append(info, VideoInfo{
			ID:           video.ID,
			StreamID:     video.StreamID,
			UserID:       video.UserID,
			UserLogin:    video.UserLogin,
			UserName:     video.UserName,
			Title:        video.Title,
			Description:  video.Description,
			CreatedAt:    video.CreatedAt,
			PublishedAt:  video.PublishedAt,
			URL:          video.URL,
			ThumbnailURL: video.ThumbnailURL,
			Viewable:     video.Viewable,
			ViewCount:    video.ViewCount,
			Language:     video.Language,
			Type:         video.Type,
			Duration:     video.Duration,
		})
	}

	return info, nil
}

func (c *TwitchConnection) GetCategories(ctx context.Context) ([]Category, error) {
	queryParams := map[string]string{}
	body, err := c.twitchMakeHTTPRequest("GET", "games/top", queryParams, nil)
	if err != nil {
		return nil, err
	}

	var resp TwitchCategoryResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	var categories []TwitchCategory
	categories = append(categories, resp.Data...)

	// pagination
	cursor := resp.Pagination.Cursor
	for cursor != "" {
		queryParams["after"] = cursor
		body, err = c.twitchMakeHTTPRequest("GET", "games/top", queryParams, nil)
		if err != nil {
			return nil, err
		}
		var resp TwitchCategoryResponse
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return nil, err
		}
		categories = append(categories, resp.Data...)
		cursor = resp.Pagination.Cursor
	}

	var info []Category
	for _, category := range categories {
		info = append(info, Category{
			ID:   category.ID,
			Name: category.Name,
		})
	}

	return info, nil
}

func (c *TwitchConnection) GetGlobalBadges(ctx context.Context) ([]Badge, error) {
	body, err := c.twitchMakeHTTPRequest("GET", "chat/badges/global", nil, nil)
	if err != nil {
		return nil, err
	}

	var twitchGlobalBadges TwitchGlobalBadgeResponse
	err = json.Unmarshal(body, &twitchGlobalBadges)
	if err != nil {
		return nil, err
	}

	if len(twitchGlobalBadges.Data) == 0 {
		return nil, fmt.Errorf("badges not found")
	}

	var badges []Badge

	for _, v := range twitchGlobalBadges.Data {
		for _, b := range v.Versions {
			badges = append(badges, Badge{
				Version:     b.ID,
				Name:        v.SetID,
				IamgeUrl:    b.ImageURL4X,
				ImageUrl1X:  b.ImageURL1X,
				ImageUrl2X:  b.ImageURL2X,
				ImageUrl4X:  b.ImageURL4X,
				Description: b.Description,
				Title:       b.Title,
				ClickAction: b.ClickAction,
				ClickUrl:    b.ClickURL,
			})
		}
	}

	return badges, nil
}

func (c *TwitchConnection) GetChannelBadges(ctx context.Context, channelId string) ([]Badge, error) {
	queryParams := map[string]string{"broadcaster_id": channelId}
	body, err := c.twitchMakeHTTPRequest("GET", "chat/badges", queryParams, nil)
	if err != nil {
		return nil, err
	}

	var twitchGlobalBadges TwitchGlobalBadgeResponse
	err = json.Unmarshal(body, &twitchGlobalBadges)
	if err != nil {
		return nil, err
	}

	if len(twitchGlobalBadges.Data) == 0 {
		return nil, fmt.Errorf("badges not found")
	}

	var badges []Badge

	for _, v := range twitchGlobalBadges.Data {
		for _, b := range v.Versions {
			badges = append(badges, Badge{
				Version:     b.ID,
				Name:        v.SetID,
				IamgeUrl:    b.ImageURL4X,
				ImageUrl1X:  b.ImageURL1X,
				ImageUrl2X:  b.ImageURL2X,
				ImageUrl4X:  b.ImageURL4X,
				Description: b.Description,
				Title:       b.Title,
				ClickAction: b.ClickAction,
				ClickUrl:    b.ClickURL,
			})
		}
	}

	return badges, nil
}

func (c *TwitchConnection) GetGlobalEmotes(ctx context.Context) ([]Emote, error) {
	body, err := c.twitchMakeHTTPRequest("GET", "chat/emotes/global", nil, nil)
	if err != nil {
		return nil, err
	}

	var twitchGlobalEmotes TwitchGlobalEmoteResponse
	err = json.Unmarshal(body, &twitchGlobalEmotes)
	if err != nil {
		return nil, err
	}

	if len(twitchGlobalEmotes.Data) == 0 {
		return nil, fmt.Errorf("emotes not found")
	}

	var emotes []Emote

	// https://dev.twitch.tv/docs/api/reference/#get-global-emotes
	for _, e := range twitchGlobalEmotes.Data {
		emote := Emote{
			ID:     e.ID,
			Name:   e.Name,
			Source: "twitch",
			Type:   EmoteTypeGlobal,
		}

		// check if emote is static or animated
		// format can be static or animated
		if utils.Contains(e.Format, "animated") {
			emote.Format = EmoteFormatAnimated
		} else {
			emote.Format = EmoteFormatStatic
		}

		emote.Scale = twitchEmoteGetLargestScale(e.Scale)

		emote.URL = twitchTemplateEmoteURL(e.ID, string(emote.Format), "dark", emote.Scale)

		emotes = append(emotes, emote)
	}

	return emotes, nil
}

func (c *TwitchConnection) GetChannelEmotes(ctx context.Context, channelId string) ([]Emote, error) {
	queryParams := map[string]string{"broadcaster_id": channelId}
	body, err := c.twitchMakeHTTPRequest("GET", "chat/emotes", queryParams, nil)
	if err != nil {
		return nil, err
	}

	var twitchGlobalEmotes TwitchGlobalEmoteResponse
	err = json.Unmarshal(body, &twitchGlobalEmotes)
	if err != nil {
		return nil, err
	}

	if len(twitchGlobalEmotes.Data) == 0 {
		return nil, fmt.Errorf("emotes not found")
	}

	var emotes []Emote

	// https://dev.twitch.tv/docs/api/reference/#get-global-emotes
	for _, e := range twitchGlobalEmotes.Data {
		emote := Emote{
			ID:     e.ID,
			Name:   e.Name,
			Source: "twitch",
			Type:   EmoteTypeSubscription,
		}

		// check if emote is static or animated
		// format can be static or animated
		if utils.Contains(e.Format, "animated") {
			emote.Format = EmoteFormatAnimated
		} else {
			emote.Format = EmoteFormatStatic
		}

		emote.Scale = twitchEmoteGetLargestScale(e.Scale)

		emote.URL = twitchTemplateEmoteURL(e.ID, string(emote.Format), "dark", emote.Scale)

		emotes = append(emotes, emote)
	}

	return emotes, nil
}

// twitchEmoteGetLargestScale returns the largest scale of the given values
//
// https://dev.twitch.tv/docs/api/reference/#get-global-emotes
func twitchEmoteGetLargestScale(values []string) string {
	if len(values) == 0 {
		return "0"
	}

	highest, err := strconv.ParseFloat(values[0], 64)
	if err != nil {
		return "0"
	}

	for _, v := range values[1:] {
		current, err := strconv.ParseFloat(v, 64)
		if err != nil {
			continue
		}
		if current > highest {
			highest = current
		}
	}

	return strconv.FormatFloat(highest, 'f', 1, 64)
}

// twitchTemplateEmoteURL returns the URL of an emote
//
// https://dev.twitch.tv/docs/api/reference/#get-global-emotes
//
// Twitch recommends using the template URL rather than the raw URL
func twitchTemplateEmoteURL(id, format, themeMode string, scale string) string {
	template := "https://static-cdn.jtvnw.net/emoticons/v2/{{id}}/{{format}}/{{theme_mode}}/{{scale}}"

	replacements := map[string]string{
		"{{id}}":         id,
		"{{format}}":     format,
		"{{theme_mode}}": themeMode,
		"{{scale}}":      scale,
	}

	for placeholder, value := range replacements {
		template = strings.Replace(template, placeholder, value, 1)
	}

	return template
}

func ArchiveVideoActivity(ctx context.Context, input dto.ArchiveVideoInput) error {
	return nil
}
