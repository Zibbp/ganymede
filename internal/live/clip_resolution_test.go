package live

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/ent"
)

func TestResolveClipResolution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		clip     string
		vod      string
		live     string
		expected string
	}{
		{"explicit clip wins", "720p", "audio", "best", "720p"},
		{"falls back to vod", "", "720p", "best", "720p"},
		{"falls back to live", "", "", "720p", "720p"},
		{"audio vod preserved for validation", "", "audio", "best", "audio"},
		{"all empty", "", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, ResolveClipResolution(tt.clip, tt.vod, tt.live))
		})
	}
}

func TestResolveClipResolutionForLive(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "720p", ResolveClipResolutionForLive(&ent.Live{
		ClipResolution: "720p",
		VodResolution:  "audio",
		Resolution:     "best",
	}))
	assert.Equal(t, "1080p", ResolveClipResolutionForLive(&ent.Live{
		VodResolution: "1080p",
		Resolution:    "best",
	}))
	assert.Equal(t, "best", ResolveClipResolutionForLive(&ent.Live{
		Resolution: "best",
	}))
	assert.Equal(t, "", ResolveClipResolutionForLive(nil))
}

func TestValidateClipResolution(t *testing.T) {
	t.Parallel()

	for _, quality := range []string{"best", "1080p", "720p", "480p", "360p", "160p", "1440p", ""} {
		assert.NoError(t, ValidateClipResolution(quality), "quality %q should be allowed for clips", quality)
	}
	assert.Error(t, ValidateClipResolution("audio"), "audio must be rejected for clips")
}
