package utils

import (
	"fmt"
	"testing"
)

// TestSelectClosestQuality validates the logic for selecting the closest quality.
func TestSelectClosestQuality(t *testing.T) {
	tests := []struct {
		target   string
		options  []string
		expected string
	}{
		{"best", []string{"chunked", "1080p60", "360p30", "480p30", "720p60", "audio_only"}, "chunked"},
		{"1440p", []string{"chunked", "360p30", "480p30", "720p60", "audio_only"}, "chunked"},
		{"1080p", []string{"chunked", "360p30", "480p30", "720p60", "audio_only"}, "chunked"},
		{"1080p", []string{"chunked", "1080p60", "360p30", "480p30", "720p60", "audio_only"}, "1080p60"},
		{"1080p60", []string{"chunked", "1080p60", "360p30", "480p30", "720p60", "audio_only"}, "1080p60"},
		{"1080p30", []string{"chunked", "1080p60", "360p30", "480p30", "720p60", "audio_only"}, "1080p60"},
		{"720p", []string{"chunked", "1080p60", "360p30", "480p30", "720p60", "audio_only"}, "720p60"},
		{"best", []string{"chunked", "1080p60", "360p30", "480p30", "720p60", "audio_only"}, "chunked"},
		{"audio_only", []string{"chunked", "1080p60", "360p30", "480p30", "720p60", "audio_only"}, "audio_only"},
		{"500p", []string{"chunked", "1080p60", "360p30", "480p30", "720p60", "audio_only"}, "chunked"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Target: %s", test.target), func(t *testing.T) {
			result := SelectClosestQuality(test.target, test.options)
			if result != test.expected {
				t.Errorf("For target %s, expected %s but got %s", test.target, test.expected, result)
			}
		})
	}
}

// TestParseQuality validates the parsing of quality strings into structured data.
func TestParseQuality(t *testing.T) {
	tests := []struct {
		input      string
		wantRes    int
		wantFPS    int
		wantOrigin string
	}{
		{"1440p", 1440, 60, "1440p"},
		{"1440p30", 1440, 30, "1440p30"},
		{"1080p60", 1080, 60, "1080p60"},
		{"720p", 720, 60, "720p"},
		{"480p30", 480, 30, "480p30"},
		{"360p", 360, 60, "360p"},
		{"720", 720, 60, "720"},
		{"best", 0, 0, "best"},
		{"audio_only", 0, 0, "audio_only"},
		{"1080p", 1080, 60, "1080p"},
		{"", 0, 0, ""},
		{"999p120", 999, 120, "999p120"},
		{"240", 240, 60, "240"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			q := parseQuality(tt.input)
			if q.Resolution != tt.wantRes || q.FPS != tt.wantFPS || q.Original != tt.wantOrigin {
				t.Errorf("parseQuality(%q) = {Resolution:%d, FPS:%d, Original:%q}, want {Resolution:%d, FPS:%d, Original:%q}",
					tt.input, q.Resolution, q.FPS, q.Original, tt.wantRes, tt.wantFPS, tt.wantOrigin)
			}
		})
	}
}
