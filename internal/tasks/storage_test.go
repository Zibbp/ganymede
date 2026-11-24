package tasks

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/internal/storage"
	"golang.org/x/sys/unix"
)

func TestIsNoSpaceError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"raw syscall error", unix.ENOSPC, true},
		{"wrapped syscall error", fmt.Errorf("convert video: %w", unix.ENOSPC), true},
		{"ffmpeg stderr text", errors.New("ffmpeg: error writing to '/data/temp/x.mp4': No space left on device"), true},
		{"unrelated error", errors.New("connection reset by peer"), false},
		{"nil error", nil, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isNoSpaceError(tc.err))
		})
	}
}

func TestStorageWatchdogAction(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		previous storage.State
		current  storage.State
		notify   bool
	}{
		{"startup ok does not notify", "", storage.StateOK, false},
		{"ok to warning notifies", storage.StateOK, storage.StateWarning, true},
		{"warning to critical notifies", storage.StateWarning, storage.StateCritical, true},
		{"ok to critical notifies", storage.StateOK, storage.StateCritical, true},
		{"warning to ok notifies recovery", storage.StateWarning, storage.StateOK, true},
		{"critical to ok notifies recovery", storage.StateCritical, storage.StateOK, true},
		{"ok to ok does not notify", storage.StateOK, storage.StateOK, false},
		{"ok to disabled does not notify", storage.StateOK, storage.StateDisabled, false},
		{"ok to unknown does not notify", storage.StateOK, storage.StateUnknown, false},
		{"disabled to ok does not notify", storage.StateDisabled, storage.StateOK, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.notify, storageWatchdogAction(tc.previous, tc.current))
		})
	}
}

func TestWarningUnknownOKSequence(t *testing.T) {
	t.Parallel()

	// An unreadable reading between warning and OK must not swallow the
	// recovery notification.
	lastSeen := storage.StateOK
	lastSeen = lastSeenUpdate(lastSeen, storage.StateWarning)
	assert.True(t, storageWatchdogAction(lastSeen, storage.StateWarning))

	lastSeen = lastSeenUpdate(lastSeen, storage.StateUnknown)
	assert.True(t, storageWatchdogAction(lastSeen, storage.StateOK), "recovery after unknown should notify")

	lastSeen = lastSeenUpdate(lastSeen, storage.StateOK)
	// A second OK is silent again.
	assert.False(t, storageWatchdogAction(lastSeen, storage.StateOK))
}
