package utils

import (
	"testing"
	"time"
)

func TestComputeLiveChatAnchor(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	streamEnd := base.Add(1 * time.Hour)

	tests := []struct {
		name             string
		captureStart     time.Time
		streamEnd        time.Time
		videoDurationSec int
		want             time.Time
		wantOK           bool
	}{
		{
			name:             "normal latency stream",
			captureStart:     base,
			streamEnd:        streamEnd,
			videoDurationSec: 3600 + 10, // 10s capture latency
			want:             base.Add(-10 * time.Second),
			wantOK:           true,
		},
		{
			name:             "low latency stream",
			captureStart:     base,
			streamEnd:        streamEnd,
			videoDurationSec: 3600 + 2, // 2s capture latency
			want:             base.Add(-2 * time.Second),
			wantOK:           true,
		},
		{
			// chat start need not equal capture start; the anchor is
			// still stream-end derived
			name:             "chat start slightly after capture start",
			captureStart:     base.Add(5 * time.Second),
			streamEnd:        streamEnd,
			videoDurationSec: 3600 + 10,
			want:             base.Add(-10 * time.Second),
			wantOK:           true,
		},
		{
			name:             "capture stopped long before stream end",
			captureStart:     base,
			streamEnd:        streamEnd.Add(2 * time.Hour),
			videoDurationSec: 600,
			wantOK:           false,
		},
		{
			name:             "anchor lands after capture start",
			captureStart:     base,
			streamEnd:        streamEnd,
			videoDurationSec: 3600 - 300,
			wantOK:           false,
		},
		{
			name:             "implausibly large latency",
			captureStart:     base,
			streamEnd:        streamEnd,
			videoDurationSec: 3600 + 200,
			wantOK:           false,
		},
		{
			name:             "zero stream end",
			captureStart:     base,
			streamEnd:        time.Time{},
			videoDurationSec: 3590,
			wantOK:           false,
		},
		{
			name:             "zero duration",
			captureStart:     base,
			streamEnd:        streamEnd,
			videoDurationSec: 0,
			wantOK:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := ComputeLiveChatAnchor(tt.captureStart, tt.streamEnd, tt.videoDurationSec)
			if ok != tt.wantOK {
				t.Fatalf("ComputeLiveChatAnchor() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && !got.Equal(tt.want) {
				t.Fatalf("ComputeLiveChatAnchor() = %v, want %v", got, tt.want)
			}
		})
	}
}
