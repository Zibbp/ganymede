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

func TestClipsDeferralAnchor(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)

	t.Run("first check anchors the retry to the window start", func(t *testing.T) {
		startedAt := now.AddDate(0, 0, -7)
		anchor, ok := clipsDeferralAnchor(time.Time{}, startedAt)
		assert.True(t, ok)
		assert.Equal(t, startedAt, anchor)
		// The retry window then covers the whole deferred window.
		assert.Equal(t, startedAt, clipsWindowStart(now, 7, false, anchor))
	})

	t.Run("already checked keeps its timestamp", func(t *testing.T) {
		_, ok := clipsDeferralAnchor(now.AddDate(0, 0, -3), now.AddDate(0, 0, -7))
		assert.False(t, ok)
	})

	t.Run("zero interval has no anchor", func(t *testing.T) {
		_, ok := clipsDeferralAnchor(time.Time{}, time.Time{})
		assert.False(t, ok)
	})
}
