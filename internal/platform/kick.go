package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

func (c *KickConnection) GetVideo(ctx context.Context, id string, withChapters bool, withMutedSegments bool) (*VideoInfo, error) {
	body, err := c.kickMakeHTTPRequest(KickPrivateApiUrl, "GET", fmt.Sprintf("video/%s", id), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get video: %w", err)
	}

	var resp KickVideo
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal video response: %v", err)
	}

	durationSeconds := resp.Livestream.Duration / 1000

	firstCategory := "unknown"
	if len(resp.Livestream.Categories) > 0 {
		firstCategory = resp.Livestream.Categories[0].Name
	}

	info := VideoInfo{
		ID:                          fmt.Sprintf("%d", resp.ID),
		StreamID:                    fmt.Sprintf("%d", resp.LiveStreamID),
		UserID:                      fmt.Sprintf("%d", resp.Livestream.Channel.UserID),
		UserLogin:                   resp.Livestream.Channel.Slug,
		UserName:                    resp.Livestream.Channel.User.Username,
		Title:                       resp.Livestream.SessionTitle,
		Description:                 resp.Livestream.SessionTitle,
		CreatedAt:                   resp.CreatedAt,
		PublishedAt:                 resp.CreatedAt,
		URL:                         resp.Source,
		ThumbnailURL:                resp.Livestream.Thumbnail,
		Viewable:                    resp.Status,
		ViewCount:                   int64(resp.Views),
		Language:                    "unknown",
		Type:                        "archive",
		Duration:                    time.Duration(durationSeconds),
		Category:                    &firstCategory,
		SpriteThumbnailsManifestUrl: nil,
	}

	return &info, nil
}

func (c *KickConnection) GetLiveStream(ctx context.Context, channelName string) (*LiveStreamInfo, error) {
	// Get channel for user id
	channelInfo, err := c.GetChannel(ctx, channelName)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel info: %w", err)
	}

	params := url.Values{
		"broadcaster_user_id": []string{channelInfo.ID},
	}

	body, err := c.kickMakeHTTPRequest(KickApiUrl, "GET", "livestreams", params, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get livestreams: %w", err)
	}

	var resp KickAPIResponse[KickLiveStream]
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal livestream response: %v", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no live stream found for channel: %s", channelName)
	}

	liveStream := resp.Data[0]

	startedAt, err := time.Parse(time.RFC3339, liveStream.StartedAt)
	if err != nil {
		return nil, err
	}

	// Get chatroom ID from the live stream
	chatRoom, err := c.GetChatRoom(ctx, channelInfo.Login)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat room: %w", err)
	}

	info := LiveStreamInfo{
		ID:           fmt.Sprintf("%d", liveStream.ChannelID),
		UserID:       fmt.Sprintf("%d", liveStream.BroadcasterUserID),
		ChatRoomID:   fmt.Sprintf("%d", chatRoom.ID),
		UserLogin:    channelInfo.Login,
		UserName:     channelInfo.DisplayName,
		Title:        liveStream.StreamTitle,
		GameID:       fmt.Sprintf("%d", liveStream.Category.ID),
		GameName:     liveStream.Category.Name,
		Type:         "live",
		StartedAt:    startedAt,
		ViewerCount:  int64(liveStream.ViewerCount),
		Language:     liveStream.Language,
		ThumbnailURL: liveStream.Thumbnail,
	}

	return &info, nil
}

func (c *KickConnection) GetLiveStreams(ctx context.Context, channelNames []string) ([]LiveStreamInfo, error) {
	return nil, ErrNotImplemented
}

func (c *KickConnection) GetChannel(ctx context.Context, channelName string) (*ChannelInfo, error) {
	params := url.Values{
		"slug": []string{channelName},
	}

	body, err := c.kickMakeHTTPRequest(KickApiUrl, "GET", "channels", params, nil)
	if err != nil {
		return nil, err
	}

	var resp KickAPIResponse[KickChannel]
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("channel not found: %s", channelName)
	}

	// Need to query the user as well to get additional information
	params = url.Values{
		"id": []string{fmt.Sprintf("%d", resp.Data[0].BroadcasterUserID)},
	}

	userBody, err := c.kickMakeHTTPRequest(KickApiUrl, "GET", "users", params, nil)
	if err != nil {
		return nil, err
	}

	var userResp KickAPIResponse[KickUser]
	err = json.Unmarshal(userBody, &userResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user response: %v", err)
	}
	if len(userResp.Data) == 0 {
		return nil, fmt.Errorf("user not found: %s", channelName)
	}

	info := ChannelInfo{
		ID:              fmt.Sprintf("%d", userResp.Data[0].UserID),
		Login:           userResp.Data[0].Name,
		DisplayName:     resp.Data[0].Slug,
		Type:            "kick",
		Description:     "unknown", // Kick API does not provide description
		BroadcasterType: "unknown", // Kick API does not provide broadcaster type
		ProfileImageURL: userResp.Data[0].ProfilePicture,
		OfflineImageURL: userResp.Data[0].ProfilePicture, // Kick API does not provide offline image URL
		ViewCount:       int64(resp.Data[0].Stream.ViewerCount),
		CreatedAt:       time.Now(), // Kick API does not provide created at, using current time
	}

	return &info, nil
}

func (c *KickConnection) GetVideos(ctx context.Context, channelId string, videoType VideoType, withChapters bool, withMutedSegments bool) ([]VideoInfo, error) {
	return nil, ErrNotImplemented
}

func (c *KickConnection) GetCategories(ctx context.Context) ([]Category, error) {
	return nil, ErrNotImplemented
}

func (c *KickConnection) GetGlobalBadges(ctx context.Context) ([]Badge, error) {
	return nil, ErrNotImplemented
}

func (c *KickConnection) GetChannelBadges(ctx context.Context, channelId string) ([]Badge, error) {
	return nil, ErrNotImplemented
}

func (c *KickConnection) GetGlobalEmotes(ctx context.Context) ([]Emote, error) {
	return nil, ErrNotImplemented
}

func (c *KickConnection) GetChannelEmotes(ctx context.Context, channelId string) ([]Emote, error) {
	return nil, ErrNotImplemented
}

func (c *KickConnection) GetChannelClips(ctx context.Context, channelId string, filter ClipsFilter) ([]ClipInfo, error) {
	return nil, ErrNotImplemented
}

func (c *KickConnection) GetClip(ctx context.Context, id string) (*ClipInfo, error) {
	return nil, ErrNotImplemented
}

func (c *KickConnection) CheckIfStreamIsLive(ctx context.Context, channelName string) (bool, error) {
	return true, ErrNotImplemented
}

func (c *KickConnection) GetStreams(ctx context.Context, limit int) ([]LiveStreamInfo, error) {
	return nil, ErrNotImplemented
}

func (c *KickConnection) DownloadVodChat(ctx context.Context, videoId string, startTime time.Time, endTime time.Time, outputPath string) error {
	// videoId is the live stream/chat room ID in the case of Kick
	formattedStartTime := fmt.Sprintf(
		"%sZ",
		url.QueryEscape(startTime.Format("2006-01-02T15:04:05.000")),
	)

	// Create/open the output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Write JSON array opening bracket
	_, err = file.WriteString("[\n")
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	cursor := formattedStartTime
	messageCount := 0

	for {
		params := url.Values{
			"start_time": []string{cursor},
		}

		body, err := c.kickMakeHTTPRequest(KickPrivateApiUrl, "GET", fmt.Sprintf("chat/%s/history", videoId), params, nil)
		if err != nil {
			return fmt.Errorf("failed to get vod chat: %w", err)
		}

		var resp KickOldAPIResponse[KickVodChatMesssageResponse]
		err = json.Unmarshal(body, &resp)
		if err != nil {
			return fmt.Errorf("failed to unmarshal vod chat response: %v", err)
		}

		if len(resp.Data.Messages) == 0 {
			// Check if we have a cursor for pagination - continue even with no messages
			if resp.Data.Cursor == "" {
				break // No more pages available
			}

			// Convert cursor and continue to next page
			cursorMicroseconds, err := strconv.ParseInt(resp.Data.Cursor, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse cursor: %w", err)
			}

			cursorTime := time.Unix(0, cursorMicroseconds*1000)
			if cursorTime.After(endTime) {
				break // We've passed the end time
			}

			cursor = fmt.Sprintf(
				"%sZ",
				url.QueryEscape(cursorTime.Format("2006-01-02T15:04:05.000")),
			)

			time.Sleep(100 * time.Millisecond)
			continue // Continue to next iteration without processing messages
		}

		// Process messages and write to file
		for _, message := range resp.Data.Messages {
			// Check if message is beyond end time
			if message.CreatedAt.After(endTime) {
				// Write closing bracket and return
				_, err = file.WriteString("\n]")
				if err != nil {
					return fmt.Errorf("failed to write closing bracket: %w", err)
				}
				return nil
			}

			// Add comma before message if not first message
			if messageCount > 0 {
				_, err = file.WriteString(",\n")
				if err != nil {
					return fmt.Errorf("failed to write comma: %w", err)
				}
			}

			// Marshal and write message
			messageJSON, err := json.Marshal(message)
			if err != nil {
				return fmt.Errorf("failed to marshal message: %w", err)
			}

			_, err = file.Write(messageJSON)
			if err != nil {
				return fmt.Errorf("failed to write message to file: %w", err)
			}

			messageCount++
		}

		// Check if we have a cursor for pagination
		if resp.Data.Cursor == "" {
			break // No more pages
		}

		// Convert cursor (microsecond epoch) to ISO8601 format
		cursorMicroseconds, err := strconv.ParseInt(resp.Data.Cursor, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse cursor: %w", err)
		}

		// Convert microseconds to time and format as ISO8601 with milliseconds
		cursorTime := time.Unix(0, cursorMicroseconds*1000) // Convert microseconds to nanoseconds
		cursor = fmt.Sprintf(
			"%sZ",
			url.QueryEscape(cursorTime.Format("2006-01-02T15:04:05.000")),
		)

		// Check if cursor time is beyond end time
		if cursorTime.After(endTime) {
			break
		}

		// Add a small delay to avoid hitting rate limits
		time.Sleep(100 * time.Millisecond)
	}

	// Write closing bracket
	_, err = file.WriteString("\n]")
	if err != nil {
		return fmt.Errorf("failed to write closing bracket: %w", err)
	}

	return nil
}
