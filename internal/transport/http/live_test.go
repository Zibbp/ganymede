package http

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/internal/live"
)

func TestWatchedChannelClipResolutionValidation(t *testing.T) {
	t.Parallel()

	validate := validator.New()

	t.Run("audio clip quality rejected by request validation", func(t *testing.T) {
		t.Parallel()
		err := validate.Struct(AddWatchedChannelRequest{
			ChannelID:      "00000000-0000-0000-0000-000000000000",
			Resolution:     "audio",
			VodResolution:  "audio",
			ClipResolution: "audio",
		})
		assert.Error(t, err, "explicit audio clip_resolution must fail validation")
	})

	t.Run("video clip qualities accepted by request validation", func(t *testing.T) {
		t.Parallel()
		for _, quality := range []string{"best", "1080p", "720p", "480p", "360p", "160p", "1440p", ""} {
			err := validate.Struct(UpdateWatchedChannelRequest{
				Resolution:        "audio",
				VodResolution:     "audio",
				ClipResolution:    quality,
				ClipsLimit:        1,
				ClipsIntervalDays: 1,
			})
			assert.NoError(t, err, "clip_resolution %q should pass validation", quality)
		}
	})

	t.Run("effective audio via fallback rejected", func(t *testing.T) {
		t.Parallel()
		// clip omitted + VOD audio resolves to audio through the
		// clip -> VOD -> live fallback chain.
		effective := live.ResolveClipResolution("", "audio", "best")
		assert.Error(t, live.ValidateClipResolution(effective))
	})

	t.Run("explicit clip quality overrides audio VOD", func(t *testing.T) {
		t.Parallel()
		effective := live.ResolveClipResolution("360p", "audio", "audio")
		require.Equal(t, "360p", effective)
		assert.NoError(t, live.ValidateClipResolution(effective))
	})
}
