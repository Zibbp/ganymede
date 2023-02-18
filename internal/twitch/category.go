package twitch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/rs/zerolog/log"
	entTwitchCategory "github.com/zibbp/ganymede/ent/twitchcategory"
	"github.com/zibbp/ganymede/internal/database"
)

type CategoryResponse struct {
	Data       []TwitchCategory `json:"data"`
	Pagination Pagination       `json:"pagination"`
}

type TwitchCategory struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	BoxArtURL string `json:"box_art_url"`
	IgdbID    string `json:"igdb_id"`
}

// SetTwitchCategories sets the twitch categories in the database
func SetTwitchCategories() error {
	categories, err := GetCategories()
	if err != nil {
		return fmt.Errorf("failed to get twitch categories: %v", err)
	}

	for _, category := range categories {
		err = database.DB().Client.TwitchCategory.Create().SetID(category.ID).SetName(category.Name).SetBoxArtURL(category.BoxArtURL).SetIgdbID(category.IgdbID).OnConflictColumns(entTwitchCategory.FieldID).UpdateNewValues().Exec(context.Background())
		if err != nil {
			return fmt.Errorf("failed to upsert twitch category: %v", err)
		}
	}

	log.Debug().Msgf("successfully set twitch categories")

	return nil
}

// GetCategories gets the top 100 twitch categories
// It then gets the next 100 categories until there are no more using the cursor
// Returns a different number of categories each time it is called for some reason
func GetCategories() ([]TwitchCategory, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.twitch.tv/helix/games/top?first=100", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Client-ID", os.Getenv("TWITCH_CLIENT_ID"))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("TWITCH_ACCESS_TOKEN")))
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get twitch categories: %v", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Error().Err(err).Msgf("failed to get twitch categories: %v", string(body))
		return nil, fmt.Errorf("failed to get twitch categories: %v", resp)
	}

	var categoryResponse CategoryResponse
	err = json.Unmarshal(body, &categoryResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	var twitchCategories []TwitchCategory
	twitchCategories = append(twitchCategories, categoryResponse.Data...)

	// pagination
	var cursor string
	cursor = categoryResponse.Pagination.Cursor
	for cursor != "" {
		response, err := getCategoriesWithCursor(cursor)
		if err != nil {
			return nil, fmt.Errorf("failed to get twitch categories: %v", err)
		}
		twitchCategories = append(twitchCategories, response.Data...)
		cursor = response.Pagination.Cursor
	}

	return twitchCategories, nil
}

func getCategoriesWithCursor(cursor string) (*CategoryResponse, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.twitch.tv/helix/games/top?first=100&after=%s", cursor), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Client-ID", os.Getenv("TWITCH_CLIENT_ID"))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("TWITCH_ACCESS_TOKEN")))
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get twitch categories: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get twitch categories: %v", resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var categoryResponse CategoryResponse
	err = json.Unmarshal(body, &categoryResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &categoryResponse, nil

}
