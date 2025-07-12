package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/chapter"
	"github.com/zibbp/ganymede/internal/dto"
	"github.com/zibbp/ganymede/internal/utils"
)

// GetVideo implements the Platform interface to get video information from Twitch. Optional parameters are chapters and muted segments. These use the undocumented Twitch GraphQL API.
func (c *TwitchConnection) GetVideo(ctx context.Context, id string, withChapters bool, withMutedSegments bool) (*VideoInfo, error) {
	params := url.Values{
		"id": []string{id},
	}
	body, err := c.twitchMakeHTTPRequest("GET", "videos", params, nil)
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

	// TODO: fix for restriction (sub-only)
	gqlVideo, err := c.TwitchGQLGetVideo(id)
	if err != nil {
		return nil, err
	}

	// parse dates
	createdAt, err := time.Parse(time.RFC3339, videoResponse.Data[0].CreatedAt)
	if err != nil {
		return nil, err
	}
	publishedAt, err := time.Parse(time.RFC3339, videoResponse.Data[0].PublishedAt)
	if err != nil {
		return nil, err
	}

	// get duration
	duration, err := time.ParseDuration(videoResponse.Data[0].Duration)
	if err != nil {
		return nil, fmt.Errorf("error parsing duration: %v", err)
	}

	info := VideoInfo{
		ID:                          videoResponse.Data[0].ID,
		StreamID:                    videoResponse.Data[0].StreamID,
		UserID:                      videoResponse.Data[0].UserID,
		UserLogin:                   videoResponse.Data[0].UserLogin,
		UserName:                    videoResponse.Data[0].UserName,
		Title:                       videoResponse.Data[0].Title,
		Description:                 videoResponse.Data[0].Description,
		CreatedAt:                   createdAt,
		PublishedAt:                 publishedAt,
		URL:                         videoResponse.Data[0].URL,
		ThumbnailURL:                videoResponse.Data[0].ThumbnailURL,
		Viewable:                    videoResponse.Data[0].Viewable,
		ViewCount:                   videoResponse.Data[0].ViewCount,
		Language:                    videoResponse.Data[0].Language,
		Type:                        videoResponse.Data[0].Type,
		Duration:                    duration,
		Category:                    &gqlVideo.Game.Name,
		SpriteThumbnailsManifestUrl: &gqlVideo.SeekPreviewsURL,
	}

	// get chapters
	if withChapters {
		gqlChapters, err := c.TwitchGQLGetChapters(info.ID)
		if err != nil {
			return nil, err
		}

		var chapters []chapter.Chapter
		convertedChapters, err := convertTwitchChaptersToChapters(gqlChapters, int(info.Duration.Seconds()))
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
	params := url.Values{
		"user_login": []string{channelName},
	}
	body, err := c.twitchMakeHTTPRequest("GET", "streams", params, nil)
	if err != nil {
		return nil, err
	}

	var resp TwitchLiveStreamsRepsponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("failed to fetch stream for channel %s: %w", channelName, ErrorNoStreamsFound{})
	}

	startedAt, err := time.Parse(time.RFC3339, resp.Data[0].StartedAt)
	if err != nil {
		return nil, err
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
		StartedAt:    startedAt,
		Language:     resp.Data[0].Language,
		ThumbnailURL: resp.Data[0].ThumbnailURL,
	}

	return &info, nil
}

func (c *TwitchConnection) GetLiveStreams(ctx context.Context, channelNames []string) ([]LiveStreamInfo, error) {
	params := url.Values{}
	for _, channel := range channelNames {
		params.Add("user_login", channel)
	}

	body, err := c.twitchMakeHTTPRequest("GET", "streams", params, nil)
	if err != nil {
		return nil, err
	}

	var resp TwitchLiveStreamsRepsponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("failed to fetch stream for channels: %w", ErrorNoStreamsFound{})
	}

	streams := make([]LiveStreamInfo, 0, len(resp.Data))
	for _, stream := range resp.Data {
		startedAt, err := time.Parse(time.RFC3339, stream.StartedAt)
		if err != nil {
			return nil, err
		}

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
			StartedAt:    startedAt,
			Language:     stream.Language,
			ThumbnailURL: stream.ThumbnailURL,
		})
	}

	return streams, nil
}

func (c *TwitchConnection) GetChannel(ctx context.Context, channelName string) (*ChannelInfo, error) {
	params := url.Values{
		"login": []string{channelName},
	}
	body, err := c.twitchMakeHTTPRequest("GET", "users", params, nil)
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

	createdAt, err := time.Parse(time.RFC3339, resp.Data[0].CreatedAt)
	if err != nil {
		return nil, err
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
		CreatedAt:       createdAt,
	}

	return &info, nil
}

func (c *TwitchConnection) GetVideos(ctx context.Context, channelId string, videoType VideoType, withChapters bool, withMutedSegments bool) ([]VideoInfo, error) {
	params := url.Values{
		"user_id": []string{channelId},
		"first":   []string{"100"},
		"type":    []string{string(videoType)},
	}
	body, err := c.twitchMakeHTTPRequest("GET", "videos", params, nil)
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
		params.Del("after")
		params.Set("after", cursor)
		body, err = c.twitchMakeHTTPRequest("GET", "videos", params, nil)
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
		// if withChapters or withMutedSegments is true, get the video from the GetVideo function which fetches extra information
		// else just use the video from the API response
		if withChapters || withMutedSegments {
			video, err := c.GetVideo(ctx, video.ID, withChapters, withMutedSegments)
			if err != nil {
				return nil, err
			}

			info = append(info, *video)
		} else {

			// parse dates
			createdAt, err := time.Parse(time.RFC3339, video.CreatedAt)
			if err != nil {
				return nil, err
			}
			publishedAt, err := time.Parse(time.RFC3339, video.PublishedAt)
			if err != nil {
				return nil, err
			}
			// get duration
			duration, err := time.ParseDuration(video.Duration)
			if err != nil {
				return nil, fmt.Errorf("error parsing duration: %v", err)
			}

			info = append(info, VideoInfo{
				ID:           video.ID,
				StreamID:     video.StreamID,
				UserID:       video.UserID,
				UserLogin:    video.UserLogin,
				UserName:     video.UserName,
				Title:        video.Title,
				Description:  video.Description,
				CreatedAt:    createdAt,
				PublishedAt:  publishedAt,
				URL:          video.URL,
				ThumbnailURL: video.ThumbnailURL,
				Viewable:     video.Viewable,
				ViewCount:    video.ViewCount,
				Language:     video.Language,
				Type:         video.Type,
				Duration:     duration,
			})
		}
	}

	return info, nil
}

func (c *TwitchConnection) GetCategories(ctx context.Context) ([]Category, error) {
	params := url.Values{}
	body, err := c.twitchMakeHTTPRequest("GET", "games/top", params, nil)
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
		params.Del("after")
		params.Set("after", cursor)
		body, err = c.twitchMakeHTTPRequest("GET", "games/top", params, nil)
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
	params := url.Values{
		"broadcaster_id": []string{channelId},
	}
	body, err := c.twitchMakeHTTPRequest("GET", "chat/badges", params, nil)
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
	params := url.Values{
		"broadcaster_id": []string{channelId},
	}
	body, err := c.twitchMakeHTTPRequest("GET", "chat/emotes", params, nil)
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

// GetChannelClips gets a Twitch channel's clip with some filter options. Twitch returns the clips sorted by view count descending.
func (c *TwitchConnection) GetChannelClips(ctx context.Context, channelId string, filter ClipsFilter) ([]ClipInfo, error) {

	internalLimit := filter.Limit

	// set limit to 100 if limit is 0 or greater than 100
	if filter.Limit == 0 || filter.Limit > 100 {
		internalLimit = 100
	}

	limitStr := strconv.Itoa(internalLimit)
	params := url.Values{
		"broadcaster_id": []string{channelId},
		"started_at":     []string{filter.StartedAt.Format(time.RFC3339)},
		"ended_at":       []string{filter.EndedAt.Format(time.RFC3339)},
		"first":          []string{limitStr},
	}

	body, err := c.twitchMakeHTTPRequest("GET", "clips", params, nil)
	if err != nil {
		return nil, err
	}

	var resp TwitchGetClipsResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	var clips []TwitchClip
	clips = append(clips, resp.Data...)

	// pagination
	cursor := resp.Pagination.Cursor
	for cursor != "" {
		params.Del("after")
		params.Set("after", cursor)

		body, err := c.twitchMakeHTTPRequest("GET", "clips", params, nil)
		if err != nil {
			return nil, err
		}

		var resp TwitchGetClipsResponse
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return nil, err
		}

		clips = append(clips, resp.Data...)
		cursor = resp.Pagination.Cursor

		// break if limit is reached except if limit is 0 (all clips)
		if filter.Limit != 0 {
			if len(clips) >= filter.Limit {
				break
			}
		}
	}

	var info []ClipInfo
	for _, clip := range clips {
		// parse dates
		createdAt, err := time.Parse(time.RFC3339, clip.CreatedAt)
		if err != nil {
			return nil, err
		}

		offset := 0
		if clip.VodOffset != nil {
			if vodOffset, ok := clip.VodOffset.(int); ok {
				offset = vodOffset
			}
		}

		info = append(info, ClipInfo{
			ID:           clip.ID,
			URL:          clip.URL,
			ChannelID:    clip.BroadcasterID,
			ChannelName:  &clip.BroadcasterName,
			CreatorID:    &clip.CreatorID,
			CreatorName:  &clip.CreatorName,
			VideoID:      clip.VideoID,
			GameID:       &clip.GameID,
			Language:     &clip.Language,
			Title:        clip.Title,
			ViewCount:    clip.ViewCount,
			CreatedAt:    createdAt,
			ThumbnailURL: clip.ThumbnailURL,
			Duration:     int(clip.Duration),
			VodOffset:    &offset,
		})
	}

	// get exact number of clips if limit is set
	if filter.Limit != 0 && filter.Limit < len(info) {
		info = info[:filter.Limit]
	}

	return info, nil
}

// GetClip gets a Twitch clip given it's ID
func (c *TwitchConnection) GetClip(ctx context.Context, id string) (*ClipInfo, error) {
	params := url.Values{
		"id": []string{id},
	}

	body, err := c.twitchMakeHTTPRequest("GET", "clips", params, nil)
	if err != nil {
		return nil, err
	}

	var resp TwitchGetClipsResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	var clips []TwitchClip
	clips = append(clips, resp.Data...)

	if len(clips) == 0 {
		return nil, fmt.Errorf("clip not found")
	}

	var info ClipInfo
	for _, clip := range clips {
		// parse dates
		createdAt, err := time.Parse(time.RFC3339, clip.CreatedAt)
		if err != nil {
			return nil, err
		}

		// Parse clip vod offset
		offset := 0
		if clip.VodOffset != nil {
			switch v := clip.VodOffset.(type) {
			case int:
				offset = v
			case float64: // If VodOffset might be a float
				offset = int(v) // Convert to int
			case string:
				if parsed, err := strconv.Atoi(v); err == nil {
					offset = parsed
				} else {
					log.Warn().Msgf("failed to convert VodOffset string to int: %v", err)
				}
			default:
				log.Warn().Msgf("VodOffset is an unsupported type, unable to convert:  %T\n", v)
			}
		}

		info = ClipInfo{
			ID:           clip.ID,
			URL:          clip.URL,
			ChannelID:    clip.BroadcasterID,
			ChannelName:  &clip.BroadcasterName,
			CreatorID:    &clip.CreatorID,
			CreatorName:  &clip.CreatorName,
			VideoID:      clip.VideoID,
			GameID:       &clip.GameID,
			Language:     &clip.Language,
			Title:        clip.Title,
			ViewCount:    clip.ViewCount,
			CreatedAt:    createdAt,
			ThumbnailURL: clip.ThumbnailURL,
			Duration:     int(clip.Duration),
			VodOffset:    &offset,
		}
	}

	return &info, nil
}

// CheckIfStreamIsLive checks if a Twitch stream is live by attempting to parse the m3u8 playlist of the stream.
func (c *TwitchConnection) CheckIfStreamIsLive(ctx context.Context, channelName string) (bool, error) {
	token, err := c.TwitchGQLGetPlaybackAccessToken(channelName)
	if err != nil {
		return false, fmt.Errorf("failed to get playback access token: %v", err)
	}
	// Construct the m3u8 URL for the stream
	m3u8URL := fmt.Sprintf("https://usher.ttvnw.net/api/channel/hls/%s.m3u8?sig=%s&token=%s", channelName, token.Signature, token.Value)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	// HTTP request to fetch the m3u8 playlist
	req, err := http.NewRequest("GET", m3u8URL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", utils.ChromeUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to fetch m3u8 playlist: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	// If the response status is not 200 or 403 the stream is not live
	// This request is not authenticated, so it can return 403 if the stream is sub-only or geo-blocked
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusForbidden {
		return false, fmt.Errorf("received unexpected status code: %d", resp.StatusCode)
	}

	return true, nil
}
