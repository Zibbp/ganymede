package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

func TestGetVideoDurationPrefersStreamDurationWithLargeInitialGap(t *testing.T) {
	tmpDir := t.TempDir()
	mediaPath := createTimestampOffsetMedia(t, tmpDir, "mpegts")

	probe, err := ProbeMediaDuration(t.Context(), mediaPath)
	if err != nil {
		t.Fatalf("probe media with initial gap: %v", err)
	}
	if probe.HasTimestampAnomaly() {
		t.Fatalf("initial gap alone must not be treated as corrupt timestamps: %+v", probe)
	}

	duration, err := GetVideoDuration(t.Context(), mediaPath)
	if err != nil {
		t.Fatalf("get validated duration: %v", err)
	}
	if duration < 2 || duration > 4 {
		t.Fatalf("validated duration = %d, want about 3 seconds", duration)
	}
}

func TestMediaDurationHasTimestampAnomaly(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		duration MediaDuration
		want     bool
	}{
		{name: "aligned", duration: MediaDuration{FormatDuration: 3600, LongestStreamDuration: 3600, MaxStreamEndTime: 3600}},
		{name: "late stream", duration: MediaDuration{FormatDuration: 3600, LongestStreamDuration: 3600, MaxStreamStartTime: 12.13, MaxStreamEndTime: 3600}},
		{name: "initial gap", duration: MediaDuration{FormatDuration: 15, LongestStreamDuration: 3, MaxStreamStartTime: 12, MaxStreamEndTime: 15}},
		{name: "shared start offset", duration: MediaDuration{FormatDuration: 20, LongestStreamDuration: 3, MinStreamStartTime: 5, MaxStreamStartTime: 17}, want: true},
		{name: "priming", duration: MediaDuration{FormatDuration: 3600, LongestStreamDuration: 3600, MaxStreamStartTime: 0.1, MaxStreamEndTime: 3600.1}},
		{name: "different latest and longest streams", duration: MediaDuration{FormatDuration: 150, LongestStreamDuration: 100, MaxStreamStartTime: 90, MaxStreamEndTime: 100}, want: true},
		{name: "relative tolerance", duration: MediaDuration{FormatDuration: 1100, LongestStreamDuration: 1000, MaxStreamEndTime: 1000}},
		{name: "beyond relative tolerance", duration: MediaDuration{FormatDuration: 1101, LongestStreamDuration: 1000, MaxStreamEndTime: 1000}, want: true},
		{name: "inflated container", duration: MediaDuration{FormatDuration: 20003, LongestStreamDuration: 3}, want: true},
		{name: "inflated beyond gap", duration: MediaDuration{FormatDuration: 100, LongestStreamDuration: 3, MaxStreamStartTime: 12}, want: true},
		{name: "unknown streams", duration: MediaDuration{FormatDuration: 100}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.duration.HasTimestampAnomaly(); got != tc.want {
				t.Fatalf("HasTimestampAnomaly() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestProbeMediaDurationUsesPerStreamEndTimes(t *testing.T) {
	for _, formatDuration := range []int{100, 150} {
		t.Run(strconv.Itoa(formatDuration), func(t *testing.T) {
			dir := t.TempDir()
			writeExecutable(t, filepath.Join(dir, "ffprobe"), fmt.Sprintf(`#!/bin/sh
echo '{"streams":[{"codec_type":"video","duration":"100","start_time":"0"},{"codec_type":"audio","duration":"1","start_time":"90"},{"codec_type":"subtitle","duration":"500","start_time":"500"}],"format":{"duration":"%d"}}'
`, formatDuration))
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			duration, err := ProbeMediaDuration(t.Context(), "input.mp4")
			if err != nil {
				t.Fatal(err)
			}
			if duration.MaxStreamEndTime != 100 || duration.MaxStreamStartTime != 90 || duration.LongestStreamDuration != 100 {
				t.Fatalf("unexpected stream bounds: %+v", duration)
			}
			wantAnomaly := formatDuration == 150
			if duration.HasTimestampAnomaly() != wantAnomaly {
				t.Fatalf("HasTimestampAnomaly() = %v, want %v", duration.HasTimestampAnomaly(), wantAnomaly)
			}
			var passes []bool
			committed := false
			err = exportWithTimestampRepair(t.Context(), "test", "output.mp4", func(normalize bool) error {
				passes = append(passes, normalize)
				return nil
			}, func() error {
				committed = true
				return nil
			}, io.Discard)
			if wantAnomaly {
				if err == nil || !strings.Contains(err.Error(), "did not repair") || committed || len(passes) != 2 || passes[0] || !passes[1] {
					t.Fatalf("inflated timeline was not rejected: err=%v committed=%v passes=%v", err, committed, passes)
				}
			} else if err != nil || !committed || len(passes) != 1 || passes[0] {
				t.Fatalf("valid timeline was not committed: err=%v committed=%v passes=%v", err, committed, passes)
			}
		})
	}
}

func createTimestampOffsetMedia(t *testing.T, dir, format string) string {
	t.Helper()
	return createStreamOffsetMedia(t, dir, format, 20000, 0)
}

func createStreamOffsetMedia(t *testing.T, dir, format string, videoOffset, audioOffset int) string {
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
		"-filter_complex", fmt.Sprintf("[0:v]setpts=PTS+%d/TB[v];[1:a]asetpts=PTS+%d/TB[a]", videoOffset, audioOffset),
		"-map", "[v]", "-map", "[a]",
		"-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p",
		"-c:a", "aac",
		// Preserve the initial gap instead of filling it with duplicate frames.
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

func mediaAVStartOffset(t *testing.T, path string) float64 {
	t.Helper()
	// HLS stream start metadata can differ from the actual packet timestamps.
	out, err := exec.CommandContext(t.Context(), "ffprobe", "-v", "error",
		"-show_entries", "packet=codec_type,pts_time", "-of", "json", path).Output()
	if err != nil {
		t.Fatalf("probe packet timestamps: %v", err)
	}
	var data struct {
		Packets []struct {
			CodecType string `json:"codec_type"`
			PTSTime   string `json:"pts_time"`
		} `json:"packets"`
	}
	if err := json.Unmarshal(out, &data); err != nil {
		t.Fatalf("parse packet timestamps: %v", err)
	}
	starts := make(map[string]float64)
	for _, packet := range data.Packets {
		if packet.CodecType != "video" && packet.CodecType != "audio" {
			continue
		}
		start, err := strconv.ParseFloat(packet.PTSTime, 64)
		if err != nil {
			t.Fatalf("parse %s packet timestamp %q: %v", packet.CodecType, packet.PTSTime, err)
		}
		if previous, ok := starts[packet.CodecType]; !ok || start < previous {
			starts[packet.CodecType] = start
		}
	}
	if len(starts) != 2 {
		t.Fatalf("expected audio and video streams, got %v", starts)
	}
	return starts["video"] - starts["audio"]
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
