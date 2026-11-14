package exec

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/hls"
)

const (
	liveHLSRetryWindow    = 45 * time.Second
	liveHLSRetryInterval  = time.Second
	liveHLSRequestTimeout = 8 * time.Second
	liveHLSMaxInitSize    = 64 << 20
)

var liveHLSURIAttribute = regexp.MustCompile(`URI="([^"]*)"`)
var liveHLSExtension = regexp.MustCompile(`^\.[a-zA-Z0-9]{1,9}$`)

// liveHLSProxy keeps FFmpeg's HLS demuxer alive while an upstream request is
// temporarily unavailable. In particular FFmpeg may treat a failed EXT-X-MAP
// request as a clean end of input even when its reconnect options are enabled.
type liveHLSProxy struct {
	server        *http.Server
	listener      net.Listener
	client        *http.Client
	logger        zerolog.Logger
	token         string
	retryWindow   time.Duration
	retryInterval time.Duration
	readTimeout   time.Duration
	mu            sync.Mutex
	closed        bool
	failedSince   time.Time
	failureTimer  *time.Timer
	expired       chan struct{}
	expireOnce    sync.Once
	expiryContext context.Context
	cancelExpiry  context.CancelFunc
}

func newLiveHLSProxy(ctx context.Context, videoID, channel string, retryWindow, retryInterval, readTimeout time.Duration) (*liveHLSProxy, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for live HLS recovery: %w", err)
	}
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		listener.Close()
		return nil, fmt.Errorf("create live HLS recovery token: %w", err)
	}
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = defaultTransport.Clone()
	}
	transport.ResponseHeaderTimeout = 5 * time.Second
	transport.TLSHandshakeTimeout = 5 * time.Second
	transport.DialContext = (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	expiryContext, cancelExpiry := context.WithCancel(ctx)
	p := &liveHLSProxy{
		listener:      listener,
		client:        &http.Client{Transport: transport},
		logger:        log.With().Str("video_id", videoID).Str("channel", channel).Logger(),
		token:         hex.EncodeToString(token[:]),
		retryWindow:   retryWindow,
		retryInterval: retryInterval,
		readTimeout:   readTimeout,
		expired:       make(chan struct{}),
		expiryContext: expiryContext,
		cancelExpiry:  cancelExpiry,
	}
	p.server = &http.Server{Handler: p, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := p.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			p.logger.Error().Err(err).Msg("live HLS recovery server stopped")
		}
	}()
	return p, nil
}

func (p *liveHLSProxy) Close() {
	p.mu.Lock()
	p.closed = true
	p.failedSince = time.Time{}
	if p.failureTimer != nil {
		p.failureTimer.Stop()
		p.failureTimer = nil
	}
	p.mu.Unlock()
	p.cancelExpiry()
	_ = p.server.Close()
	p.client.CloseIdleConnections()
}

func (p *liveHLSProxy) resourceURL(base *url.URL, ref, kind string) (string, error) {
	u, err := url.Parse(ref)
	if err != nil {
		return "", err
	}
	u = base.ResolveReference(u)
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("unsupported live HLS resource URL")
	}
	ext := path.Ext(u.Path)
	if !liveHLSExtension.MatchString(ext) {
		ext = ".ts"
	}
	if kind == "playlist" {
		ext = ".m3u8"
	}
	return fmt.Sprintf("http://%s/%s/%s/%s%s", p.listener.Addr(), p.token, kind,
		base64.RawURLEncoding.EncodeToString([]byte(u.String())), ext), nil
}

func (p *liveHLSProxy) playlistURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	return p.resourceURL(u, raw, "playlist")
}

func (p *liveHLSProxy) rewritePlaylist(body []byte, base *url.URL) ([]byte, error) {
	lines := strings.Split(string(body), "\n")
	playlistNext := false
	for i, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			continue
		}
		if line[0] != '#' {
			kind := "media"
			if playlistNext {
				kind = "playlist"
				playlistNext = false
			}
			rewritten, err := p.resourceURL(base, line, kind)
			if err != nil {
				return nil, err
			}
			lines[i] = rewritten
			continue
		}
		kind := "media"
		if strings.HasPrefix(line, "#EXT-X-MAP:") {
			kind = "init"
		} else if strings.HasPrefix(line, "#EXT-X-MEDIA:") || strings.HasPrefix(line, "#EXT-X-I-FRAME-STREAM-INF:") || strings.HasPrefix(line, "#EXT-X-RENDITION-REPORT:") {
			kind = "playlist"
		} else if strings.HasPrefix(line, "#EXT-X-STREAM-INF:") {
			playlistNext = true
		}
		var rewriteErr error
		lines[i] = liveHLSURIAttribute.ReplaceAllStringFunc(line, func(attr string) string {
			ref := strings.TrimSuffix(strings.TrimPrefix(attr, `URI="`), `"`)
			rewritten, err := p.resourceURL(base, ref, kind)
			if err != nil {
				rewriteErr = err
				return attr
			}
			return `URI="` + rewritten + `"`
		})
		if rewriteErr != nil {
			return nil, rewriteErr
		}
	}
	return []byte(strings.Join(lines, "\n")), nil
}

func (p *liveHLSProxy) failureDeadline() time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return time.Now()
	}
	if p.failedSince.IsZero() {
		p.failedSince = time.Now()
		start := p.failedSince
		p.failureTimer = time.AfterFunc(p.retryWindow, func() {
			p.mu.Lock()
			defer p.mu.Unlock()
			if p.failedSince.Equal(start) {
				p.expireLocked()
			}
		})
	}
	return p.failedSince.Add(p.retryWindow)
}

func (p *liveHLSProxy) expireLocked() {
	p.expireOnce.Do(func() {
		p.logger.Warn().Dur("retry_window", p.retryWindow).Msg("live HLS recovery timed out")
		close(p.expired)
		p.cancelExpiry()
	})
}

func (p *liveHLSProxy) expireIfOverdue() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.failedSince.IsZero() && !time.Now().Before(p.failedSince.Add(p.retryWindow)) {
		p.expireLocked()
		return true
	}
	return false
}

func (p *liveHLSProxy) currentDeadline() time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.failedSince.IsZero() {
		return time.Time{}
	}
	return p.failedSince.Add(p.retryWindow)
}

func (p *liveHLSProxy) clearFailure() {
	p.mu.Lock()
	start := p.failedSince
	if !start.IsZero() && !time.Now().Before(start.Add(p.retryWindow)) {
		p.expireLocked()
		p.mu.Unlock()
		return
	}
	if p.failureTimer != nil {
		p.failureTimer.Stop()
		p.failureTimer = nil
	}
	p.failedSince = time.Time{}
	p.mu.Unlock()
	if !start.IsZero() {
		p.logger.Info().Dur("interruption", time.Since(start)).Msg("live HLS requests recovered")
	}
}

func (p *liveHLSProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if r.Method != http.MethodGet || len(parts) != 3 || parts[0] != p.token || (parts[1] != "playlist" && parts[1] != "init" && parts[1] != "media") {
		http.NotFound(w, r)
		return
	}
	encoded := strings.TrimSuffix(parts[2], path.Ext(parts[2]))
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		http.Error(w, "invalid resource", http.StatusBadRequest)
		return
	}
	u, err := url.Parse(string(decoded))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		http.Error(w, "invalid resource", http.StatusBadRequest)
		return
	}
	warned := false
	for {
		if p.expireIfOverdue() {
			http.Error(w, "live HLS recovery timed out", http.StatusBadGateway)
			return
		}
		deadline := p.currentDeadline()
		attemptCtx := r.Context()
		cancel := func() {}
		stopExpiry := func() bool { return true }
		if parts[1] != "media" {
			attemptDeadline := time.Now().Add(liveHLSRequestTimeout)
			if !deadline.IsZero() && deadline.Before(attemptDeadline) {
				attemptDeadline = deadline
			}
			attemptCtx, cancel = context.WithDeadline(attemptCtx, attemptDeadline)
		} else {
			attemptCtx, cancel = context.WithCancel(attemptCtx)
			stopExpiry = context.AfterFunc(p.expiryContext, cancel)
		}
		req, err := http.NewRequestWithContext(attemptCtx, http.MethodGet, u.String(), nil)
		if err != nil {
			stopExpiry()
			cancel()
			http.Error(w, "invalid resource", http.StatusBadRequest)
			return
		}
		if rangeHeader := r.Header.Get("Range"); rangeHeader != "" && parts[1] != "playlist" {
			req.Header.Set("Range", rangeHeader)
		}
		if userAgent := r.Header.Get("User-Agent"); userAgent != "" {
			req.Header.Set("User-Agent", userAgent)
		}
		resp, err := p.client.Do(req)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if parts[1] == "playlist" || parts[1] == "init" {
				limit := int64(liveHLSMaxInitSize)
				if parts[1] == "playlist" {
					limit = hls.MaxPlaylistSize
				}
				body, readErr := io.ReadAll(io.LimitReader(resp.Body, limit+1))
				_ = resp.Body.Close()
				cancel()
				if readErr == nil && int64(len(body)) > limit {
					http.Error(w, "live HLS resource too large", http.StatusBadGateway)
					return
				}
				if readErr == nil && parts[1] == "playlist" {
					body, readErr = p.rewritePlaylist(body, resp.Request.URL)
				}
				if readErr == nil {
					copyLiveHLSHeaders(w.Header(), resp.Header)
					status := resp.StatusCode
					if parts[1] == "playlist" {
						w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
						w.Header().Del("Content-Range")
						w.Header().Del("Accept-Ranges")
						status = http.StatusOK
					} else {
						w.Header().Set("Content-Type", "application/octet-stream")
					}
					w.WriteHeader(status)
					_, _ = w.Write(body)
					return
				}
				err = readErr
			} else {
				copyLiveHLSHeaders(w.Header(), resp.Header)
				w.Header().Set("Content-Type", "application/octet-stream")
				w.WriteHeader(resp.StatusCode)
				reader := &liveHLSIdleReader{reader: resp.Body, timeout: p.readTimeout, cancel: cancel}
				_, copyErr := io.Copy(w, reader)
				_ = resp.Body.Close()
				stopExpiry()
				cancel()
				if copyErr != nil {
					if reader.readErr != nil && r.Context().Err() == nil && p.expiryContext.Err() == nil {
						p.failureDeadline()
						p.logger.Warn().Str("resource_type", "media").Dur("retry_window", p.retryWindow).Msg("live HLS media transfer interrupted")
					}
					// Do not let a truncated chunked response look complete to FFmpeg.
					panic(http.ErrAbortHandler)
				}
				p.clearFailure()
				return
			}
		} else if err == nil {
			_ = resp.Body.Close()
			stopExpiry()
			cancel()
			if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusRequestTimeout && resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
				w.WriteHeader(resp.StatusCode)
				return
			}
			err = fmt.Errorf("upstream HTTP status %d", resp.StatusCode)
		} else {
			stopExpiry()
			cancel()
		}
		if r.Context().Err() != nil {
			return
		}
		deadline = p.failureDeadline()
		if !warned {
			p.logger.Warn().Str("resource_type", parts[1]).Dur("retry_window", p.retryWindow).Msg("live HLS request failed; retrying")
			warned = true
		}
		delay := min(p.retryInterval, time.Until(deadline))
		if delay <= 0 {
			continue
		}
		timer := time.NewTimer(delay)
		select {
		case <-r.Context().Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// liveHLSIdleReader cancels only while an upstream read makes no progress;
// it does not impose a total duration limit on an active media transfer.
type liveHLSIdleReader struct {
	reader  io.Reader
	timeout time.Duration
	cancel  context.CancelFunc
	readErr error
}

func (r *liveHLSIdleReader) Read(buf []byte) (int, error) {
	timer := time.AfterFunc(r.timeout, r.cancel)
	defer timer.Stop()
	n, err := r.reader.Read(buf)
	if err != nil && err != io.EOF {
		r.readErr = err
	}
	return n, err
}

func copyLiveHLSHeaders(dst, src http.Header) {
	for _, name := range []string{"Content-Range", "Accept-Ranges"} {
		if value := src.Get(name); value != "" {
			dst.Set(name, value)
		}
	}
}
