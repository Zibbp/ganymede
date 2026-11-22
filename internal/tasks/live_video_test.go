package tasks

import (
	"testing"
	"time"
)

func TestKickChatAnchorPlausible(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 22, 20, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		t    time.Time
		want bool
	}{
		{"zero", time.Time{}, false},
		{"hls latency behind", now.Add(-8 * time.Second), true},
		{"small clock skew ahead", now.Add(5 * time.Second), true},
		{"far future", now.Add(time.Minute), false},
		{"stale", now.Add(-10 * time.Minute), false},
	}
	for _, tc := range cases {
		if got := kickChatAnchorPlausible(tc.t, now); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}
