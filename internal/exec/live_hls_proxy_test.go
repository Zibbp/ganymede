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

	"github.com/zibbp/ganymede/internal/hls"
)

func testLiveHLSProxy(t *testing.T) *liveHLSProxy {
	t.Helper()
	return testLiveHLSProxyWithWindow(t, 2*time.Second)
}

func testLiveHLSProxyWithWindow(t *testing.T, window time.Duration) *liveHLSProxy {
	t.Helper()
	return testLiveHLSProxyWithTimeout(t, window, liveHLSRequestTimeout)
}

func testLiveHLSProxyWithTimeout(t *testing.T, window, readTimeout time.Duration) *liveHLSProxy {
	t.Helper()
	p, err := newLiveHLSProxy(t.Context(), "test-video", "test-channel", window, 10*time.Millisecond, readTimeout)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

type liveHLSResponse struct {
	status int
	body   []byte
	err    error
}

func getLiveHLSResource(ctx context.Context, uri string) <-chan liveHLSResponse {
	done := make(chan liveHLSResponse, 1)
	go func() {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
		if err != nil {
			done <- liveHLSResponse{err: err}
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			done <- liveHLSResponse{err: err}
			return
		}
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(resp.Body)
		done <- liveHLSResponse{status: resp.StatusCode, body: body, err: err}
	}()
	return done
}

func awaitLiveHLSResponse(t *testing.T, done <-chan liveHLSResponse) liveHLSResponse {
	t.Helper()
	select {
	case result := <-done:
		return result
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for proxy response")
		return liveHLSResponse{}
	}
}

func proxyResourceURL(t *testing.T, p *liveHLSProxy, origin, ref, kind string) string {
	t.Helper()
	base, err := url.Parse(origin)
	if err != nil {
		t.Fatal(err)
	}
	uri, err := p.resourceURL(base, ref, kind)
	if err != nil {
		t.Fatal(err)
	}
	return uri
}

func TestLiveHLSProxyMediaIdleStartsRecovery(t *testing.T) {
	var attempts atomic.Int32
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		_, _ = w.Write(bytes.Repeat([]byte("x"), 8192))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer source.Close()
	p := testLiveHLSProxyWithTimeout(t, 150*time.Millisecond, 50*time.Millisecond)
	result := awaitLiveHLSResponse(t, getLiveHLSResource(t.Context(), proxyResourceURL(t, p, source.URL, "/segment.ts", "media")))
	if result.err != nil || result.status != http.StatusBadGateway || bytes.Contains(result.body, []byte("xxxx")) {
		t.Fatalf("incomplete media reached client: %+v", result)
	}
	if attempts.Load() < 2 {
		t.Fatal("stalled media was not retried")
	}
	select {
	case <-p.expired:
	default:
		t.Fatal("stalled media did not expire the recovery window")
	}
}

func TestLiveHLSProxyCancelledMediaDoesNotStartRecovery(t *testing.T) {
	started := make(chan struct{})
	finished := make(chan struct{})
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), 8192))
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
		close(finished)
	}))
	defer source.Close()
	p := testLiveHLSProxyWithTimeout(t, 150*time.Millisecond, 2*time.Second)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := getLiveHLSResource(ctx, proxyResourceURL(t, p, source.URL, "/segment.ts", "media"))
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("upstream request did not start")
	}
	cancel()
	if result := awaitLiveHLSResponse(t, done); !errors.Is(result.err, context.Canceled) {
		t.Fatalf("cancelled request: %+v", result)
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("upstream request was not cancelled")
	}
	select {
	case <-p.expired:
		t.Fatal("client cancellation started a recovery window")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestLiveHLSProxyRecoveryDeadlineInterruptsMediaBody(t *testing.T) {
	var attempts atomic.Int32
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			http.Error(w, "temporary outage", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write(bytes.Repeat([]byte("x"), 8192))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer source.Close()
	p := testLiveHLSProxyWithTimeout(t, 200*time.Millisecond, 2*time.Second)
	result := awaitLiveHLSResponse(t, getLiveHLSResource(t.Context(), proxyResourceURL(t, p, source.URL, "/segment.ts", "media")))
	if result.err != nil || result.status != http.StatusBadGateway || bytes.Contains(result.body, []byte("xxxx")) {
		t.Fatalf("expired media reached client: %+v", result)
	}
	if attempts.Load() != 2 {
		t.Fatalf("upstream attempts = %d, want 2", attempts.Load())
	}
}

func TestLiveHLSProxyNewRecoveryInterruptsExistingMediaBody(t *testing.T) {
	started := make(chan struct{})
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/init" {
			http.Error(w, "temporary outage", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write(bytes.Repeat([]byte("x"), 8192))
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
	}))
	defer source.Close()
	p := testLiveHLSProxyWithTimeout(t, 200*time.Millisecond, 2*time.Second)
	mediaDone := getLiveHLSResource(t.Context(), proxyResourceURL(t, p, source.URL, "/segment.ts", "media"))
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("media request did not start")
	}
	initDone := getLiveHLSResource(t.Context(), proxyResourceURL(t, p, source.URL, "/init", "init"))
	for _, done := range []<-chan liveHLSResponse{mediaDone, initDone} {
		result := awaitLiveHLSResponse(t, done)
		if result.err != nil || result.status != http.StatusBadGateway || bytes.Contains(result.body, []byte("xxxx")) {
			t.Fatalf("request survived recovery deadline: %+v", result)
		}
	}
}

func TestLiveHLSProxySuccessfulResourcesResetRecoveryWindow(t *testing.T) {
	for _, kind := range []string{"media", "init", "playlist"} {
		t.Run(kind, func(t *testing.T) {
			var attempts atomic.Int32
			source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if attempts.Add(1)%2 == 1 {
					http.Error(w, "temporary outage", http.StatusServiceUnavailable)
					return
				}
				_, _ = io.WriteString(w, "#EXTM3U\n")
			}))
			defer source.Close()
			p := testLiveHLSProxyWithTimeout(t, 150*time.Millisecond, time.Second)
			base, _ := url.Parse(source.URL)
			resource, err := p.resourceURL(base, "/resource", kind)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 2; i++ {
				resp, err := http.Get(resource)
				if err != nil {
					t.Fatal(err)
				}
				body, err := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				if err != nil || string(body) != "#EXTM3U\n" {
					t.Fatalf("recovered media = %q, error = %v", body, err)
				}
				select {
				case <-p.expired:
					t.Fatal("a recovered transfer left the recovery timer active")
				case <-time.After(200 * time.Millisecond):
				}
			}
			if attempts.Load() != 4 {
				t.Fatalf("upstream attempts = %d, want 4", attempts.Load())
			}
		})
	}
}

func TestLiveHLSProxyActiveMediaCanExceedRecoveryWindow(t *testing.T) {
	const chunks = 8
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for i := 0; i < chunks; i++ {
			if _, err := w.Write([]byte("chunk")); err != nil {
				return
			}
			w.(http.Flusher).Flush()
			time.Sleep(30 * time.Millisecond)
		}
	}))
	defer source.Close()
	p := testLiveHLSProxyWithTimeout(t, 100*time.Millisecond, 150*time.Millisecond)
	base, _ := url.Parse(source.URL)
	resource, err := p.resourceURL(base, "/segment.ts", "media")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(resource)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil || string(body) != strings.Repeat("chunk", chunks) {
		t.Fatalf("active media body = %q, error = %v", body, err)
	}
	select {
	case <-p.expired:
		t.Fatal("active media transfer expired")
	default:
	}
}

func TestLiveHLSProxyUsesSafeContentTypes(t *testing.T) {
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, "#EXTM3U\n# <script>alert(1)</script>\n")
	}))
	defer source.Close()
	p := testLiveHLSProxy(t)
	base, _ := url.Parse(source.URL)
	for _, tc := range []struct{ kind, contentType string }{
		{kind: "playlist", contentType: "application/vnd.apple.mpegurl"},
		{kind: "init", contentType: "application/octet-stream"},
		{kind: "media", contentType: "application/octet-stream"},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			resource, err := p.resourceURL(base, "/resource", tc.kind)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := http.Get(resource)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = resp.Body.Close() }()
			if _, err := io.Copy(io.Discard, resp.Body); err != nil {
				t.Fatal(err)
			}
			if got := resp.Header.Get("Content-Type"); got != tc.contentType {
				t.Errorf("Content-Type = %q, want %q", got, tc.contentType)
			}
			if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
			}
		})
	}
}

func TestLiveHLSProxyRecoversStalledMediaDuringCapture(t *testing.T) {
	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "source")
	outputDir := filepath.Join(dir, "capture")
	for _, path := range []string{sourceDir, outputDir} {
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	generate := osExec.CommandContext(t.Context(), "ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc=size=160x120:rate=10:duration=12",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=12",
		"-c:v", "libx264", "-preset", "ultrafast", "-g", "20", "-c:a", "aac",
		"-hls_time", "2", "-hls_list_size", "0", "-hls_playlist_type", "vod",
		"-hls_segment_filename", filepath.Join(sourceDir, "seg%03d.ts"), "-f", "hls", filepath.Join(sourceDir, "source.m3u8"))
	if out, err := generate.CombinedOutput(); err != nil {
		t.Fatalf("generate source: %v\n%s", err, out)
	}
	var interrupted atomic.Bool
	fileServer := http.FileServer(http.Dir(sourceDir))
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/seg003.ts" && interrupted.CompareAndSwap(false, true) {
			segment, err := os.ReadFile(filepath.Join(sourceDir, filepath.Base(r.URL.Path)))
			if err != nil {
				t.Errorf("read segment: %v", err)
				return
			}
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Errorf("hijack segment response: %v", err)
				return
			}
			defer func() { _ = conn.Close() }()
			_, _ = fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Length: %d\r\nContent-Type: video/mp2t\r\n\r\n", len(segment))
			_, _ = conn.Write(segment[:min(len(segment)/2, 32768)])
			time.Sleep(500 * time.Millisecond)
			return
		}
		fileServer.ServeHTTP(w, r)
	}))
	defer source.Close()
	p := testLiveHLSProxyWithTimeout(t, 2*time.Second, 100*time.Millisecond)
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
	if !interrupted.Load() {
		t.Fatal("capture did not exercise a stalled media body")
	}
	probe, err := ProbeMediaDuration(t.Context(), output)
	if err != nil {
		t.Fatal(err)
	}
	if probe.FormatDuration < 11 || probe.FormatDuration > 13 {
		t.Fatalf("capture duration = %f, want approximately 12 seconds", probe.FormatDuration)
	}
	decode := osExec.CommandContext(t.Context(), "ffmpeg", "-v", "error", "-i", output, "-f", "null", "-")
	if out, err := decode.CombinedOutput(); err != nil {
		t.Fatalf("decode capture: %v\n%s", err, out)
	}
	for _, media := range []string{filepath.Join(sourceDir, "source.m3u8"), output} {
		frames, err := osExec.CommandContext(t.Context(), "ffprobe", "-v", "error", "-count_frames", "-select_streams", "v:0", "-show_entries", "stream=nb_read_frames", "-of", "csv=p=0", media).CombinedOutput()
		if err != nil {
			t.Fatalf("count frames: %v\n%s", err, frames)
		}
		counts := strings.Fields(string(frames))
		if len(counts) == 0 {
			t.Fatal("no video frames counted")
		}
		for _, count := range counts {
			if count != "120" {
				t.Fatalf("%s: decoded %s video frames, want 120", media, count)
			}
		}
	}

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
	defer func() { _ = resp.Body.Close() }()
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
		_ = resp.Body.Close()
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
			_ = resp.Body.Close()
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
	_ = resp.Body.Close()
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
	defer func() { _ = resp.Body.Close() }()
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
	t.Logf("captured %.2f seconds across initialization retries", probe.FormatDuration)

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
	err = superviseLiveArchiveCommand(ctx, cmd, done, nil, 0, failedProxy)
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

func TestLiveHLSProxyRetriesCompleteMedia(t *testing.T) {
	for _, interruption := range []string{"truncated", "chunked", "idle"} {
		t.Run(interruption, func(t *testing.T) {
			payload := bytes.Repeat([]byte("complete media"), 4096)
			var attempts atomic.Int32
			source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Range") != "bytes=100-" {
					t.Errorf("range = %q, want bytes=100-", r.Header.Get("Range"))
				}
				attempt := attempts.Add(1)
				w.Header().Set("Content-Range", fmt.Sprintf("bytes 100-%d/%d", len(payload)+99, len(payload)+100))
				if interruption != "chunked" {
					w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
				}
				w.WriteHeader(http.StatusPartialContent)
				if attempt == 1 {
					_, _ = w.Write(payload[:8192])
					w.(http.Flusher).Flush()
					switch interruption {
					case "idle":
						<-r.Context().Done()
					case "chunked":
						panic(http.ErrAbortHandler)
					}
					return
				}
				_, _ = w.Write(payload)
			}))
			defer source.Close()
			p := testLiveHLSProxyWithTimeout(t, time.Second, 50*time.Millisecond)
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, proxyResourceURL(t, p, source.URL, "/segment.ts", "media"), nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Range", "bytes=100-")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = resp.Body.Close() }()
			body, err := io.ReadAll(resp.Body)
			if err != nil || !bytes.Equal(body, payload) {
				t.Fatalf("media differs after retry: bytes=%d, error=%v", len(body), err)
			}
			if resp.StatusCode != http.StatusPartialContent || resp.Header.Get("Content-Range") != fmt.Sprintf("bytes 100-%d/%d", len(payload)+99, len(payload)+100) || resp.ContentLength != int64(len(payload)) {
				t.Fatalf("range response changed: status=%d headers=%v", resp.StatusCode, resp.Header)
			}
			if attempts.Load() != 2 || p.recovering() {
				t.Fatalf("attempts=%d, recovering=%v", attempts.Load(), p.recovering())
			}
		})
	}
}

func TestLiveHLSProxyConcurrentSuccessDoesNotClearFailure(t *testing.T) {
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing.ts" {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		_, _ = io.WriteString(w, "complete")
	}))
	defer source.Close()
	p := testLiveHLSProxyWithWindow(t, 250*time.Millisecond)
	failed := getLiveHLSResource(t.Context(), proxyResourceURL(t, p, source.URL, "/missing.ts", "media"))
	select {
	case <-p.changed:
	case <-time.After(time.Second):
		t.Fatal("recovery did not start")
	}
	result := awaitLiveHLSResponse(t, getLiveHLSResource(t.Context(), proxyResourceURL(t, p, source.URL, "/healthy.ts", "media")))
	if result.err != nil || result.status != http.StatusOK || string(result.body) != "complete" {
		t.Fatalf("healthy resource: %+v", result)
	}
	if !p.recovering() {
		t.Fatal("concurrent success cleared the outstanding failure")
	}
	result = awaitLiveHLSResponse(t, failed)
	if result.err != nil || result.status != http.StatusBadGateway {
		t.Fatalf("outstanding request did not expire: %+v", result)
	}
}

func TestLiveHLSProxyRejectsOversizedResource(t *testing.T) {
	var attempts atomic.Int32
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.Header().Set("Content-Length", fmt.Sprint(liveHLSMaxResourceSize+1))
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer source.Close()
	p := testLiveHLSProxy(t)
	result := awaitLiveHLSResponse(t, getLiveHLSResource(t.Context(), proxyResourceURL(t, p, source.URL, "/oversized.ts", "media")))
	if result.err != nil || result.status != http.StatusBadGateway || attempts.Load() != 1 {
		t.Fatalf("oversized resource: %+v, attempts=%d", result, attempts.Load())
	}
}

func TestLiveHLSProxyBoundsChunkedPlaylist(t *testing.T) {
	var attempts atomic.Int32
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		_, _ = w.Write(bytes.Repeat([]byte("x"), hls.MaxPlaylistSize+1))
	}))
	defer source.Close()
	p := testLiveHLSProxy(t)
	result := awaitLiveHLSResponse(t, getLiveHLSResource(t.Context(), proxyResourceURL(t, p, source.URL, "/oversized.m3u8", "playlist")))
	if result.err != nil || result.status != http.StatusBadGateway || attempts.Load() != 1 {
		t.Fatalf("oversized chunked playlist: %+v, attempts=%d", result, attempts.Load())
	}
}

func TestLiveHLSProxyReportsFirstMediaProgramTime(t *testing.T) {
	t.Parallel()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/live.m3u8":
			_, _ = io.WriteString(w, "#EXTM3U\n#EXT-X-TARGETDURATION:2\n#EXT-X-MEDIA-SEQUENCE:7\n"+
				"#EXT-X-PROGRAM-DATE-TIME:2026-11-22T20:00:00.000Z\n#EXTINF:2.000,\nseg7.ts\n"+
				"#EXTINF:2.000,\nseg8.ts\n")
		default:
			_, _ = io.WriteString(w, "media")
		}
	}))
	defer upstream.Close()

	p := testLiveHLSProxy(t)
	got := make(chan time.Time, 2)
	p.setFirstMediaHook(func(t time.Time) { got <- t })

	playlist := awaitLiveHLSResponse(t, getLiveHLSResource(t.Context(), proxyResourceURL(t, p, upstream.URL+"/live.m3u8", upstream.URL+"/live.m3u8", "playlist")))
	if playlist.err != nil || playlist.status != http.StatusOK {
		t.Fatalf("playlist: status=%d err=%v", playlist.status, playlist.err)
	}
	for _, seg := range []string{"seg8.ts", "seg7.ts"} {
		res := awaitLiveHLSResponse(t, getLiveHLSResource(t.Context(), proxyResourceURL(t, p, upstream.URL+"/live.m3u8", seg, "media")))
		if res.err != nil || res.status != http.StatusOK {
			t.Fatalf("%s: status=%d err=%v", seg, res.status, res.err)
		}
	}

	want := time.Date(2026, 11, 22, 20, 0, 2, 0, time.UTC)
	select {
	case first := <-got:
		if !first.Equal(want) {
			t.Fatalf("first media program time = %s, want %s", first, want)
		}
	default:
		t.Fatal("first media hook was not called")
	}
	select {
	case extra := <-got:
		t.Fatalf("hook called more than once (%s)", extra)
	default:
	}
}

func TestParseHLSProgramDateTime(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"2026-11-22T20:00:00.123Z", "2026-11-22T21:00:00.123+0100", "2026-11-22T21:00:00.123+01:00"} {
		got, err := parseHLSProgramDateTime(value)
		if err != nil {
			t.Fatalf("%s: %v", value, err)
		}
		if want := time.Date(2026, 11, 22, 20, 0, 0, 123e6, time.UTC); !got.Equal(want) {
			t.Fatalf("%s: got %s want %s", value, got, want)
		}
	}
	if _, err := parseHLSProgramDateTime("garbage"); err == nil {
		t.Fatal("expected error for invalid value")
	}
}
