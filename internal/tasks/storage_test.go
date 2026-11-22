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
		name         string
		previous     storage.State
		current      storage.State
		notify       bool
		stopCaptures bool
	}{
		{"startup ok does not notify", "", storage.StateOK, false, false},
		{"ok to warning notifies", storage.StateOK, storage.StateWarning, true, false},
		{"warning to critical notifies and stops captures", storage.StateWarning, storage.StateCritical, true, true},
		{"ok to critical notifies and stops captures", storage.StateOK, storage.StateCritical, true, true},
		{"warning to ok notifies recovery", storage.StateWarning, storage.StateOK, true, false},
		{"critical to ok notifies recovery", storage.StateCritical, storage.StateOK, true, false},
		{"ok to ok does not notify", storage.StateOK, storage.StateOK, false, false},
		{"ok to disabled does not notify", storage.StateOK, storage.StateDisabled, false, false},
		{"ok to unknown does not notify", storage.StateOK, storage.StateUnknown, false, false},
		{"disabled to ok does not notify", storage.StateDisabled, storage.StateOK, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			notify, stopCaptures := storageWatchdogAction(tc.previous, tc.current)
			assert.Equal(t, tc.notify, notify)
			assert.Equal(t, tc.stopCaptures, stopCaptures)
		})
	}
}
