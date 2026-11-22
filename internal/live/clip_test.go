package live

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClipsWindowStart(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)

	t.Run("interval zero checks all clips", func(t *testing.T) {
		assert.True(t, clipsWindowStart(now, 0, false, now.AddDate(0, 0, -7)).IsZero())
	})

	t.Run("first check uses the fixed lookback", func(t *testing.T) {
		assert.Equal(t, now.AddDate(0, 0, -7), clipsWindowStart(now, 7, false, time.Time{}))
	})

	t.Run("ignore last checked uses the fixed lookback", func(t *testing.T) {
		assert.Equal(t, now.AddDate(0, 0, -7), clipsWindowStart(now, 7, true, now.AddDate(0, 0, -30)))
	})

	t.Run("steady state matches the fixed lookback", func(t *testing.T) {
		// The check gate requires the interval to have passed, so the last
		// check is at least as old as the fixed lookback.
		lastChecked := now.AddDate(0, 0, -7)
		assert.Equal(t, lastChecked, clipsWindowStart(now, 7, false, lastChecked))
	})

	t.Run("deferred retry keeps old clips in the window", func(t *testing.T) {
		// A clip created just inside the old window edge must stay in the
		// window on a retry days later.
		lastChecked := now.AddDate(0, 0, -10)
		assert.Equal(t, lastChecked, clipsWindowStart(now, 7, false, lastChecked))
	})
}
