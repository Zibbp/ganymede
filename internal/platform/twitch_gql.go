package platform

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/chapter"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/utils"
)

type TwitchGQLPlaybackAccessTokenResponse struct {
	Data   TwitchGQLPlaybackAccessTokenData `json:"data"`
	Errors []TwitchGQLError                 `json:"errors"`
}

type TwitchGQLError struct {
	Message string `json:"message"`
}

type TwitchGQLPlaybackAccessTokenData struct {
	StreamPlaybackAccessToken TwitchGQLPlaybackAccessToken `json:"streamPlaybackAccessToken"`
}

type TwitchGQLPlaybackAccessToken struct {
	Value     string `json:"value"`
	Signature string `json:"signature"`
}

type TwitchGQLVideoResponse struct {
	Data       TwitchGQLVideoData `json:"data"`
	Extensions TwitchExtensions   `json:"extensions"`
}

type TwitchGQLVideoData struct {
	Video TwitchGQLVideo `json:"video"`
}

type TwitchGQLVideo struct {
	BroadcastType       string                    `json:"broadcastType"`
	ResourceRestriction TwitchResourceRestriction `json:"resourceRestriction"`
	Game                TwitchGQLGame             `json:"game"`
	Title               string                    `json:"title"`
	CreatedAt           string                    `json:"createdAt"`
	SeekPreviewsURL     string                    `json:"seekPreviewsURL"` // storyboard thumbnails manifest
}

type TwitchGQLGame struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TwitchResourceRestriction struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type TwitchExtensions struct {
	DurationMilliseconds int64  `json:"durationMilliseconds"`
	RequestID            string `json:"requestID"`
}

type TwitchGQLMutedSegmentsResponse struct {
	Data       TwitchGQLMutedSegmentsData `json:"data"`
	Extensions TwitchExtensions           `json:"extensions"`
}

type TwitchGQLMutedSegmentsData struct {
	Video TwitchGQLMutedSegmentsVideo `json:"video"`
}

type TwitchGQLMutedSegmentsVideo struct {
	ID       string         `json:"id"`
	MuteInfo TwitchMuteInfo `json:"muteInfo"`
}

type TwitchMuteInfo struct {
	MutedSegmentConnection TwitchGQLMutedSegmentConnection `json:"mutedSegmentConnection"`
	TypeName               string                          `json:"__typename"`
}

type TwitchGQLMutedSegmentConnection struct {
	Nodes []TwitchGQLMutedSegment `json:"nodes"`
}

type TwitchGQLMutedSegment struct {
	Duration int    `json:"duration"`
	Offset   int    `json:"offset"`
	TypeName string `json:"__typename"`
}

type TwitchGQLChaptersResponse struct {
	Data       TwitchGQLChaptersData `json:"data"`
	Extensions TwitchExtensions      `json:"extensions"`
}

type TwitchGQLChaptersData struct {
	Video TwitchGQLChaptersVideo `json:"video"`
}

type TwitchGQLChaptersVideo struct {
	ID       string           `json:"id"`
	Moments  TwitchGQLMoments `json:"moments"`
	Typename string           `json:"__typename"`
}

type TwitchGQLChapter struct {
	Moments              TwitchGQLMoments   `json:"moments"`
	ID                   string             `json:"id"`
	DurationMilliseconds int64              `json:"durationMilliseconds"`
	PositionMilliseconds int64              `json:"positionMilliseconds"`
	Type                 string             `json:"type"`
	Description          string             `json:"description"`
	SubDescription       string             `json:"subDescription"`
	ThumbnailURL         string             `json:"thumbnailURL"`
	Details              TwitchGQLDetails   `json:"details"`
	Video                TwitchGQLNodeVideo `json:"video"`
	Typename             string             `json:"__typename"`
}

type TwitchGQLChapterEdge struct {
	Node     TwitchGQLChapter `json:"node"`
	Typename string           `json:"__typename"`
}

type TwitchGQLMoments struct {
	Edges    []TwitchGQLChapterEdge `json:"edges"`
	Typename string                 `json:"__typename"`
}

type TwitchGQLDetails struct {
	Game     TwitchGQLGameInfo `json:"game"`
	Typename string            `json:"__typename"`
}

type TwitchGQLGameInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	BoxArtURL   string `json:"boxArtURL"`
	Typename    string `json:"__typename"`
}

type TwitchGQLNodeVideo struct {
	ID            string `json:"id"`
	LengthSeconds int64  `json:"lengthSeconds"`
	Typename      string `json:"__typename"`
}

// GQLRequest sends a generic GQL request and returns the response.
func twitchGQLRequest(body string) ([]byte, error) {
	return twitchGQLRequestWithAuth(body, true)
}

// twitchGQLRequestWithAuth sends a generic GQL request and optionally includes the configured Twitch OAuth token.
func twitchGQLRequestWithAuth(body string, includeAuth bool) ([]byte, error) {
	client := &http.Client{}
	const maxAttempts = 3

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequest("POST", "https://gql.twitch.tv/gql", strings.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}

		req.Header.Set("Client-ID", "kimne78kx3ncx6brgo4mv6wki5h1ko")
		req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
		req.Header.Set("Origin", "https://www.twitch.tv")
		req.Header.Set("Referer", "https://www.twitch.tv/")
		req.Header.Set("Sec-Fetch-Mode", "cors")
		req.Header.Set("Sec-Fetch-Site", "same-site")
		req.Header.Set("User-Agent", utils.ChromeUserAgent)

		twitchToken := config.Get().Parameters.TwitchToken
		if includeAuth && twitchToken != "" {
			req.Header.Set("Authorization", fmt.Sprintf("OAuth %s", twitchToken))
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("error sending request: %w", err)
		} else {
			bodyBytes, readErr := io.ReadAll(resp.Body)
			if closeErr := resp.Body.Close(); closeErr != nil {
				log.Debug().Err(closeErr).Msg("error closing response body")
			}

			if readErr != nil {
				lastErr = fmt.Errorf("error reading response body: %w", readErr)
			} else if resp.StatusCode >= http.StatusInternalServerError || resp.StatusCode == http.StatusTooManyRequests {
				lastErr = fmt.Errorf("received retryable status code: %d", resp.StatusCode)
			} else {
				return bodyBytes, nil
			}
		}

		if attempt < maxAttempts {
			time.Sleep(time.Duration(attempt) * 300 * time.Millisecond)
		}
	}

	return nil, lastErr
}

func parsePlaybackAccessTokenResponse(respBytes []byte) (*TwitchGQLPlaybackAccessToken, error) {
	var resp TwitchGQLPlaybackAccessTokenResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("error unmarshalling playback access token response: %w", err)
	}

	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("gql playback access token error: %s", resp.Errors[0].Message)
	}

	if resp.Data.StreamPlaybackAccessToken.Signature == "" || resp.Data.StreamPlaybackAccessToken.Value == "" {
		return nil, fmt.Errorf("empty playback access token response")
	}

	return &resp.Data.StreamPlaybackAccessToken, nil
}

func (c *TwitchConnection) TwitchGQLGetMutedSegments(id string) ([]TwitchGQLMutedSegment, error) {
	body := fmt.Sprintf(`{"operationName":"VideoPlayer_MutedSegmentsAlertOverlay","variables":{"vodID":"%s","includePrivate":false},"extensions":{"persistedQuery":{"version":1,"sha256Hash":"c36e7400657815f4704e6063d265dff766ed8fc1590361c6d71e4368805e0b49"}}}`, id)
	respBytes, err := twitchGQLRequest(body)
	if err != nil {
		return nil, fmt.Errorf("error getting video muted segments: %w", err)
	}

	var resp TwitchGQLMutedSegmentsResponse
	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return resp.Data.Video.MuteInfo.MutedSegmentConnection.Nodes, nil
}

func (c *TwitchConnection) TwitchGQLGetVideo(id string) (*TwitchGQLVideo, error) {
	body := fmt.Sprintf(`{"query": "query{video(id:\"%s\"){broadcastType,resourceRestriction{id,type},game{id,name},title,createdAt,seekPreviewsURL}}"}`, id)
	respBytes, err := twitchGQLRequest(body)
	if err != nil {
		return nil, fmt.Errorf("error getting video muted segments: %w", err)
	}

	var resp TwitchGQLVideoResponse
	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return &resp.Data.Video, nil
}

func (c *TwitchConnection) TwitchGQLGetChapters(id string) ([]TwitchGQLChapterEdge, error) {
	body := fmt.Sprintf(`{"operationName":"VideoPlayer_ChapterSelectButtonVideo","variables":{"videoID":"%s","includePrivate":false},"extensions":{"persistedQuery":{"version":1,"sha256Hash":"8d2793384aac3773beab5e59bd5d6f585aedb923d292800119e03d40cd0f9b41"}}}`, id)
	respBytes, err := twitchGQLRequest(body)
	if err != nil {
		return nil, fmt.Errorf("error getting video chapters: %w", err)
	}

	var resp TwitchGQLChaptersResponse
	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return resp.Data.Video.Moments.Edges, nil
}

// TwitchGQLGetPlaybackAccessToken retrieves the playback access token for a live stream.
func (c *TwitchConnection) TwitchGQLGetPlaybackAccessToken(channel string) (*TwitchGQLPlaybackAccessToken, error) {
	// TODO: remove hardcoded options in the future if used for other functions
	body := fmt.Sprintf(`{
		"operationName": "PlaybackAccessToken",
		"variables": {
			"isLive": true,
			"login": "%s",
			"isVod": false,
			"vodID": "",
			"playerType": "site"
		},
		"query": "query PlaybackAccessToken($isLive: Boolean!, $login: String!, $isVod: Boolean!, $vodID: ID!, $playerType: String!) {\nstreamPlaybackAccessToken(channelName: $login, params: {platform: \"web\", playerBackend: \"mediaplayer\", playerType: $playerType}) @include(if: $isLive) {\nvalue\nsignature\n}\nvideoPlaybackAccessToken(id: $vodID, params: {platform: \"web\", playerBackend: \"mediaplayer\", playerType: $playerType}) @include(if: $isVod) {\nvalue\nsignature\n}\n}"
	}`, channel)

	twitchToken := config.Get().Parameters.TwitchToken
	if twitchToken != "" {
		respBytes, err := twitchGQLRequestWithAuth(body, true)
		if err != nil {
			log.Warn().Err(err).Msg("failed to get playback access token with Twitch OAuth token, retrying without token")
		} else {
			token, parseErr := parsePlaybackAccessTokenResponse(respBytes)
			if parseErr == nil {
				return token, nil
			}
			log.Warn().Err(parseErr).Msg("configured Twitch OAuth token appears invalid for playback access, retrying without token")
		}
	}

	respBytes, err := twitchGQLRequestWithAuth(body, false)
	if err != nil {
		return nil, fmt.Errorf("error getting playback access token without oauth token: %w", err)
	}

	token, err := parsePlaybackAccessTokenResponse(respBytes)
	if err != nil {
		return nil, err
	}

	return token, nil
}

// convertTwitchChaptersToChapters converts Twitch chapters to chapters. Twitch chapters are in milliseconds.
func convertTwitchChaptersToChapters(chapters []TwitchGQLChapterEdge, duration int) ([]chapter.Chapter, error) {
	if len(chapters) == 0 {
		return []chapter.Chapter{}, nil
	}

	convertedChapters := make([]chapter.Chapter, len(chapters))
	for i := 0; i < len(chapters); i++ {
		convertedChapters[i].ID = chapters[i].Node.ID
		convertedChapters[i].Title = chapters[i].Node.Description
		convertedChapters[i].Type = string(chapters[i].Node.Type)
		convertedChapters[i].Start = int(chapters[i].Node.PositionMilliseconds / 1000)

		if i+1 < len(chapters) {
			convertedChapters[i].End = int(chapters[i+1].Node.PositionMilliseconds / 1000)
		} else {
			convertedChapters[i].End = duration
		}
	}

	return convertedChapters, nil
}
