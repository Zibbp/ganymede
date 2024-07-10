package platform

import (
	"context"
	"encoding/json"
	"fmt"
)

func (c *TwitchConnection) GetVideo(ctx context.Context, id string) (*VideoInfo, error) {
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
		ID:            videoResponse.Data[0].ID,
		StreamID:      videoResponse.Data[0].StreamID,
		UserID:        videoResponse.Data[0].UserID,
		UserLogin:     videoResponse.Data[0].UserLogin,
		UserName:      videoResponse.Data[0].UserName,
		Title:         videoResponse.Data[0].Title,
		Description:   videoResponse.Data[0].Description,
		CreatedAt:     videoResponse.Data[0].CreatedAt,
		PublishedAt:   videoResponse.Data[0].PublishedAt,
		URL:           videoResponse.Data[0].URL,
		ThumbnailURL:  videoResponse.Data[0].ThumbnailURL,
		Viewable:      videoResponse.Data[0].Viewable,
		ViewCount:     videoResponse.Data[0].ViewCount,
		Language:      videoResponse.Data[0].Language,
		Type:          videoResponse.Data[0].Type,
		Duration:      videoResponse.Data[0].Duration,
		MutedSegments: videoResponse.Data[0].MutedSegments,
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

func (c *TwitchConnection) GetVideos(ctx context.Context, channelId string, videoType string) ([]VideoInfo, error) {
	queryParams := map[string]string{"user_id": channelId, "first": "100", "type": videoType}
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
			ID:            video.ID,
			StreamID:      video.StreamID,
			UserID:        video.UserID,
			UserLogin:     video.UserLogin,
			UserName:      video.UserName,
			Title:         video.Title,
			Description:   video.Description,
			CreatedAt:     video.CreatedAt,
			PublishedAt:   video.PublishedAt,
			URL:           video.URL,
			ThumbnailURL:  video.ThumbnailURL,
			Viewable:      video.Viewable,
			ViewCount:     video.ViewCount,
			Language:      video.Language,
			Type:          video.Type,
			Duration:      video.Duration,
			MutedSegments: video.MutedSegments,
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
