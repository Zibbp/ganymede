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
		{"1080p60", []string{"audio", "160p", "360p", "720p", "1080p60", "480p", "720p60", "worst", "best", "audio_only"}, "1080p60"},
		{"1080p", []string{"audio", "160p", "360p", "720p", "1080p30", "480p", "720p60", "worst", "best", "audio_only"}, "1080p30"},
		{"1080p", []string{"audio", "160p", "360p", "720p", "1080p", "480p", "720p60", "worst", "best", "audio_only"}, "1080p"},
		{"1080p30", []string{"audio", "160p", "360p", "720p", "1080p30", "1080p60", "480p", "720p60", "worst", "best", "audio_only"}, "1080p30"},
		{"1080", []string{"audio", "160p", "360p", "720p", "1080p30", "1080p60", "480p", "720p60", "worst", "best", "audio_only"}, "1080p60"},
		{"1080p", []string{"audio", "160p", "360p", "720p", "1080p30", "1080p60", "480p", "720p60", "worst", "best", "audio_only"}, "1080p60"},
		{"1080p", []string{"audio", "160p", "360p", "720p", "1080p", "480p", "720p60", "worst", "best", "audio_only"}, "1080p"},
		{"1080", []string{"audio", "160p", "360p", "720p", "1080p", "480p", "720p60", "worst", "best", "audio_only"}, "1080p"},
		{"720p", []string{"audio", "160p", "360p", "720p", "1080p60", "480p", "720p60", "worst", "best", "audio_only"}, "720p60"},
		{"720", []string{"audio", "160p", "360p", "720p", "1080p60", "480p", "720p60", "worst", "best", "audio_only"}, "720p60"},
		{"720p30", []string{"audio", "160p", "360p", "720p", "1080p60", "480p", "720p60", "worst", "best", "audio_only"}, "720p"},
		{"720p29", []string{"audio", "160p", "360p", "720p", "1080p60", "480p", "720p29", "worst", "best", "audio_only"}, "720p29"},
		{"480p", []string{"audio", "160p", "360p", "720p", "1080p60", "480p", "720p60", "worst", "best", "audio_only"}, "480p"},
		{"480p", []string{"audio", "160p", "360p", "720p", "1080p60", "480p30", "720p60", "worst", "best", "audio_only"}, "480p30"},
		{"480p30", []string{"audio", "160p", "360p", "720p", "1080p60", "480p", "720p60", "worst", "best", "audio_only"}, "480p"},
		{"240p", []string{"audio", "160p", "360p", "720p", "1080p60", "480p", "720p60", "worst", "best", "audio_only"}, "best"},
		{"best", []string{"audio", "160p", "360p", "720p", "1080p60", "480p", "720p60", "worst", "best", "audio_only"}, "best"},
		{"audio_only", []string{"audio", "160p", "360p", "720p", "1080p60", "480p", "720p60", "worst", "best", "audio_only"}, "audio_only"},
		{"500p", []string{"audio", "160p", "360p", "720p", "1080p60", "480p", "720p60", "worst", "best", "audio_only"}, "best"},
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
