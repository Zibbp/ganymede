package twitch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

type GQLVideoResponse struct {
	Data       GQLVideoData `json:"data"`
	Extensions Extensions   `json:"extensions"`
}

type GQLVideoData struct {
	Video GQLVideo `json:"video"`
}

type GQLVideo struct {
	BroadcastType       string              `json:"broadcastType"`
	ResourceRestriction ResourceRestriction `json:"resourceRestriction"`
	Game                GQLGame             `json:"game"`
	Title               string              `json:"title"`
	CreatedAt           string              `json:"createdAt"`
}

type GQLGame struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ResourceRestriction struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type Extensions struct {
	DurationMilliseconds int64  `json:"durationMilliseconds"`
	RequestID            string `json:"requestID"`
}

type GQLMutedSegmentsResponse struct {
	Data       GQLMutedSegmentsData `json:"data"`
	Extensions Extensions           `json:"extensions"`
}

type GQLMutedSegmentsData struct {
	Video GQLMutedSegmentsVideo `json:"video"`
}

type GQLMutedSegmentsVideo struct {
	ID       string   `json:"id"`
	MuteInfo MuteInfo `json:"muteInfo"`
}

type MuteInfo struct {
	MutedSegmentConnection GQLMutedSegmentConnection `json:"mutedSegmentConnection"`
	TypeName               string                    `json:"__typename"`
}

type GQLMutedSegmentConnection struct {
	Nodes []GQLMutedSegment `json:"nodes"`
}

type GQLMutedSegment struct {
	Duration int    `json:"duration"`
	Offset   int    `json:"offset"`
	TypeName string `json:"__typename"`
}

type GQLChaptersResponse struct {
	Data       GQLChaptersData `json:"data"`
	Extensions Extensions      `json:"extensions"`
}

type GQLChaptersData struct {
	Video GQLChaptersVideo `json:"video"`
}

type GQLChaptersVideo struct {
	ID       string     `json:"id"`
	Moments  GQLMoments `json:"moments"`
	Typename string     `json:"__typename"`
}

type GQLChapter struct {
	Moments              GQLMoments   `json:"moments"`
	ID                   string       `json:"id"`
	DurationMilliseconds int64        `json:"durationMilliseconds"`
	PositionMilliseconds int64        `json:"positionMilliseconds"`
	Type                 string       `json:"type"`
	Description          string       `json:"description"`
	SubDescription       string       `json:"subDescription"`
	ThumbnailURL         string       `json:"thumbnailURL"`
	Details              GQLDetails   `json:"details"`
	Video                GQLNodeVideo `json:"video"`
	Typename             string       `json:"__typename"`
}

type GQLChapterEdge struct {
	Node     GQLChapter `json:"node"`
	Typename string     `json:"__typename"`
}

type GQLMoments struct {
	Edges    []GQLChapterEdge `json:"edges"`
	Typename string           `json:"__typename"`
}

type GQLDetails struct {
	Game     GQLGameInfo `json:"game"`
	Typename string      `json:"__typename"`
}

type GQLGameInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	BoxArtURL   string `json:"boxArtURL"`
	Typename    string `json:"__typename"`
}

type GQLNodeVideo struct {
	ID            string `json:"id"`
	LengthSeconds int64  `json:"lengthSeconds"`
	Typename      string `json:"__typename"`
}

// GQLRequest sends a generic GQL request and returns the response.
func gqlRequest(body string) ([]byte, error) {
	client := &http.Client{}
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
	// req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Debug().Err(err).Msg("error closing response body")
		}
	}()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return bodyBytes, nil
}

// GQLGetVideo returns the GraphQL version of the video. This often contains data not available in the public API.
func GQLGetVideo(id string) (GQLVideo, error) {
	body := fmt.Sprintf(`{"query": "query{video(id:%s){broadcastType,resourceRestriction{id,type},game{id,name},title,createdAt}}"}`, id)
	respBytes, err := gqlRequest(body)
	if err != nil {
		return GQLVideo{}, fmt.Errorf("error getting video: %w", err)
	}

	var resp GQLVideoResponse
	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		return GQLVideo{}, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return resp.Data.Video, nil
}

func GQLGetMutedSegments(id string) ([]GQLMutedSegment, error) {
	body := fmt.Sprintf(`{"operationName":"VideoPlayer_MutedSegmentsAlertOverlay","variables":{"vodID":"%s","includePrivate":false},"extensions":{"persistedQuery":{"version":1,"sha256Hash":"c36e7400657815f4704e6063d265dff766ed8fc1590361c6d71e4368805e0b49"}}}`, id)
	respBytes, err := gqlRequest(body)
	if err != nil {
		return nil, fmt.Errorf("error getting video muted segments: %w", err)
	}

	var resp GQLMutedSegmentsResponse
	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return resp.Data.Video.MuteInfo.MutedSegmentConnection.Nodes, nil
}

func GQLGetChapters(id string) ([]GQLChapterEdge, error) {
	body := fmt.Sprintf(`{"operationName":"VideoPlayer_ChapterSelectButtonVideo","variables":{"videoID":"%s","includePrivate":false},"extensions":{"persistedQuery":{"version":1,"sha256Hash":"8d2793384aac3773beab5e59bd5d6f585aedb923d292800119e03d40cd0f9b41"}}}`, id)
	respBytes, err := gqlRequest(body)
	if err != nil {
		return nil, fmt.Errorf("error getting video chapters: %w", err)
	}

	var resp GQLChaptersResponse
	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return resp.Data.Video.Moments.Edges, nil
}
