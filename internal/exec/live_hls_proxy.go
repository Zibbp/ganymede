package exec

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/hls"
)

const (
	liveHLSRetryWindow     = 45 * time.Second
	liveHLSRetryInterval   = time.Second
	liveHLSRequestTimeout  = 8 * time.Second
	liveHLSMaxResourceSize = 64 << 20
)

var errLiveHLSResourceTooLarge = errors.New("live HLS resource too large")

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
	failures      map[string]struct{}
	changed       chan struct{}
	failureTimer  *time.Timer
	expired       chan struct{}
	expireOnce    sync.Once
	expiryContext context.Context
	cancelExpiry  context.CancelFunc

	// onFirstMedia, when set, receives the EXT-X-PROGRAM-DATE-TIME of the first
	// media segment served to FFmpeg, i.e. the wall clock of the first captured frame.
	onFirstMedia   func(time.Time)
	segmentTimes   map[string]time.Time
	firstMediaSeen bool
}

func newLiveHLSProxy(ctx context.Context, videoID, channel string, retryWindow, retryInterval, readTimeout time.Duration) (*liveHLSProxy, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for live HLS recovery: %w", err)
	}
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		_ = listener.Close()
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
		failures:      make(map[string]struct{}),
		changed:       make(chan struct{}, 1),
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
	trackTimes := p.wantsSegmentTimes()
	var segmentTimes map[string]time.Time
	var programTime time.Time
	var segmentDuration time.Duration
	byteRange := false
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
			if trackTimes && kind == "media" {
				if ref, err := url.Parse(line); err == nil {
					if segmentTimes == nil {
						segmentTimes = make(map[string]time.Time)
					}
					// The zero time marks a segment without a usable program time;
					// byte ranges share a URI, so their time is ambiguous.
					segmentTime := programTime
					if byteRange {
						segmentTime = time.Time{}
					}
					segmentTimes[base.ResolveReference(ref).String()] = segmentTime
				}
				if !programTime.IsZero() {
					programTime = programTime.Add(segmentDuration)
				}
			}
			segmentDuration = 0
			byteRange = false
			rewritten, err := p.resourceURL(base, line, kind)
			if err != nil {
				return nil, err
			}
			lines[i] = rewritten
			continue
		}
		if trackTimes {
			if value, ok := strings.CutPrefix(line, "#EXT-X-PROGRAM-DATE-TIME:"); ok {
				if t, err := parseHLSProgramDateTime(value); err == nil {
					programTime = t
				}
			} else if value, ok := strings.CutPrefix(line, "#EXTINF:"); ok {
				seconds, _, _ := strings.Cut(value, ",")
				if f, err := strconv.ParseFloat(strings.TrimSpace(seconds), 64); err == nil && f > 0 {
					segmentDuration = time.Duration(f * float64(time.Second))
				}
			} else if strings.HasPrefix(line, "#EXT-X-BYTERANGE:") {
				byteRange = true
			}
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
	if segmentTimes != nil {
		p.mu.Lock()
		if !p.firstMediaSeen {
			// Keep earlier entries: a refresh may drop a segment that FFmpeg is
			// still downloading.
			if p.segmentTimes == nil {
				p.segmentTimes = make(map[string]time.Time, len(segmentTimes))
			}
			maps.Copy(p.segmentTimes, segmentTimes)
		}
		p.mu.Unlock()
	}
	return []byte(strings.Join(lines, "\n")), nil
}

// setFirstMediaHook must be called before FFmpeg starts requesting media.
func (p *liveHLSProxy) setFirstMediaHook(fn func(time.Time)) {
	p.mu.Lock()
	p.onFirstMedia = fn
	p.mu.Unlock()
}

func (p *liveHLSProxy) wantsSegmentTimes() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.onFirstMedia != nil && !p.firstMediaSeen
}

// recordFirstMedia reports the program time of the first media segment FFmpeg
// received, then stops tracking segment times.
func (p *liveHLSProxy) recordFirstMedia(resource string) {
	p.mu.Lock()
	hook := p.onFirstMedia
	t, isSegment := p.segmentTimes[resource]
	// Keys and other auxiliary resources are not listed as segments.
	if hook == nil || p.firstMediaSeen || !isSegment {
		p.mu.Unlock()
		return
	}
	p.segmentTimes = nil
	p.firstMediaSeen = true
	p.mu.Unlock()
	if !t.IsZero() {
		hook(t)
	}
}

func parseHLSProgramDateTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999Z0700"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid EXT-X-PROGRAM-DATE-TIME %q", value)
}

func (p *liveHLSProxy) failureDeadline(resource string) time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.expiryContext.Err() != nil {
		return time.Now()
	}
	p.failures[resource] = struct{}{}
	if p.failedSince.IsZero() {
		p.failedSince = time.Now()
		p.notifyRecoveryChange()
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

func (p *liveHLSProxy) recovering() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return !p.failedSince.IsZero()
}

func (p *liveHLSProxy) notifyRecoveryChange() {
	select {
	case p.changed <- struct{}{}:
	default:
	}
}

// Only completion of the failed resources ends recovery; concurrent successful
// requests must not extend the deadline of an outstanding failure.
func (p *liveHLSProxy) clearFailure(resource string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.expiryContext.Err() != nil {
		return false
	}
	start := p.failedSince
	if !start.IsZero() && !time.Now().Before(start.Add(p.retryWindow)) {
		p.expireLocked()
		return false
	}
	delete(p.failures, resource)
	if start.IsZero() || len(p.failures) != 0 {
		return true
	}
	p.failureTimer.Stop()
	p.failureTimer = nil
	p.failedSince = time.Time{}
	p.notifyRecoveryChange()
	p.logger.Info().Dur("interruption", time.Since(start)).Msg("live HLS requests recovered")
	return true
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
	// Cancellation and recovery expiry apply to every attempt, including sleeps.
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	stopExpiry := context.AfterFunc(p.expiryContext, cancel)
	defer stopExpiry()
	kind := parts[1]
	resource := kind + ":" + u.String()
	if kind != "playlist" {
		resource += ":" + r.Header.Get("Range")
	}
	warned := false
	for {
		if p.expireIfOverdue() || p.expiryContext.Err() != nil {
			http.Error(w, "live HLS recovery timed out", http.StatusBadGateway)
			return
		}
		if ctx.Err() != nil {
			return
		}
		body, resp, err := p.fetchResource(ctx, r, u.String(), kind)
		if errors.Is(err, errLiveHLSResourceTooLarge) {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if err == nil {
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				if !p.clearFailure(resource) {
					http.Error(w, "live HLS recovery timed out", http.StatusBadGateway)
					return
				}
				status := resp.StatusCode
				if kind == "playlist" {
					w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
					status = http.StatusOK
				} else {
					if kind == "media" {
						p.recordFirstMedia(u.String())
					}
					copyLiveHLSHeaders(w.Header(), resp.Header)
					w.Header().Set("Content-Type", "application/octet-stream")
				}
				w.Header().Set("Content-Length", strconv.Itoa(len(body)))
				w.WriteHeader(status)
				_, _ = w.Write(body)
				return
			}
			if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusRequestTimeout && resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
				w.WriteHeader(resp.StatusCode)
				return
			}
		}
		if ctx.Err() != nil {
			continue
		}
		deadline := p.failureDeadline(resource)
		if !warned {
			p.logger.Warn().Str("resource_type", kind).Dur("retry_window", p.retryWindow).Msg("live HLS request failed; retrying")
			warned = true
		}
		timer := time.NewTimer(max(0, min(p.retryInterval, time.Until(deadline))))
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
}

// Buffer a complete, bounded resource before committing any bytes to FFmpeg.
// Failed attempts are discarded so a retry cannot duplicate or omit media bytes.
func (p *liveHLSProxy) fetchResource(ctx context.Context, downstream *http.Request, uri, kind string) ([]byte, *http.Response, error) {
	var cancel context.CancelFunc
	if kind == "media" {
		ctx, cancel = context.WithCancel(ctx)
	} else {
		ctx, cancel = context.WithTimeout(ctx, p.readTimeout)
	}
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, nil, err
	}
	if value := downstream.Header.Get("Range"); value != "" && kind != "playlist" {
		req.Header.Set("Range", value)
	}
	if value := downstream.Header.Get("User-Agent"); value != "" {
		req.Header.Set("User-Agent", value)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp, nil
	}
	limit := int64(liveHLSMaxResourceSize)
	if kind == "playlist" {
		limit = hls.MaxPlaylistSize
	}
	if resp.ContentLength > limit {
		return nil, resp, errLiveHLSResourceTooLarge
	}
	reader := &liveHLSIdleReader{reader: resp.Body, timeout: p.readTimeout, cancel: cancel}
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, resp, err
	}
	if int64(len(body)) > limit {
		return nil, resp, errLiveHLSResourceTooLarge
	}
	if kind == "playlist" {
		body, err = p.rewritePlaylist(body, resp.Request.URL)
	}
	return body, resp, err
}

// liveHLSIdleReader cancels only while an upstream read makes no progress;
// it does not impose a total duration limit on an active media transfer.
type liveHLSIdleReader struct {
	reader  io.Reader
	timeout time.Duration
	cancel  context.CancelFunc
}

func (r *liveHLSIdleReader) Read(buf []byte) (int, error) {
	timer := time.AfterFunc(r.timeout, r.cancel)
	defer timer.Stop()
	return r.reader.Read(buf)
}

func copyLiveHLSHeaders(dst, src http.Header) {
	for _, name := range []string{"Content-Range", "Accept-Ranges"} {
		if value := src.Get(name); value != "" {
			dst.Set(name, value)
		}
	}
}
