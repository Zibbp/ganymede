package tasks

import (
	"testing"

	"github.com/zibbp/ganymede/internal/platform"
)

func TestPickStreamVod(t *testing.T) {
	t.Parallel()

	videos := []platform.VideoInfo{
		{ID: "10", StreamID: "100"},
		{ID: "20", StreamID: "200"},
		{ID: "30", StreamID: "300"},
	}

	tests := []struct {
		name        string
		videos      []platform.VideoInfo
		extStreamID string
		wantID      string
	}{
		{
			name:        "matches archived broadcast by stream id",
			videos:      videos,
			extStreamID: "200",
			wantID:      "20",
		},
		{
			name:        "first match wins when stream ids repeat",
			videos:      []platform.VideoInfo{{ID: "11", StreamID: "100"}, {ID: "12", StreamID: "100"}},
			extStreamID: "100",
			wantID:      "11",
		},
		{
			name:        "no matching stream id",
			videos:      videos,
			extStreamID: "999",
			wantID:      "",
		},
		{
			name:        "archived broadcast matches the last video",
			videos:      videos,
			extStreamID: "300",
			wantID:      "30",
		},
		{
			name:        "no videos",
			videos:      nil,
			extStreamID: "100",
			wantID:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := pickStreamVod(tt.videos, tt.extStreamID)
			if tt.wantID == "" && got != nil {
				t.Fatalf("pickStreamVod() = %+v, want nil", got)
			}
			if tt.wantID != "" && (got == nil || got.ID != tt.wantID) {
				t.Fatalf("pickStreamVod() = %+v, want video id %s", got, tt.wantID)
			}
		})
	}
}
