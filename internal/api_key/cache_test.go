package api_key

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestVerificationCache_ShouldTouch covers the in-memory debounce that the
// HTTP middleware uses to skip the goroutine + SQL round-trip on hot keys.
// We drive the clock manually via the unexported `now` field so the test
// is deterministic and doesn't sleep.
func TestVerificationCache_ShouldTouch(t *testing.T) {
	c := NewVerificationCache(60 * time.Second)

	// Controlled clock so we can step through the debounce window.
	base := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return base }

	id := uuid.New()
	other := uuid.New()
	debounce := 60 * time.Second

	// First call: nothing recorded yet → should touch.
	assert.True(t, c.ShouldTouch(id, debounce), "first call must allow touch")

	// Second call within the window: should NOT touch.
	assert.False(t, c.ShouldTouch(id, debounce), "second call within window must skip")

	// Different id has its own debounce slot.
	assert.True(t, c.ShouldTouch(other, debounce), "independent id must not be blocked")

	// Step the clock just before the window expires: still suppressed.
	c.now = func() time.Time { return base.Add(59 * time.Second) }
	assert.False(t, c.ShouldTouch(id, debounce), "still inside window")

	// Step past the window: allowed again.
	c.now = func() time.Time { return base.Add(60*time.Second + time.Millisecond) }
	assert.True(t, c.ShouldTouch(id, debounce), "after window must allow touch")

	// And the previous successful touch resets the window.
	assert.False(t, c.ShouldTouch(id, debounce), "fresh window after re-touch")
}

// TestVerificationCache_InvalidateByIDClearsLastTouched documents that
// revoke wipes both the auth cache entry and the touch debounce hint, so
// a re-mint of the same id (or a stale process restart scenario) is not
// silently suppressed.
func TestVerificationCache_InvalidateByIDClearsLastTouched(t *testing.T) {
	c := NewVerificationCache(60 * time.Second)
	id := uuid.New()
	debounce := 60 * time.Second

	assert.True(t, c.ShouldTouch(id, debounce))
	assert.False(t, c.ShouldTouch(id, debounce))

	c.InvalidateByID(id)

	// After invalidate the slot is gone, so the very next call is allowed.
	assert.True(t, c.ShouldTouch(id, debounce), "InvalidateByID must clear the touch hint")
}

// TestVerificationCache_ClearResetsLastTouched mirrors the above for the
// blanket Clear() helper that tests use.
func TestVerificationCache_ClearResetsLastTouched(t *testing.T) {
	c := NewVerificationCache(60 * time.Second)
	id := uuid.New()
	debounce := 60 * time.Second

	assert.True(t, c.ShouldTouch(id, debounce))
	assert.False(t, c.ShouldTouch(id, debounce))

	c.Clear()

	assert.True(t, c.ShouldTouch(id, debounce), "Clear must reset the touch hint")
}
