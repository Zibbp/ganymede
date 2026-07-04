package hls

import (
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
