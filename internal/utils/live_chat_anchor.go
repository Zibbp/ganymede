package utils

import "time"

// A live capture lags wall-clock time by the stream's broadcast latency, so
// video second 0 is expected slightly before capture start; larger deviations
// mean the capture ended well before the stream and the anchor cannot be trusted.
const (
	// liveChatAnchorMaxSkew bounds how far after capture start the anchor may
	// land (clock skew between capture and chat worker starts).
	liveChatAnchorMaxSkew = 15 * time.Second
	// liveChatAnchorMaxLatency bounds the plausible capture latency range.
	liveChatAnchorMaxLatency = 180 * time.Second
)

// ComputeLiveChatAnchor returns the wall-clock time that video second 0 of a
// live capture corresponds to, derived as stream end minus captured video
// duration. ok is false for missing inputs or results outside the plausible
// capture latency bounds (callers anchor chat to the capture start instead).
func ComputeLiveChatAnchor(captureStart time.Time, streamEnd time.Time, videoDurationSec int) (time.Time, bool) {
	if captureStart.IsZero() || streamEnd.IsZero() || videoDurationSec <= 0 {
		return time.Time{}, false
	}

	anchor := streamEnd.Add(-time.Duration(videoDurationSec) * time.Second)
	if anchor.After(captureStart.Add(liveChatAnchorMaxSkew)) {
		// Video second 0 cannot be after the capture started: the captured
		// media ended long before the stream did, so the stream end does not
		// bound the capture.
		return time.Time{}, false
	}
	if anchor.Before(captureStart.Add(-liveChatAnchorMaxLatency)) {
		return time.Time{}, false
	}
	return anchor, true
}
