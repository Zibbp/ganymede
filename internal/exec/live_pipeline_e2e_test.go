package exec

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	osExec "os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/hls"
)

// TestExportHlsToMp4PreservesLateStreamStart covers captures whose audio and
// video timelines start several seconds apart.
func TestExportHlsToMp4PreservesLateStreamStart(t *testing.T) {
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

	// 30s A/V content with a 12s initial video gap.
	srcMP4 := filepath.Join(tmpDir, "src.mp4")
	run("ffmpeg",
		"-y", "-hide_banner",
		"-f", "lavfi", "-i", "testsrc=size=160x120:rate=10:duration=30",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000:duration=30",
		"-filter_complex", "[0:v]setpts=PTS+12/TB[v]",
		"-map", "[v]", "-map", "1:a",
		"-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac",
		"-fps_mode", "passthrough",
		srcMP4,
	)

	const extID = "latestart123"
	hlsDir := filepath.Join(tmpDir, "hls")
	if err := os.MkdirAll(hlsDir, 0o755); err != nil {
		t.Fatalf("create hls dir: %v", err)
	}
	run("ffmpeg",
		"-y", "-hide_banner", "-i", srcMP4,
		"-c", "copy",
		"-start_number", "0", "-hls_time", "10", "-hls_list_size", "0",
		"-hls_playlist_type", "vod",
		"-hls_segment_type", "fmp4",
		"-hls_fmp4_init_filename", extID+"_init.mp4",
		"-hls_segment_filename", filepath.Join(hlsDir, extID+"_segment%06d.m4s"),
		"-f", "hls", filepath.Join(hlsDir, extID+"-video.m3u8"),
	)
	inputOffset := mediaAVStartOffset(t, filepath.Join(hlsDir, extID+"-video.m3u8"))
	if math.Abs(inputOffset) < 0.5 {
		t.Fatalf("HLS fixture must have separated A/V starts, got offset %f", inputOffset)
	}

	exportPath := filepath.Join(tmpDir, "export.mp4")
	if err := ExportHlsToMp4(ctx, ent.Vod{
		ID:              uuid.New(),
		ExtID:           extID,
		Title:           "late stream start",
		TmpVideoHlsPath: hlsDir,
	}, exportPath); err != nil {
		t.Fatalf("export HLS to MP4: %v", err)
	}

	probe, err := ProbeMediaDuration(ctx, exportPath)
	if err != nil {
		t.Fatalf("probe exported MP4: %v", err)
	}
	if probe.HasTimestampAnomaly() {
		t.Fatalf("exported MP4 still has a timestamp anomaly: format=%f stream=%f max_start=%f",
			probe.FormatDuration, probe.LongestStreamDuration, probe.MaxStreamStartTime)
	}
	if delta := mediaAVStartOffset(t, exportPath) - inputOffset; math.Abs(delta) > 0.05 {
		t.Fatalf("HLS export changed the A/V start offset by %f seconds", delta)
	}
	if want := probe.LongestStreamDuration + math.Abs(inputOffset); math.Abs(probe.FormatDuration-want) > 1 {
		t.Fatalf("exported duration = %f, want about %f seconds including the initial gap", probe.FormatDuration, want)
	}

	run("ffmpeg", "-v", "error", "-i", exportPath, "-f", "null", "-")
}

func TestHlsExportCorrectsTimestampDiscontinuities(t *testing.T) {
	for _, tc := range []struct {
		name        string
		firstStart  int
		secondStart int
	}{
		{name: "forward jump", secondStart: 20030},
		{name: "clock reset", secondStart: 0},
		{name: "shared offset and jump", firstStart: 5, secondStart: 20035},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for i, offset := range []int{tc.firstStart, tc.secondStart} {
				cmd := osExec.CommandContext(t.Context(), "ffmpeg", "-v", "error", "-y",
					"-f", "lavfi", "-i", "testsrc=size=160x120:rate=10:duration=30",
					"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000:duration=30",
					"-filter_complex", "[0:v]setpts=PTS+1/TB[v]",
					"-map", "[v]", "-map", "1:a",
					"-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p", "-c:a", "aac",
					"-fps_mode", "passthrough", "-output_ts_offset", fmt.Sprint(offset),
					"-muxdelay", "0", "-f", "mpegts", filepath.Join(dir, fmt.Sprintf("%d.ts", i)))
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("create HLS segment: %v\n%s", err, out)
				}
			}
			playlist := filepath.Join(dir, "input.m3u8")
			if err := os.WriteFile(playlist, []byte("#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:42\n"+
				"#EXT-X-MEDIA-SEQUENCE:0\n#EXTINF:30,\n0.ts\n#EXT-X-DISCONTINUITY\n#EXTINF:30,\n1.ts\n#EXT-X-ENDLIST\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			first, err := ProbeMediaDuration(t.Context(), filepath.Join(dir, "0.ts"))
			if err != nil {
				t.Fatal(err)
			}
			second, err := ProbeMediaDuration(t.Context(), filepath.Join(dir, "1.ts"))
			if err != nil {
				t.Fatal(err)
			}
			if math.Abs(second.MinStreamStartTime-first.MinStreamStartTime-30) < 1 {
				t.Fatal("fixture does not contain a timestamp discontinuity")
			}
			starts, err := probeMediaStreamStarts(t.Context(), playlist)
			if err != nil {
				t.Fatal(err)
			}
			inputOffset := mediaAVStartOffset(t, playlist)
			for _, normalized := range []bool{false, true} {
				t.Run(fmt.Sprintf("normalized=%v", normalized), func(t *testing.T) {
					exportPath := filepath.Join(dir, fmt.Sprintf("export-%v.mp4", normalized))
					tmpPath, commit, cleanup, err := tempExportTarget(exportPath)
					if err != nil {
						t.Fatal(err)
					}
					defer cleanup()
					var output bytes.Buffer
					passes := 0
					err = exportWithTimestampRepair(t.Context(), "test", tmpPath, func(repair bool) error {
						passes++
						args := buildHlsToMp4FFmpegArgs(ent.Vod{}, playlist, tmpPath, "", normalized || repair, starts)
						return runHlsToMp4FFmpeg(t.Context(), ent.Vod{}, args, tmpPath, &output)
					}, commit, &output)
					if err != nil {
						t.Fatalf("export HLS discontinuity: %v\n%s", err, output.String())
					}
					if tc.firstStart > 0 && !normalized && passes != 2 {
						t.Fatalf("shared start offset should require a repair pass, got %d passes", passes)
					}
					probe, err := ProbeMediaDuration(t.Context(), exportPath)
					if err != nil {
						t.Fatal(err)
					}
					if probe.HasTimestampAnomaly() || probe.FormatDuration < 60 || probe.FormatDuration > 62 {
						t.Fatalf("expected about 61 seconds including the preserved initial gap: %+v", probe)
					}
					if delta := mediaAVStartOffset(t, exportPath) - inputOffset; math.Abs(delta) > 0.05 {
						t.Fatalf("discontinuity correction changed the A/V offset by %f seconds", delta)
					}
					decode := osExec.CommandContext(t.Context(), "ffmpeg", "-v", "error", "-i", exportPath, "-f", "null", "-")
					if out, err := decode.CombinedOutput(); err != nil {
						t.Fatalf("decode exported HLS: %v\n%s", err, out)
					}
				})
			}
		})
	}
}

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
	// Drop one segment connection before its response, as can happen when
	// a backup interruption breaks a request. HLS must retry that segment.
	var interrupted atomic.Bool
	fileServer := http.FileServer(http.Dir(srcHLS))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".ts") && interrupted.CompareAndSwap(false, true) {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Errorf("hijack segment response: %v", err)
				return
			}
			_ = conn.Close()
			return
		}
		fileServer.ServeHTTP(w, r)
	}))
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
		false, "-c:v copy -c:a copy", false,
	)
	joined := strings.Join(args, " ")
	for _, want := range []string{"-hls_segment_type", "fmp4", "-bsf:a", "aac_adtstoasc", "temp_file", ".m4s"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("capture args missing %q: %v", want, args)
		}
	}
	run("ffmpeg", args...)
	if !interrupted.Load() {
		t.Fatal("capture did not exercise the interrupted segment response")
	}

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
	if duration < 11 || duration > 13 {
		t.Fatalf("capture lost media after interrupted read: duration %d, want approximately 12 seconds", duration)
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
