package twitch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type GQLResponse struct {
	Data       Data       `json:"data"`
	Extensions Extensions `json:"extensions"`
}

type Data struct {
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

type GQLChapterResponse struct {
	Data       GQLChapterData `json:"data"`
	Extensions Extensions     `json:"extensions"`
}

type GQLChapterData struct {
	Video GQLChapterDataVideo `json:"video"`
}

type GQLChapterDataVideo struct {
	ID       string  `json:"id"`
	Moments  Moments `json:"moments"`
	Typename string  `json:"__typename"`
}

type Node struct {
	Moments              Moments   `json:"moments"`
	ID                   string    `json:"id"`
	DurationMilliseconds int64     `json:"durationMilliseconds"`
	PositionMilliseconds int64     `json:"positionMilliseconds"`
	Type                 Type      `json:"type"`
	Description          string    `json:"description"`
	SubDescription       string    `json:"subDescription"`
	ThumbnailURL         string    `json:"thumbnailURL"`
	Details              Details   `json:"details"`
	Video                NodeVideo `json:"video"`
	Typename             string    `json:"__typename"`
}

type Edge struct {
	Node     Node   `json:"node"`
	Typename string `json:"__typename"`
}

type Moments struct {
	Edges    []Edge `json:"edges"`
	Typename string `json:"__typename"`
}

type Details struct {
	Game     GameClass       `json:"game"`
	Typename DetailsTypename `json:"__typename"`
}

type GameClass struct {
	ID          string       `json:"id"`
	DisplayName string       `json:"displayName"`
	BoxArtURL   string       `json:"boxArtURL"`
	Typename    GameTypename `json:"__typename"`
}

type NodeVideo struct {
	ID            string `json:"id"`
	LengthSeconds int64  `json:"lengthSeconds"`
	Typename      string `json:"__typename"`
}

type GameTypename string

const (
	Game GameTypename = "Game"
)

type DetailsTypename string

const (
	GameChangeMomentDetails DetailsTypename = "GameChangeMomentDetails"
)

func gqlRequest(body string) (GQLResponse, error) {
	var response GQLResponse

	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://gql.twitch.tv/gql", strings.NewReader(body))
	if err != nil {
		return response, err
	}
	req.Header.Set("Client-ID", "kimne78kx3ncx6brgo4mv6wki5h1ko")
	req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	req.Header.Set("Origin", "https://www.twitch.tv")
	req.Header.Set("Referer", "https://www.twitch.tv/")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return response, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("error reading response body: %w", err)
	}

	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		return response, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return response, nil

}

func GQLGetVideo(id string) (GQLResponse, error) {
	body := fmt.Sprintf(`{"query": "query{video(id:%s){broadcastType,resourceRestriction{id,type},game{id,name},title,createdAt}}"}`, id)
	resp, err := gqlRequest(body)
	if err != nil {
		return resp, fmt.Errorf("error getting video: %w", err)
	}

	return resp, nil
}

func gqlChapterRequest(body string) (GQLChapterResponse, error) {
	var response GQLChapterResponse

	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://gql.twitch.tv/gql", strings.NewReader(body))
	if err != nil {
		return response, err
	}
	req.Header.Set("Client-ID", "kimne78kx3ncx6brgo4mv6wki5h1ko")
	req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	req.Header.Set("Origin", "https://www.twitch.tv")
	req.Header.Set("Referer", "https://www.twitch.tv/")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return response, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("error reading response body: %w", err)
	}

	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		return response, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return response, nil

}

func GQLGetChapters(id string) (GQLChapterResponse, error) {
	body := fmt.Sprintf(`{"operationName":"VideoPlayer_ChapterSelectButtonVideo","variables":{"videoID":"%s","includePrivate":false},"extensions":{"persistedQuery":{"version":1,"sha256Hash":"8d2793384aac3773beab5e59bd5d6f585aedb923d292800119e03d40cd0f9b41"}}}`, id)
	resp, err := gqlChapterRequest(body)
	if err != nil {
		return resp, fmt.Errorf("error getting video chapters: %w", err)
	}

	return resp, nil
}
