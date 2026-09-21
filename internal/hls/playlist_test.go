package hls

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeMultivariantTwitchStandardVideoAttributes(t *testing.T) {
	input := `#EXTM3U
#EXT-X-TWITCH-INFO:ORIGIN="sfo01",B="false"
#EXT-X-STREAM-INF:BANDWIDTH=6000000,CODECS="avc1.64002a,mp4a.40.2",RESOLUTION=1920x1080,FRAME-RATE=60.000,VIDEO="chunked",AUDIO="audio"
https://example.com/source/index-dvr.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=3000000,CODECS="avc1.640020,mp4a.40.2",RESOLUTION=1280x720,FRAME-RATE=60.000,VIDEO="720p60",AUDIO="audio"
https://example.com/720p60/index-dvr.m3u8
`

	pl, err := DecodeMultivariant(strings.NewReader(input))
	if err != nil {
		t.Fatalf("DecodeMultivariant returned error: %v", err)
	}

	if len(pl.Variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(pl.Variants))
	}

	if pl.Variants[0].Video != "chunked" {
		t.Fatalf("expected first variant video chunked, got %q", pl.Variants[0].Video)
	}
	if pl.Variants[1].Video != "720p60" {
		t.Fatalf("expected second variant video 720p60, got %q", pl.Variants[1].Video)
	}
}

func TestDecodeMultivariantTwitchSessionDataWithoutVideoAttributes(t *testing.T) {
	input := `#EXTM3U
#EXT-X-SESSION-DATA:DATA-ID="com.amazon.ivs.unavailable-video-reason",VALUE=""
#EXT-X-SESSION-DATA:DATA-ID="com.amazon.ivs.broadcast-id",VALUE="example-broadcast"
#EXT-X-SESSION-DATA:DATA-ID="com.amazon.ivs.stream-id",VALUE="example-stream"
#EXT-X-SESSION-DATA:DATA-ID="com.amazon.ivs.live-low-latency",VALUE="true"
#EXT-X-STREAM-INF:BANDWIDTH=900000,CODECS="avc1.64001e,mp4a.40.2",RESOLUTION=640x360,FRAME-RATE=30.000,STABLE-VARIANT-ID="360p30",IVS-NAME="360p30",AUDIO="audio"
https://example.com/360p30/index-dvr.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=160000,CODECS="mp4a.40.2",AUDIO="audio"
https://example.com/audio_only/index-dvr.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=6000000,CODECS="avc1.64002a,mp4a.40.2",RESOLUTION=1920x1080,FRAME-RATE=60.000,STABLE-VARIANT-ID="1080p60",IVS-NAME="source",AUDIO="audio"
https://example.com/1080p60/index-dvr.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=3000000,CODECS="avc1.640020,mp4a.40.2",RESOLUTION=1280x720,FRAME-RATE=59.940,AUDIO="audio"
https://example.com/720p60/index-dvr.m3u8
`

	pl, err := DecodeMultivariant(strings.NewReader(input))
	if err != nil {
		t.Fatalf("DecodeMultivariant returned error: %v", err)
	}

	if len(pl.Variants) != 4 {
		t.Fatalf("expected 4 variants, got %d", len(pl.Variants))
	}

	expected := []struct {
		video string
		uri   string
	}{
		{"360p30", "https://example.com/360p30/index-dvr.m3u8"},
		{"audio_only", "https://example.com/audio_only/index-dvr.m3u8"},
		{"1080p60", "https://example.com/1080p60/index-dvr.m3u8"},
		{"720p60", "https://example.com/720p60/index-dvr.m3u8"},
	}

	for i, exp := range expected {
		if pl.Variants[i].Video != exp.video {
			t.Fatalf("variant %d expected video %q, got %q", i, exp.video, pl.Variants[i].Video)
		}
		if pl.Variants[i].URI != exp.uri {
			t.Fatalf("variant %d expected URI %q, got %q", i, exp.uri, pl.Variants[i].URI)
		}
	}
}

func TestDecodeMultivariantDoesNotOverwriteExistingVideo(t *testing.T) {
	input := `#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=6000000,CODECS="avc1.64002a,mp4a.40.2",RESOLUTION=1920x1080,FRAME-RATE=60.000,VIDEO="chunked",STABLE-VARIANT-ID="1080p60",IVS-NAME="source",AUDIO="audio"
https://example.com/source/index-dvr.m3u8
`

	pl, err := DecodeMultivariant(strings.NewReader(input))
	if err != nil {
		t.Fatalf("DecodeMultivariant returned error: %v", err)
	}

	if len(pl.Variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(pl.Variants))
	}
	if pl.Variants[0].Video != "chunked" {
		t.Fatalf("expected existing VIDEO to be preserved, got %q", pl.Variants[0].Video)
	}
}

func TestFinalizeMediaPlaylistClosesInterruptedEventPlaylist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "video.m3u8")
	input := `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-PLAYLIST-TYPE:EVENT
#EXTINF:10.0,
segment0.ts
`
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatalf("failed to write playlist: %v", err)
	}

	if err := FinalizeMediaPlaylist(path); err != nil {
		t.Fatalf("FinalizeMediaPlaylist returned error: %v", err)
	}

	outputBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read playlist: %v", err)
	}
	output := string(outputBytes)
	if strings.Contains(output, "#EXT-X-PLAYLIST-TYPE:EVENT") {
		t.Fatalf("expected EVENT playlist type to be replaced, got:\n%s", output)
	}
	if !strings.Contains(output, "#EXT-X-PLAYLIST-TYPE:VOD") {
		t.Fatalf("expected VOD playlist type, got:\n%s", output)
	}
	if !strings.HasSuffix(output, "#EXT-X-ENDLIST\n") {
		t.Fatalf("expected playlist to end with ENDLIST, got:\n%s", output)
	}
	if !strings.Contains(output, "segment0.ts") {
		t.Fatalf("expected segment entries to be preserved, got:\n%s", output)
	}
}

func TestFinalizeMediaPlaylistIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "video.m3u8")
	input := `#EXTM3U
#EXT-X-PLAYLIST-TYPE:VOD
#EXTINF:10.0,
segment0.ts
#EXT-X-ENDLIST
`
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatalf("failed to write playlist: %v", err)
	}

	if err := FinalizeMediaPlaylist(path); err != nil {
		t.Fatalf("first FinalizeMediaPlaylist returned error: %v", err)
	}
	if err := FinalizeMediaPlaylist(path); err != nil {
		t.Fatalf("second FinalizeMediaPlaylist returned error: %v", err)
	}

	outputBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read playlist: %v", err)
	}
	output := string(outputBytes)
	if strings.Count(output, "#EXT-X-ENDLIST") != 1 {
		t.Fatalf("expected one ENDLIST marker, got:\n%s", output)
	}
	if strings.Count(output, "#EXT-X-PLAYLIST-TYPE:VOD") != 1 {
		t.Fatalf("expected one VOD playlist type, got:\n%s", output)
	}
}

func writeSegmentFile(t *testing.T, dir, name string, size int) {
	t.Helper()
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 251)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatalf("failed to write segment %s: %v", name, err)
	}
}

func fixedProbe(durations map[string]float64) SegmentProbe {
	return func(ctx context.Context, path string) (float64, error) {
		d, ok := durations[filepath.Base(path)]
		if !ok {
			return 10.0, nil
		}
		return d, nil
	}
}

func TestRebuildMediaPlaylistFromTruncatedFmp4(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	const extID = "12345"
	playlistPath := filepath.Join(dir, extID+"-video.m3u8")

	writeSegmentFile(t, dir, extID+"_init.mp4", 100)
	writeSegmentFile(t, dir, extID+"_segment000000.m4s", 100)
	writeSegmentFile(t, dir, extID+"_segment000001.m4s", 100)
	writeSegmentFile(t, dir, extID+"_segment000002.m4s", 100)
	// Zero-byte partial from an interrupted rollover must be skipped.
	writeSegmentFile(t, dir, extID+"_segment000003.m4s", 0)
	// Truncated playlist left by a kill mid-rewrite.
	if err := os.WriteFile(playlistPath, []byte{}, 0o644); err != nil {
		t.Fatalf("failed to write truncated playlist: %v", err)
	}

	if !HasRecoverableSegments(dir, extID) {
		t.Fatal("expected recoverable segments to be detected")
	}

	probe := fixedProbe(map[string]float64{
		extID + "_segment000000.m4s": 10.0,
		extID + "_segment000001.m4s": 10.0,
		extID + "_segment000002.m4s": 6.5,
	})
	if err := RebuildMediaPlaylistFromSegments(context.Background(), dir, extID, playlistPath, probe); err != nil {
		t.Fatalf("RebuildMediaPlaylistFromSegments returned error: %v", err)
	}

	outputBytes, err := os.ReadFile(playlistPath)
	if err != nil {
		t.Fatalf("failed to read rebuilt playlist: %v", err)
	}
	output := string(outputBytes)
	for _, want := range []string{
		"#EXT-X-VERSION:7",
		"#EXT-X-PLAYLIST-TYPE:VOD",
		`#EXT-X-MAP:URI="12345_init.mp4"`,
		"12345_segment000000.m4s",
		"12345_segment000001.m4s",
		"12345_segment000002.m4s",
		"#EXT-X-ENDLIST",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("rebuilt playlist missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "segment000003") {
		t.Fatalf("zero-byte partial segment must be skipped:\n%s", output)
	}
	first := strings.Index(output, "segment000000.m4s")
	second := strings.Index(output, "segment000001.m4s")
	third := strings.Index(output, "segment000002.m4s")
	if first == -1 || second == -1 || third == -1 || first >= second || second >= third {
		t.Fatalf("segments out of order:\n%s", output)
	}
}

func TestRebuildMediaPlaylistFromMissingPlaylist(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	const extID = "abc"
	playlistPath := filepath.Join(dir, extID+"-video.m3u8")

	writeSegmentFile(t, dir, extID+"_init.mp4", 100)
	writeSegmentFile(t, dir, extID+"_segment000000.m4s", 100)
	// No playlist file at all: the atomic-create fallback must handle it.
	if err := RebuildMediaPlaylistFromSegments(context.Background(), dir, extID, playlistPath, fixedProbe(nil)); err != nil {
		t.Fatalf("RebuildMediaPlaylistFromSegments returned error: %v", err)
	}
	outputBytes, err := os.ReadFile(playlistPath)
	if err != nil {
		t.Fatalf("failed to read rebuilt playlist: %v", err)
	}
	if !strings.Contains(string(outputBytes), "#EXT-X-ENDLIST") {
		t.Fatalf("expected finalized playlist, got:\n%s", outputBytes)
	}
}

func TestRebuildMediaPlaylistSortsUnpaddedSegments(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	const extID = "live"
	playlistPath := filepath.Join(dir, extID+"-video.m3u8")

	for _, name := range []string{"live_segment1.ts", "live_segment10.ts", "live_segment2.ts"} {
		writeSegmentFile(t, dir, name, 100)
	}
	if err := RebuildMediaPlaylistFromSegments(context.Background(), dir, extID, playlistPath, fixedProbe(nil)); err != nil {
		t.Fatalf("RebuildMediaPlaylistFromSegments returned error: %v", err)
	}
	outputBytes, err := os.ReadFile(playlistPath)
	if err != nil {
		t.Fatalf("failed to read rebuilt playlist: %v", err)
	}
	output := string(outputBytes)
	first := strings.Index(output, "segment1.ts")
	tenth := strings.Index(output, "segment10.ts")
	second := strings.Index(output, "segment2.ts")
	if first == -1 || tenth == -1 || second == -1 || first >= second || second >= tenth {
		t.Fatalf("expected numeric order 1,2,10, got:\n%s", output)
	}
	if strings.Contains(output, "#EXT-X-MAP") {
		t.Fatalf("legacy TS playlist must not reference an init file:\n%s", output)
	}
}

func TestRebuildMediaPlaylistSkipsUnprobeableSegments(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	const extID = "skip"
	playlistPath := filepath.Join(dir, extID+"-video.m3u8")

	writeSegmentFile(t, dir, extID+"_init.mp4", 100)
	writeSegmentFile(t, dir, extID+"_segment000000.m4s", 100)
	writeSegmentFile(t, dir, extID+"_segment000001.m4s", 100)
	writeSegmentFile(t, dir, extID+"_segment000002.m4s", 100)

	probe := func(ctx context.Context, path string) (float64, error) {
		if filepath.Base(path) == extID+"_segment000001.m4s" {
			return 0, fmt.Errorf("truncated segment")
		}
		return 10.0, nil
	}
	if err := RebuildMediaPlaylistFromSegments(context.Background(), dir, extID, playlistPath, probe); err != nil {
		t.Fatalf("RebuildMediaPlaylistFromSegments returned error: %v", err)
	}

	outputBytes, err := os.ReadFile(playlistPath)
	if err != nil {
		t.Fatalf("failed to read rebuilt playlist: %v", err)
	}
	output := string(outputBytes)
	if strings.Contains(output, "segment000001") {
		t.Fatalf("unprobeable segment must be skipped:\n%s", output)
	}
	if !strings.Contains(output, "#EXT-X-DISCONTINUITY") {
		t.Fatalf("expected a discontinuity marker for the skipped segment:\n%s", output)
	}
	first := strings.Index(output, "segment000000.m4s")
	last := strings.Index(output, "segment000002.m4s")
	if first == -1 || last == -1 || first >= last {
		t.Fatalf("segments out of order:\n%s", output)
	}
}

func TestRebuildMediaPlaylistFailsWhenNoSegmentProbes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	const extID = "broken"
	playlistPath := filepath.Join(dir, extID+"-video.m3u8")

	writeSegmentFile(t, dir, extID+"_init.mp4", 100)
	writeSegmentFile(t, dir, extID+"_segment000000.m4s", 100)

	probe := func(ctx context.Context, path string) (float64, error) {
		return 0, fmt.Errorf("corrupt")
	}
	if err := RebuildMediaPlaylistFromSegments(context.Background(), dir, extID, playlistPath, probe); err == nil {
		t.Fatal("expected rebuild error when no segment can be probed")
	}
}

func TestURIEncodesReservedCharacters(t *testing.T) {
	t.Parallel()

	if got := URI("12345_init.mp4"); got != "12345_init.mp4" {
		t.Fatalf("URI() = %q, want plain filename", got)
	}
	if got := URI("/tmp/a b/seg#1.m4s"); got != "/tmp/a%20b/seg%231.m4s" {
		t.Fatalf("URI() = %q, want percent-encoded path", got)
	}
}

func TestRebuildMediaPlaylistWithoutSegmentsFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	const extID = "empty"
	playlistPath := filepath.Join(dir, extID+"-video.m3u8")

	if HasRecoverableSegments(dir, extID) {
		t.Fatal("expected no recoverable segments in an empty directory")
	}
	if err := RebuildMediaPlaylistFromSegments(context.Background(), dir, extID, playlistPath, fixedProbe(nil)); err == nil {
		t.Fatal("expected rebuild error with no segments")
	}

	// fmp4 segments without their init file are not recoverable.
	writeSegmentFile(t, dir, extID+"_segment000000.m4s", 100)
	if HasRecoverableSegments(dir, extID) {
		t.Fatal("expected no recoverable segments without the init file")
	}
	if err := RebuildMediaPlaylistFromSegments(context.Background(), dir, extID, playlistPath, fixedProbe(nil)); err == nil {
		t.Fatal("expected rebuild error without the init file")
	}
}

func TestLiveCaptureIDPrefersImmutableStreamID(t *testing.T) {
	t.Parallel()

	if got := LiveCaptureID("vod999", "stream123", "/tmp/stream123_uuid-video_hls0/stream123-video.m3u8"); got != "stream123" {
		t.Fatalf("LiveCaptureID with stream ID = %q, want stream123", got)
	}
	if got := LiveCaptureID("vod999", "", "/tmp/stream123_uuid-video_hls0/stream123-video.m3u8"); got != "stream123" {
		t.Fatalf("LiveCaptureID from persisted path = %q, want stream123", got)
	}
	if got := LiveCaptureID("vod999", "", "/tmp/other-video.mp4"); got != "vod999" {
		t.Fatalf("LiveCaptureID fallback = %q, want vod999", got)
	}
}
