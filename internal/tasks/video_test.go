package tasks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zibbp/ganymede/ent"
)

func TestValidateNonEmptyFile(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		err := validateNonEmptyFile(filepath.Join(t.TempDir(), "missing.mp4"), "test file")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.mp4")
		if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
			t.Fatalf("failed to create empty file: %v", err)
		}

		err := validateNonEmptyFile(path, "test file")
		if err == nil {
			t.Fatal("expected error for empty file")
		}
	})

	t.Run("non-empty file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "ok.mp4")
		if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
			t.Fatalf("failed to create non-empty file: %v", err)
		}

		err := validateNonEmptyFile(path, "test file")
		if err != nil {
			t.Fatalf("expected nil error for non-empty file, got: %v", err)
		}
	})
}

func TestLiveHlsCaptureSurvivesStreamVideoIDUpdate(t *testing.T) {
	t.Parallel()

	const streamID = "stream123"
	const vodID = "vod999"
	tmpHlsPath := filepath.Join(t.TempDir(), streamID+"_uuid-video_hls0")
	video := ent.Vod{
		ExtID:                vodID,
		ExtStreamID:          streamID,
		TmpVideoHlsPath:      tmpHlsPath,
		TmpVideoDownloadPath: filepath.Join(tmpHlsPath, streamID+"-video.m3u8"),
	}

	if got := liveCaptureID(&video); got != streamID {
		t.Fatalf("liveCaptureID = %q, want %q", got, streamID)
	}
	if got := liveHlsPlaylistPath(&video); got != video.TmpVideoDownloadPath {
		t.Fatalf("liveHlsPlaylistPath = %q, want %q", got, video.TmpVideoDownloadPath)
	}
	if !isLiveHlsCapture(&video) {
		t.Fatal("mutated ExtID must still be detected as an HLS capture")
	}
}

func TestLiveCaptureIDFallsBackToPersistedPath(t *testing.T) {
	t.Parallel()

	const streamID = "stream123"
	tmpHlsPath := filepath.Join(t.TempDir(), streamID+"_uuid-video_hls0")
	video := ent.Vod{
		ExtID:                "vod999",
		TmpVideoHlsPath:      tmpHlsPath,
		TmpVideoDownloadPath: filepath.Join(tmpHlsPath, streamID+"-video.m3u8"),
	}

	if got := liveCaptureID(&video); got != streamID {
		t.Fatalf("liveCaptureID fallback = %q, want %q", got, streamID)
	}
	if !isLiveHlsCapture(&video) {
		t.Fatal("persisted-path fallback must still be detected as an HLS capture")
	}
}
