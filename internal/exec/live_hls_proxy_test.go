package exec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	osExec "os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testLiveHLSProxy(t *testing.T) *liveHLSProxy {
	t.Helper()
	return testLiveHLSProxyWithWindow(t, 2*time.Second)
}

func testLiveHLSProxyWithWindow(t *testing.T, window time.Duration) *liveHLSProxy {
	t.Helper()
	p, err := newLiveHLSProxy("test-video", "test-channel", window, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

func TestLiveHLSProxyRetriesInitializationWithRange(t *testing.T) {
	var attempts atomic.Int32
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "bytes=10-13" {
			t.Errorf("range = %q", r.Header.Get("Range"))
		}
		if attempts.Add(1) <= 2 {
			http.Error(w, "not available yet", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Range", "bytes 10-13/100")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = io.WriteString(w, "init")
	}))
	defer source.Close()
	p := testLiveHLSProxy(t)
	base, _ := url.Parse(source.URL)
	resource, err := p.resourceURL(base, "/init", "init")
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, resource, nil)
	req.Header.Set("Range", "bytes=10-13")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusPartialContent || string(body) != "init" || resp.Header.Get("Content-Range") != "bytes 10-13/100" || attempts.Load() != 3 {
		t.Fatalf("status=%d body=%q range=%q attempts=%d", resp.StatusCode, body, resp.Header.Get("Content-Range"), attempts.Load())
	}
}

func TestLiveHLSProxyBoundsRetriesAcrossFFmpegRequests(t *testing.T) {
	var attempts atomic.Int32
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer source.Close()
	p := testLiveHLSProxyWithWindow(t, 150*time.Millisecond)
	base, _ := url.Parse(source.URL)
	resource, err := p.resourceURL(base, "/init", "init")
	if err != nil {
		t.Fatal(err)
	}
	var attemptsAfterTimeout int32
	for i := 0; i < 2; i++ {
		if i == 1 {
			resource, err = p.resourceURL(base, "/another-init", "init")
			if err != nil {
				t.Fatal(err)
			}
		}
		resp, err := http.Get(resource)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadGateway {
			t.Fatalf("status = %d", resp.StatusCode)
		}
		if i == 0 {
			attemptsAfterTimeout = attempts.Load()
		} else if attempts.Load() != attemptsAfterTimeout {
			t.Fatal("a second request restarted the recovery window")
		}
	}
	if attempts.Load() < 2 {
		t.Fatalf("expected retry attempts, got %d", attempts.Load())
	}
}

func TestLiveHLSProxyCancellationStopsRetry(t *testing.T) {
	started := make(chan struct{}, 1)
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case started <- struct{}{}:
		default:
		}
		http.Error(w, "temporary failure", http.StatusServiceUnavailable)
	}))
	defer source.Close()
	p := testLiveHLSProxyWithWindow(t, time.Minute)
	base, _ := url.Parse(source.URL)
	resource, err := p.resourceURL(base, "/init", "init")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resource, nil)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		resp, err := http.DefaultClient.Do(req)
		if resp != nil {
			resp.Body.Close()
		}
		done <- err
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled request succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("retry did not stop after cancellation")
	}
}

func TestLiveHLSProxyDoesNotRetryPermanentError(t *testing.T) {
	var attempts atomic.Int32
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer source.Close()
	p := testLiveHLSProxy(t)
	base, _ := url.Parse(source.URL)
	resource, err := p.resourceURL(base, "/init", "init")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(resource)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden || attempts.Load() != 1 {
		t.Fatalf("status=%d attempts=%d", resp.StatusCode, attempts.Load())
	}
}

func TestLiveHLSProxyRewritesPlaylist(t *testing.T) {
	p := testLiveHLSProxy(t)
	base, _ := url.Parse("https://example.test/path/live.m3u8?token=playlist")
	playlist := "#EXTM3U\n#EXT-X-MAP:URI=\"init.mp4?token=init\",BYTERANGE=\"4@10\"\n#EXT-X-KEY:METHOD=AES-128,URI=\"../key\"\n#EXTINF:2,\nseg.m4s?token=segment\n"
	got, err := p.rewritePlaylist([]byte(playlist), base)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "/init/") || !strings.Contains(text, "/media/") || !strings.Contains(text, "BYTERANGE=\"4@10\"") || strings.Contains(text, "example.test") {
		t.Fatalf("unexpected playlist: %s", text)
	}
	master := "#EXTM3U\n#EXT-X-MEDIA:TYPE=AUDIO,URI=\"audio.m3u8\"\n#EXT-X-STREAM-INF:BANDWIDTH=1000\nvideo.m3u8\n"
	got, err = p.rewritePlaylist([]byte(master), base)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(got), "/playlist/") != 2 {
		t.Fatalf("nested playlists were not rewritten: %s", got)
	}
}

func TestLiveHLSProxyFetchesFullPlaylist(t *testing.T) {
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "" {
			t.Errorf("playlist forwarded range %q", r.Header.Get("Range"))
		}
		w.Header().Set("Accept-Ranges", "bytes")
		_, _ = io.WriteString(w, "#EXTM3U\n#EXTINF:2,\nseg.ts\n")
	}))
	defer source.Close()
	p := testLiveHLSProxy(t)
	resource, err := p.playlistURL(source.URL + "/stream.m3u8")
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, resource, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", "bytes=0-")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Accept-Ranges") != "" || !strings.Contains(string(body), "/media/") {
		t.Fatalf("status=%d accept-ranges=%q body=%s", resp.StatusCode, resp.Header.Get("Accept-Ranges"), body)
	}
}

// A changed fMP4 initialization section can become temporarily unavailable
// after media has already been recorded. The same FFmpeg process must retain
// both the earlier and later media so chat timestamps remain aligned.
func TestLiveHLSProxyRecoversInitializationDuringCapture(t *testing.T) {
	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "source")
	outputDir := filepath.Join(dir, "capture")
	for _, path := range []string{sourceDir, outputDir} {
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	generate := osExec.CommandContext(t.Context(), "ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc=size=160x120:rate=10:duration=30",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=30",
		"-c:v", "libx264", "-preset", "ultrafast", "-g", "20", "-c:a", "aac",
		"-hls_time", "2", "-hls_list_size", "0", "-hls_segment_type", "fmp4", "-hls_fmp4_init_filename", "init.mp4",
		"-hls_segment_filename", filepath.Join(sourceDir, "seg%03d.m4s"), "-f", "hls", filepath.Join(sourceDir, "source.m3u8"))
	if out, err := generate.CombinedOutput(); err != nil {
		t.Fatalf("generate source: %v\n%s", err, out)
	}
	path := filepath.Join(sourceDir, "source.m3u8")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(body), "\n")
	segments := 0
	for i, line := range lines {
		if strings.HasPrefix(line, "#EXTINF") {
			segments++
			if segments == 11 {
				lines[i] = "#EXT-X-MAP:URI=\"retry-init.mp4\"\n" + line
			}
		}
	}
	if segments < 11 {
		t.Fatalf("only %d source segments", segments)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	var attempts atomic.Int32
	var permanentFailure atomic.Bool
	fileServer := http.FileServer(http.Dir(sourceDir))
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/retry-init.mp4" {
			if attempts.Add(1) <= 2 || permanentFailure.Load() {
				http.Error(w, "temporary outage", http.StatusServiceUnavailable)
				return
			}
			http.ServeFile(w, r, filepath.Join(sourceDir, "init.mp4"))
			return
		}
		if strings.HasSuffix(r.URL.Path, ".m4s") {
			time.Sleep(20 * time.Millisecond)
		}
		fileServer.ServeHTTP(w, r)
	}))
	defer source.Close()
	p := testLiveHLSProxy(t)
	input, err := p.playlistURL(source.URL + "/source.m3u8")
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(outputDir, "capture-video.m3u8")
	args := buildLiveHlsCaptureFFmpegArgs(input, output, filepath.Join(outputDir, "capture_segment%06d.m4s"), "capture_init.mp4", false, "-c:v copy -c:a copy", false)
	cmd := osExec.CommandContext(t.Context(), "ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("capture: %v\n%s", err, out)
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected recovery after two failures, got %d requests", attempts.Load())
	}
	probe, err := ProbeMediaDuration(t.Context(), output)
	if err != nil {
		t.Fatal(err)
	}
	if probe.FormatDuration < 29 || probe.FormatDuration > 32 {
		t.Fatalf("capture duration = %f, want 30 seconds", probe.FormatDuration)
	}
	decode := osExec.CommandContext(t.Context(), "ffmpeg", "-v", "error", "-i", output, "-f", "null", "-")
	if out, err := decode.CombinedOutput(); err != nil {
		t.Fatalf("decode capture: %v\n%s", err, out)
	}
	t.Log(fmt.Sprintf("captured %.2f seconds across initialization retries", probe.FormatDuration))

	permanentFailure.Store(true)
	failedDir := filepath.Join(dir, "failed-capture")
	if err := os.Mkdir(failedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	failedProxy := testLiveHLSProxyWithWindow(t, 150*time.Millisecond)
	failedInput, err := failedProxy.playlistURL(source.URL + "/source.m3u8")
	if err != nil {
		t.Fatal(err)
	}
	partialPlaylist := filepath.Join(failedDir, "capture-video.m3u8")
	args = buildLiveHlsCaptureFFmpegArgs(failedInput, partialPlaylist, filepath.Join(failedDir, "capture_segment%06d.m4s"), "capture_init.mp4", false, "-c:v copy -c:a copy", false)
	cmd = osExec.Command("ffmpeg", args...)
	var captureLog bytes.Buffer
	cmd.Stderr = &captureLog
	cmd.SysProcAttr = liveArchiveProcessAttributes()
	done, err := startArchiveCommand(cmd)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	err = superviseLiveArchiveCommand(ctx, cmd, done, nil, 0, failedProxy.expired)
	if !errors.Is(err, ErrLiveHLSRecoveryExpired) {
		t.Fatalf("capture error = %v, want recovery timeout", err)
	}
	if _, err := EnsureLiveHlsPlaylist(t.Context(), failedDir, "capture"); err != nil {
		t.Fatalf("finalize partial capture: %v\n%s", err, captureLog.String())
	}
	partial, err := ProbeMediaDuration(t.Context(), partialPlaylist)
	if err != nil {
		entries, _ := os.ReadDir(failedDir)
		t.Fatalf("probe partial capture: %v; files: %v\n%s", err, entries, captureLog.String())
	}
	if partial.FormatDuration < 9 || partial.FormatDuration > 22 {
		t.Fatalf("partial duration = %f, expected recoverable media before the outage", partial.FormatDuration)
	}
	decode = osExec.CommandContext(t.Context(), "ffmpeg", "-v", "error", "-i", partialPlaylist, "-f", "null", "-")
	if out, err := decode.CombinedOutput(); err != nil {
		t.Fatalf("decode partial capture: %v\n%s", err, out)
	}
}
