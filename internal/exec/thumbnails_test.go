package exec

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateThumbnailsStopsAfterConsecutiveOutOfRangeFrames(t *testing.T) {
	tmpDir := t.TempDir()
	videoPath := createDummyVideo(t, tmpDir)
	thumbnailDir := filepath.Join(tmpDir, "thumbnails")
	if err := os.MkdirAll(thumbnailDir, 0o755); err != nil {
		t.Fatalf("create thumbnail directory: %v", err)
	}

	start := time.Now()
	err := GenerateThumbnails(t.Context(), GenerateThumbnailsInput{
		Video:        videoPath,
		Duration:     20_000,
		Interval:     60,
		ThumbnailDir: thumbnailDir,
		Width:        64,
		Height:       64,
	})
	if err != nil {
		t.Fatalf("generate thumbnails: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("out-of-range thumbnail extraction took %s", elapsed)
	}

	thumbnails, err := filepath.Glob(filepath.Join(thumbnailDir, "frame*.jpg"))
	if err != nil {
		t.Fatalf("list thumbnails: %v", err)
	}
	if len(thumbnails) == 0 {
		t.Fatal("expected at least one thumbnail")
	}
}

func TestGenerateThumbnailsRejectsInvalidBounds(t *testing.T) {
	err := GenerateThumbnails(t.Context(), GenerateThumbnailsInput{Duration: 0, Interval: 60})
	if err == nil {
		t.Fatal("expected invalid duration error")
	}

	err = GenerateThumbnails(t.Context(), GenerateThumbnailsInput{Duration: 60, Interval: 0})
	if err == nil {
		t.Fatal("expected invalid interval error")
	}
}
