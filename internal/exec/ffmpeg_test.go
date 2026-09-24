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

// createHevcMedia creates a small HEVC video with audio. ffmpeg tags HEVC in
// MP4 as hev1 unless told otherwise, which is the state the conversion has to
// correct.
func createHevcMedia(t *testing.T, dir string, withCoverArt bool) string {
	t.Helper()
	requireEncoder(t, "libx265")

	mediaPath := filepath.Join(dir, "hevc.mp4")
	runFFmpeg(t, "create hevc media",
		"-v", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=size=128x72:rate=10:duration=1",
		"-f", "lavfi", "-i", "sine=frequency=1000:duration=1",
		"-map", "0:v", "-map", "1:a",
		"-c:v", "libx265", "-x265-params", "log-level=none", "-pix_fmt", "yuv420p",
		"-c:a", "aac", "-t", "1", mediaPath,
	)
	if !withCoverArt {
		return mediaPath
	}

	// A thumbnail embedded by yt-dlp arrives as a second video stream. Muxing
	// it in afterwards keeps the stream order predictable: video, audio, cover.
	coverPath := filepath.Join(dir, "cover.jpg")
	withCoverPath := filepath.Join(dir, "hevc-with-cover.mp4")
	runFFmpeg(t, "create cover art",
		"-v", "error", "-y",
		"-f", "lavfi", "-i", "color=c=red:size=32x32:duration=1", "-frames:v", "1", coverPath,
	)
	runFFmpeg(t, "embed cover art",
		"-v", "error", "-y", "-i", mediaPath, "-i", coverPath,
		"-map", "0", "-map", "1", "-c", "copy", "-disposition:v:1", "attached_pic", withCoverPath,
	)
	return withCoverPath
}

// createHevcTransportStream creates the shape a live archive is captured in:
// HEVC in MPEG-TS, which has no sample entry tag of its own.
func createHevcTransportStream(t *testing.T, dir string) string {
	t.Helper()
	requireEncoder(t, "libx265")

	mediaPath := filepath.Join(dir, "hevc.ts")
	runFFmpeg(t, "create hevc transport stream",
		"-v", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=size=128x72:rate=10:duration=1",
		"-f", "lavfi", "-i", "sine=frequency=1000:duration=1",
		"-map", "0:v", "-map", "1:a",
		"-c:v", "libx265", "-x265-params", "log-level=none", "-pix_fmt", "yuv420p",
		"-c:a", "aac", "-t", "1", "-f", "mpegts", mediaPath,
	)
	return mediaPath
}

func runFFmpeg(t *testing.T, what string, args ...string) {
	t.Helper()
	if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Fatalf("%s: %v, output: %s", what, err, out)
	}
}

// requireEncoder skips the test when the ffmpeg on PATH cannot encode what the
// fixture needs. The project's own image is built with a full ffmpeg.
func requireEncoder(t *testing.T, encoder string) {
	t.Helper()
	out, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-encoders").Output()
	if err != nil {
		t.Fatalf("list ffmpeg encoders: %v", err)
	}
	if !strings.Contains(string(out), encoder) {
		t.Skipf("ffmpeg on PATH has no %s encoder", encoder)
	}
}

// probeStreamEntry returns one ffprobe stream entry, such as the
// "codec_tag_string" of "v:0", or an empty string when the stream is absent.
func probeStreamEntry(t *testing.T, path, stream, entry string) string {
	t.Helper()
	out, err := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", stream,
		"-show_entries", "stream="+entry,
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	).CombinedOutput()
	if err != nil {
		t.Fatalf("probe %s of %s: %v, output: %s", entry, stream, err, out)
	}
	// A container that carries programs reports every stream twice.
	first, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	return first
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

func TestGetVideoDurationPrefersStreamDurationWhenContainerTimelineIsAnomalous(t *testing.T) {
	tmpDir := t.TempDir()
	mediaPath := createTimestampOffsetMedia(t, tmpDir, "mpegts")

	probe, err := ProbeMediaDuration(t.Context(), mediaPath)
	if err != nil {
		t.Fatalf("probe anomalous media: %v", err)
	}
	if !probe.HasTimestampAnomaly() {
		t.Fatalf("expected timestamp anomaly, got format=%f stream=%f", probe.FormatDuration, probe.LongestStreamDuration)
	}

	duration, err := GetVideoDuration(t.Context(), mediaPath)
	if err != nil {
		t.Fatalf("get validated duration: %v", err)
	}
	if duration < 2 || duration > 4 {
		t.Fatalf("validated duration = %d, want about 3 seconds", duration)
	}
}

func createTimestampOffsetMedia(t *testing.T, dir, format string) string {
	t.Helper()

	extension := format
	if format == "mpegts" {
		extension = "ts"
	}
	mediaPath := filepath.Join(dir, "offset-timestamps."+extension)
	args := []string{
		"ffmpeg", "-v", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=size=320x180:rate=30:duration=3",
		"-f", "lavfi", "-i", "sine=frequency=1000:sample_rate=48000:duration=3",
		"-filter_complex", "[0:v]setpts=PTS+20000/TB[v];[1:a]asetpts=PTS[a]",
		"-map", "[v]", "-map", "[a]",
		"-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p",
		"-c:a", "aac",
		// Preserve the timestamp gap instead of filling it with hundreds of
		// thousands of duplicate frames. The fixture must contain three seconds
		// of media starting at an anomalous timestamp, not 20,003 seconds of
		// encoded video.
		"-fps_mode", "passthrough",
	}
	if format == "mpegts" {
		args = append(args, "-muxdelay", "0")
	}
	args = append(args, "-f", format, mediaPath)
	cmd := exec.Command(args[0], args[1:]...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create timestamp-offset media: %v, output: %s", err, out)
	}
	return mediaPath
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
