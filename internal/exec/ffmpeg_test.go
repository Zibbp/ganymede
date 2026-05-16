package exec

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// createDummyVideo creates a small test video file using ffmpeg.
func createDummyVideo(t *testing.T, dir string) string {
	t.Helper()
	videoPath := filepath.Join(dir, "test.mp4")
	cmd := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "testsrc=duration=2:size=128x128:rate=1", "-c:v", "libx264", "-pix_fmt", "yuv420p", videoPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to create dummy video: %v, output: %s", err, string(out))
	}
	return videoPath
}

func TestGetVideoDuration(t *testing.T) {
	tmpDir := t.TempDir()
	videoPath := createDummyVideo(t, tmpDir)

	ctx := context.Background()
	duration, err := GetVideoDuration(ctx, videoPath)
	if err != nil {
		t.Fatalf("GetVideoDuration failed: %v", err)
	}
	// The duration should be close to 2 seconds (allowing some tolerance)
	if duration < 1 || duration > 3 {
		t.Errorf("unexpected duration: got %d, want ~2", duration)
	}
}

func TestGetVideoDuration_FileNotExist(t *testing.T) {
	ctx := context.Background()
	_, err := GetVideoDuration(ctx, "/nonexistent/file.mp4")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestGetFfprobeData(t *testing.T) {
	tmpDir := t.TempDir()
	videoPath := createDummyVideo(t, tmpDir)

	ctx := context.Background()
	data, err := GetFfprobeData(ctx, videoPath)
	if err != nil {
		t.Fatalf("GetFfprobeData failed: %v", err)
	}
	if data == nil {
		t.Fatal("expected non-nil data")
	}
	// Check for expected keys
	if _, ok := data["streams"]; !ok {
		t.Error("expected 'streams' key in ffprobe data")
	}
	if _, ok := data["format"]; !ok {
		t.Error("expected 'format' key in ffprobe data")
	}
}

func TestGetFfprobeData_FileNotExist(t *testing.T) {
	ctx := context.Background()
	_, err := GetFfprobeData(ctx, "/nonexistent/file.mp4")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestGetFfprobeData_InvalidJSON(t *testing.T) {
	// Create a text file (not a video) to force ffprobe to fail JSON parsing
	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "notavideo.txt")
	if err := os.WriteFile(txtPath, []byte("not a video"), 0644); err != nil {
		t.Fatalf("failed to write dummy text file: %v", err)
	}
	ctx := context.Background()
	_, err := GetFfprobeData(ctx, txtPath)
	if err == nil {
		t.Error("expected error for invalid file, got nil")
	}
}

// Optional: Test JSON parsing error by mocking ffprobe output (advanced, requires more setup)
func TestGetFfprobeData_BadJSON(t *testing.T) {
	// This test simulates ffprobe returning invalid JSON by using a shell script as ffprobe
	tmpDir := t.TempDir()
	ffprobePath := filepath.Join(tmpDir, "ffprobe")
	script := "#!/bin/sh\necho 'not json'"
	if err := os.WriteFile(ffprobePath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake ffprobe: %v", err)
	}
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)      //nolint:errcheck
	os.Setenv("PATH", tmpDir+":"+origPath) //nolint:errcheck

	ctx := context.Background()
	_, err := GetFfprobeData(ctx, "dummy.mp4")
	if err == nil || !strings.Contains(err.Error(), "failed to unmarshal ffprobe output") {
		t.Errorf("expected JSON unmarshal error, got: %v", err)
	}
}
func TestGetFfprobeVideoData_Success(t *testing.T) {
	tmpDir := t.TempDir()
	videoPath := createDummyVideo(t, tmpDir)

	ctx := context.Background()
	data, err := GetFfprobeVideoData(ctx, videoPath)
	if err != nil {
		t.Fatalf("GetFfprobeVideoData failed: %v", err)
	}
	if data == nil { //nolint:all
		t.Fatal("expected non-nil data")
	}
	if len(data.Streams) == 0 { //nolint:all
		t.Error("expected at least one stream in ffprobe data")
	}
	if data.Format.Filename == "" { //nolint:all
		t.Error("expected filename in ffprobe format data")
	}
}

func TestGetFfprobeVideoData_FileNotExist(t *testing.T) {
	ctx := context.Background()
	_, err := GetFfprobeVideoData(ctx, "/nonexistent/file.mp4")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestGetFfprobeVideoData_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "notavideo.txt")
	if err := os.WriteFile(txtPath, []byte("not a video"), 0644); err != nil {
		t.Fatalf("failed to write dummy text file: %v", err)
	}
	ctx := context.Background()
	_, err := GetFfprobeVideoData(ctx, txtPath)
	if err == nil {
		t.Error("expected error for invalid file, got nil")
	}
}

func TestGetFfprobeVideoData_BadJSON(t *testing.T) {
	tmpDir := t.TempDir()
	ffprobePath := filepath.Join(tmpDir, "ffprobe")
	script := "#!/bin/sh\necho 'not json'"
	if err := os.WriteFile(ffprobePath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake ffprobe: %v", err)
	}
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)      //nolint:errcheck
	os.Setenv("PATH", tmpDir+":"+origPath) //nolint:errcheck

	ctx := context.Background()
	_, err := GetFfprobeVideoData(ctx, "dummy.mp4")
	if err == nil || !strings.Contains(err.Error(), "failed to unmarshal ffprobe output") {
		t.Errorf("expected JSON unmarshal error, got: %v", err)
	}
}

func TestGetFfprobeVideoData_NoStreams(t *testing.T) {
	// Simulate ffprobe output with no streams
	tmpDir := t.TempDir()
	ffprobePath := filepath.Join(tmpDir, "ffprobe")
	jsonOut := `{"streams":[],"format":{"filename":"dummy.mp4"}}`
	script := "#!/bin/sh\necho '" + jsonOut + "'"
	if err := os.WriteFile(ffprobePath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake ffprobe: %v", err)
	}
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)      //nolint:errcheck
	os.Setenv("PATH", tmpDir+":"+origPath) //nolint:errcheck

	ctx := context.Background()
	_, err := GetFfprobeVideoData(ctx, "dummy.mp4")
	if err == nil || !strings.Contains(err.Error(), "no streams found") {
		t.Errorf("expected no streams error, got: %v", err)
	}
}

func TestGetFfprobeVideoData_NoFilename(t *testing.T) {
	// Simulate ffprobe output with no filename in format
	tmpDir := t.TempDir()
	ffprobePath := filepath.Join(tmpDir, "ffprobe")
	jsonOut := `{"streams":[{"index":0}],"format":{}}`
	script := "#!/bin/sh\necho '" + jsonOut + "'"
	if err := os.WriteFile(ffprobePath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake ffprobe: %v", err)
	}
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)      //nolint:errcheck
	os.Setenv("PATH", tmpDir+":"+origPath) //nolint:errcheck

	ctx := context.Background()
	_, err := GetFfprobeVideoData(ctx, "dummy.mp4")
	if err == nil || !strings.Contains(err.Error(), "no filename found") {
		t.Errorf("expected no filename error, got: %v", err)
	}
}
