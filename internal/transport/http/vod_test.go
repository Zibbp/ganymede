package http

import (
	"strings"
	"testing"
)

func TestGenerateThumbnailsVTTEscapesImagePath(t *testing.T) {
	t.Setenv("TWITCH_CLIENT_ID", "test")
	t.Setenv("TWITCH_CLIENT_SECRET", "test")
	t.Setenv("CDN_URL", "https://cdn.example.com")

	metadata := SpriteMetadata{
		Duration:       60,
		SpriteImages:   []string{"/videos/channel/my folder/sprite#001.jpg"},
		SpriteInterval: 60,
		SpriteRows:     1,
		SpriteColumns:  1,
		SpriteHeight:   124,
		SpriteWidth:    220,
	}

	vtt, err := GenerateThumbnailsVTT(metadata)
	if err != nil {
		t.Fatalf("GenerateThumbnailsVTT() unexpected error: %v", err)
	}

	expectedURL := "https://cdn.example.com/videos/channel/my%20folder/sprite%23001.jpg#xywh=0,0,220,124"
	if !strings.Contains(vtt, expectedURL) {
		t.Fatalf("expected escaped URL in VTT.\nexpected contains: %q\nactual: %q", expectedURL, vtt)
	}
}
