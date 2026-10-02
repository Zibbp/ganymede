package hls

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateMediaPlaylistFiles(t *testing.T) {
	dir := t.TempDir()
	playlistPath := filepath.Join(dir, "video.m3u8")
	playlist := "#EXTM3U\n#EXT-X-VERSION:7\n#EXT-X-TARGETDURATION:1\n#EXT-X-MEDIA-SEQUENCE:0\n#EXT-X-MAP:URI=\"init.mp4\"\n#EXTINF:1.0,\nsegment.m4s\n#EXT-X-ENDLIST\n"
	if err := os.WriteFile(playlistPath, []byte(playlist), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "init.mp4"), []byte("init"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := ValidateMediaPlaylistFiles(playlistPath); err == nil {
		t.Fatal("expected missing segment to fail validation")
	}
	if err := os.WriteFile(filepath.Join(dir, "segment.m4s"), []byte("media"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateMediaPlaylistFiles(playlistPath); err != nil {
		t.Fatalf("expected complete playlist to pass validation: %v", err)
	}
}
