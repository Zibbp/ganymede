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
	Title               string              `json:"title"`
	CreatedAt           string              `json:"createdAt"`
}

type ResourceRestriction struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type Extensions struct {
	DurationMilliseconds int64  `json:"durationMilliseconds"`
	RequestID            string `json:"requestID"`
}

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
	body := fmt.Sprintf(`{"query": "query{video(id:%s){broadcastType,resourceRestriction{id,type},title,createdAt}}"}`, id)
	resp, err := gqlRequest(body)
	if err != nil {
		return resp, fmt.Errorf("error getting video: %w", err)
	}

	return resp, nil
}
