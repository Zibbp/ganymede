package exec

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	osExec "os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/hls"
)

// TestLiveFmp4CapturePipelineE2E covers live HLS capture to MP4 export with
// real ffmpeg/ffprobe using a synthetic Twitch-like HLS source.
func TestLiveFmp4CapturePipelineE2E(t *testing.T) {
	// Not parallel: sets process environment for config.
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatalf("create logs dir: %v", err)
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	t.Setenv("TWITCH_CLIENT_ID", "e2e-test")
	t.Setenv("TWITCH_CLIENT_SECRET", "e2e-test")
	t.Setenv("LOGS_DIR", logsDir)
	t.Setenv("CONFIG_DIR", configDir)
	if _, err := config.Init(); err != nil {
		t.Fatalf("init config: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()

	run := func(name string, args ...string) {
		t.Helper()
		cmd := osExec.CommandContext(ctx, name, args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("run %s %v: %v\n%s", name, args, err, out)
		}
	}

	// 1. Synthetic source: 12s H.264/AAC mp4, then HLS VOD with MPEG-TS
	// segments (ADTS AAC, matching the Twitch variant that failed).
	srcMP4 := filepath.Join(tmpDir, "src.mp4")
	run("ffmpeg",
		"-y", "-hide_banner",
		"-f", "lavfi", "-i", "testsrc=size=320x240:rate=15:duration=12",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=12",
		"-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac", "-shortest",
		srcMP4,
	)
	srcHLS := filepath.Join(tmpDir, "src_hls")
	if err := os.MkdirAll(srcHLS, 0o755); err != nil {
		t.Fatalf("create src hls dir: %v", err)
	}
	run("ffmpeg",
		"-y", "-hide_banner", "-i", srcMP4,
		"-c", "copy",
		"-start_number", "0", "-hls_time", "4", "-hls_list_size", "0",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", filepath.Join(srcHLS, "seg%03d.ts"),
		"-f", "hls", filepath.Join(srcHLS, "src.m3u8"),
	)
	server := httptest.NewServer(http.FileServer(http.Dir(srcHLS)))
	defer server.Close()

	// 2. Production capture args; the bsf flag is the regression guard.
	capDir := filepath.Join(tmpDir, "cap_hls0")
	if err := os.MkdirAll(capDir, 0o755); err != nil {
		t.Fatalf("create capture dir: %v", err)
	}
	const extID = "e2e123"
	playlistPath := filepath.Join(capDir, extID+"-video.m3u8")
	segmentPattern := filepath.Join(capDir, extID+"_segment%06d.m4s")
	initFilename := extID + "_init.mp4"
	args := buildLiveHlsCaptureFFmpegArgs(
		server.URL+"/src.m3u8", playlistPath, segmentPattern, initFilename,
		false, "-c:v copy -c:a copy",
	)
	joined := strings.Join(args, " ")
	for _, want := range []string{"-hls_segment_type", "fmp4", "-bsf:a", "aac_adtstoasc", "temp_file", ".m4s"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("capture args missing %q: %v", want, args)
		}
	}
	run("ffmpeg", args...)

	// 3. Segments, init, and playlist with MAP + segment references.
	for _, pattern := range []string{extID + "_segment*.m4s", extID + "_init.mp4"} {
		matches, err := filepath.Glob(filepath.Join(capDir, pattern))
		if err != nil || len(matches) == 0 {
			t.Fatalf("expected captured files for %q, got %v, err %v", pattern, matches, err)
		}
	}
	playlistBytes, err := os.ReadFile(playlistPath)
	if err != nil {
		t.Fatalf("read capture playlist: %v", err)
	}
	for _, want := range []string{"#EXT-X-MAP", "#EXTINF"} {
		if !strings.Contains(string(playlistBytes), want) {
			t.Fatalf("capture playlist missing %q:\n%s", want, playlistBytes)
		}
	}

	// 4. Finalize like post-process does.
	if err := hls.FinalizeMediaPlaylist(playlistPath); err != nil {
		t.Fatalf("finalize playlist: %v", err)
	}
	finalized, err := os.ReadFile(playlistPath)
	if err != nil {
		t.Fatalf("read finalized playlist: %v", err)
	}
	for _, want := range []string{"#EXT-X-PLAYLIST-TYPE:VOD", "#EXT-X-ENDLIST"} {
		if !strings.Contains(string(finalized), want) {
			t.Fatalf("finalized playlist missing %q:\n%s", want, finalized)
		}
	}

	// 5. Duration probe.
	duration, err := GetVideoDuration(ctx, playlistPath)
	if err != nil {
		t.Fatalf("probe playlist duration: %v", err)
	}
	if duration <= 0 {
		t.Fatalf("invalid playlist duration %d", duration)
	}

	// 6. MP4 export via the real convert-task function.
	exportPath := filepath.Join(tmpDir, "export.mp4")
	if err := ExportHlsToMp4(ctx, ent.Vod{
		ID:              uuid.New(),
		ExtID:           extID,
		Title:           "e2e live",
		TmpVideoHlsPath: capDir,
	}, exportPath); err != nil {
		t.Fatalf("export HLS to MP4: %v", err)
	}
	info, err := os.Stat(exportPath)
	if err != nil || info.Size() == 0 {
		t.Fatalf("exported MP4 missing or empty: %v %v", info, err)
	}
	probe, err := ProbeMediaDuration(ctx, exportPath)
	if err != nil {
		t.Fatalf("probe exported MP4: %v", err)
	}
	if probe.HasTimestampAnomaly() {
		t.Fatalf("exported MP4 timestamp anomaly: format=%f stream=%f", probe.FormatDuration, probe.LongestStreamDuration)
	}
	run("ffmpeg", "-v", "error", "-i", exportPath, "-f", "null", "-")

	// 7. Truncate the playlist like a mid-rewrite kill, then rebuild it.
	if err := os.WriteFile(playlistPath, []byte{}, 0o644); err != nil {
		t.Fatalf("truncate playlist: %v", err)
	}
	rescued, err := EnsureLiveHlsPlaylist(ctx, capDir, extID)
	if err != nil {
		t.Fatalf("ensure playlist after truncation: %v", err)
	}
	if rescued != playlistPath {
		t.Fatalf("expected rescued playlist %q, got %q", playlistPath, rescued)
	}
	rescuedBytes, err := os.ReadFile(playlistPath)
	if err != nil {
		t.Fatalf("read rescued playlist: %v", err)
	}
	for _, want := range []string{"#EXT-X-MAP", "#EXTINF", "#EXT-X-ENDLIST"} {
		if !strings.Contains(string(rescuedBytes), want) {
			t.Fatalf("rescued playlist missing %q:\n%s", want, rescuedBytes)
		}
	}
	rescuedDuration, err := GetVideoDuration(ctx, playlistPath)
	if err != nil {
		t.Fatalf("probe rescued playlist duration: %v", err)
	}
	if rescuedDuration <= 0 {
		t.Fatalf("invalid rescued playlist duration %d", rescuedDuration)
	}
	// Allow small rounding skew versus the original duration.
	if diff := rescuedDuration - duration; diff < -3 || diff > 3 {
		t.Fatalf("rescued duration %d drifts from original %d", rescuedDuration, duration)
	}
}
