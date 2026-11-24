package platform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/hls"
	"github.com/zibbp/ganymede/internal/utils"
)

// Kick has no public API that exposes playback or chat, so this uses the same
// unofficial web endpoints the kick.com frontend and streamlink use. They sit
// behind Cloudflare: requests carry browser-like headers and are retried with
// backoff on 403/429/5xx.
const (
	kickBaseURL        = "https://kick.com"
	kickWebAPIURL      = "https://web.kick.com/api"
	kickEmoteImageURL  = "https://files.kick.com/emotes/%s/fullsize"
	kickExtIDPrefix    = "kick:"
	kickRequestTimeout = 20 * time.Second
	kickMaxAttempts    = 4
	// Thumbnails are cosmetic; never let them hold up a live check.
	kickThumbnailTimeout = 10 * time.Second

	// KickDefaultPusherKey/Cluster are used when the realtime credential
	// endpoint is unavailable. Kick rotates these; the endpoint is preferred.
	KickDefaultPusherKey     = "34bf7a0ff419a2a775b9"
	KickDefaultPusherCluster = "us2"
)

// ErrNotSupported is returned for platform operations Kick does not provide.
var ErrNotSupported = errors.New("operation not supported by this platform")

// KickChannelExtID namespaces Kick channel IDs; channel ext_id is globally
// unique and Kick and Twitch numeric IDs overlap.
func KickChannelExtID(id int64) string {
	return fmt.Sprintf("%s%d", kickExtIDPrefix, id)
}

// ParseKickChannelExtID returns the numeric Kick channel ID from an ext_id.
func ParseKickChannelExtID(extID string) (int, error) {
	return strconv.Atoi(strings.TrimPrefix(extID, kickExtIDPrefix))
}

// ChannelLogin returns the login used for platform API calls. Channels whose
// Ganymede name had to be disambiguated keep their real login in ext_login.
func ChannelLogin(name, extLogin string) string {
	if extLogin != "" {
		return extLogin
	}
	return name
}

type KickConnection struct {
	HTTPClient *http.Client
	UserAgent  string

	emoteCacheMu sync.Mutex
	emoteCache   map[string]kickEmoteCacheEntry
}

type kickEmoteCacheEntry struct {
	fetched time.Time
	sets    []KickEmoteSet
}

func NewKickConnection() *KickConnection {
	return &KickConnection{
		HTTPClient: &http.Client{Timeout: kickRequestTimeout},
		UserAgent:  utils.ChromeUserAgent,
		emoteCache: map[string]kickEmoteCacheEntry{},
	}
}

// KickChannel is the subset of /api/v2/channels/{slug} used by Ganymede.
type KickChannel struct {
	ID               int64                 `json:"id"`
	UserID           int64                 `json:"user_id"`
	Slug             string                `json:"slug"`
	PlaybackURL      string                `json:"playback_url"`
	VodEnabled       bool                  `json:"vod_enabled"`
	SubscriberBadges []KickSubscriberBadge `json:"subscriber_badges"`
	Livestream       *KickLivestream       `json:"livestream"`
	User             struct {
		ID         int64  `json:"id"`
		Username   string `json:"username"`
		Bio        string `json:"bio"`
		ProfilePic string `json:"profile_pic"`
	} `json:"user"`
	Chatroom struct {
		ID int64 `json:"id"`
	} `json:"chatroom"`
	BannerImage *struct {
		URL string `json:"url"`
	} `json:"banner_image"`
}

type KickLivestream struct {
	ID            int64          `json:"id"`
	Slug          string         `json:"slug"`
	SessionTitle  string         `json:"session_title"`
	IsLive        bool           `json:"is_live"`
	StartTime     string         `json:"start_time"`
	CreatedAt     string         `json:"created_at"`
	Language      string         `json:"language"`
	ViewerCount   int64          `json:"viewer_count"`
	Thumbnail     *KickThumbnail `json:"thumbnail"`
	Categories    []KickCategory `json:"categories"`
	IsMature      bool           `json:"is_mature"`
	DurationMilli int64          `json:"duration"`
}

type KickThumbnail struct {
	URL string `json:"url"`
}

type KickCategory struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type KickSubscriberBadge struct {
	ID         int64 `json:"id"`
	Months     int   `json:"months"`
	BadgeImage struct {
		Src string `json:"src"`
	} `json:"badge_image"`
}

type KickEmote struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	SubscribersOnly bool   `json:"subscribers_only"`
}

type KickEmoteSet struct {
	Name   string      `json:"name"`
	Slug   string      `json:"slug"`
	Emotes []KickEmote `json:"emotes"`
}

// kickEmoteSetRaw tolerates the channel set using a numeric id while the
// global sets use strings.
type kickEmoteSetRaw struct {
	Name   string          `json:"name"`
	Slug   string          `json:"slug"`
	ID     json.RawMessage `json:"id"`
	Emotes []KickEmote     `json:"emotes"`
}

// KickEmoteURL returns the image URL for a Kick emote ID.
func KickEmoteURL(id string) string {
	return fmt.Sprintf(kickEmoteImageURL, url.PathEscape(id))
}

func (k *KickConnection) client() *http.Client {
	if k.HTTPClient == nil {
		k.HTTPClient = &http.Client{Timeout: kickRequestTimeout}
	}
	return k.HTTPClient
}

func (k *KickConnection) userAgent() string {
	if k.UserAgent == "" {
		return utils.ChromeUserAgent
	}
	return k.UserAgent
}

func (k *KickConnection) newRequest(ctx context.Context, method, rawURL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", k.userAgent())
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Origin", kickBaseURL)
	req.Header.Set("Referer", kickBaseURL+"/")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// errKickNotFound marks a 404, which is not retried.
var errKickNotFound = errors.New("kick resource not found")

// errKickPlaylistNotFound means the channel reports a stream whose playlist no
// longer exists, i.e. the broadcast has just ended.
var errKickPlaylistNotFound = errors.New("kick playlist not found (stream ended)")

// doJSON performs a request and decodes JSON into out, retrying transient
// Cloudflare and server failures.
func (k *KickConnection) doJSON(ctx context.Context, method, rawURL string, payload []byte, out any) error {
	var lastErr error
	backoff := 2 * time.Second
	for attempt := 1; attempt <= kickMaxAttempts; attempt++ {
		var body io.Reader
		if payload != nil {
			body = strings.NewReader(string(payload))
		}
		req, err := k.newRequest(ctx, method, rawURL, body)
		if err != nil {
			return err
		}
		resp, err := k.client().Do(req)
		if err == nil {
			data, readErr := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
			_ = resp.Body.Close()
			switch {
			case readErr != nil:
				lastErr = fmt.Errorf("read kick response: %w", readErr)
			case resp.StatusCode == http.StatusNotFound:
				return fmt.Errorf("%s: %w", rawURL, errKickNotFound)
			case resp.StatusCode == http.StatusOK:
				if out == nil {
					return nil
				}
				if err := json.Unmarshal(data, out); err != nil {
					return fmt.Errorf("decode kick response from %s: %w", rawURL, err)
				}
				return nil
			case resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
				lastErr = fmt.Errorf("kick request %s returned %d (possible Cloudflare challenge or rate limit)", rawURL, resp.StatusCode)
			default:
				return fmt.Errorf("kick request %s returned %d", rawURL, resp.StatusCode)
			}
		} else {
			lastErr = fmt.Errorf("kick request %s: %w", rawURL, err)
		}

		if attempt == kickMaxAttempts {
			break
		}
		log.Debug().Err(lastErr).Int("attempt", attempt).Msg("retrying kick request")
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		backoff *= 2
	}
	return lastErr
}

// GetKickChannel fetches the raw Kick channel document by slug.
func (k *KickConnection) GetKickChannel(ctx context.Context, slug string) (*KickChannel, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		return nil, errors.New("kick channel slug is empty")
	}
	var ch KickChannel
	if err := k.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/api/v2/channels/%s", kickBaseURL, url.PathEscape(slug)), nil, &ch); err != nil {
		return nil, fmt.Errorf("get kick channel %s: %w", slug, err)
	}
	if ch.ID == 0 {
		return nil, fmt.Errorf("get kick channel %s: empty response", slug)
	}
	return &ch, nil
}

// GetLiveMasterPlaylist returns the decoded HLS multivariant playlist of a live
// Kick channel. Kick playback URLs are not signed, unlike Twitch.
func (k *KickConnection) GetLiveMasterPlaylist(ctx context.Context, slug string) (*hls.Multivariant, error) {
	ch, err := k.GetKickChannel(ctx, slug)
	if err != nil {
		return nil, err
	}
	if ch.Livestream == nil || !ch.Livestream.IsLive || ch.PlaybackURL == "" {
		return nil, fmt.Errorf("kick channel %s is not live: %w", slug, ErrorNoStreamsFound{})
	}
	req, err := k.newRequest(ctx, http.MethodGet, ch.PlaybackURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.apple.mpegurl, */*")
	resp, err := k.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch kick playlist: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("fetch kick playlist: %w", errKickPlaylistNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch kick playlist: status %d", resp.StatusCode)
	}
	playlist, err := hls.DecodeMultivariant(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("decode kick playlist: %w", err)
	}
	if len(playlist.Variants) == 0 {
		return nil, errors.New("kick playlist has no variants")
	}
	return playlist, nil
}

// KickRealtimeCredentials returns the Pusher app key and cluster Kick
// currently assigns to a channel's chat, falling back to known defaults.
func (k *KickConnection) KickRealtimeCredentials(ctx context.Context, channelID int64) (appKey, cluster string) {
	payload, _ := json.Marshal(map[string]any{
		"client": map[string]string{"id": uuid.NewString(), "type": "web"},
		"capabilities": map[string]any{
			"accepted_providers": []map[string]string{{"provider": "pusher"}},
		},
	})
	var resp struct {
		Data struct {
			Connections []struct {
				Provider    string `json:"provider"`
				Credentials struct {
					AppKey  string `json:"app_key"`
					Cluster string `json:"cluster"`
				} `json:"credentials"`
			} `json:"connections"`
		} `json:"data"`
	}
	endpoint := fmt.Sprintf("%s/v1/realtime/channels/%d/chat/connection", kickWebAPIURL, channelID)
	if err := k.doJSON(ctx, http.MethodPost, endpoint, payload, &resp); err != nil {
		log.Warn().Err(err).Int64("kick_channel_id", channelID).Msg("failed to fetch kick realtime credentials; using defaults")
		return KickDefaultPusherKey, KickDefaultPusherCluster
	}
	for _, c := range resp.Data.Connections {
		if c.Provider == "pusher" && c.Credentials.AppKey != "" && c.Credentials.Cluster != "" {
			return c.Credentials.AppKey, c.Credentials.Cluster
		}
	}
	return KickDefaultPusherKey, KickDefaultPusherCluster
}

// GetKickEmoteSets returns the channel, global and emoji emote sets visible in
// a channel's chat. Results are cached briefly to keep chat conversion cheap.
func (k *KickConnection) GetKickEmoteSets(ctx context.Context, slug string) ([]KickEmoteSet, error) {
	slug = strings.ToLower(slug)
	k.emoteCacheMu.Lock()
	if k.emoteCache == nil {
		k.emoteCache = map[string]kickEmoteCacheEntry{}
	}
	if entry, ok := k.emoteCache[slug]; ok && time.Since(entry.fetched) < 10*time.Minute {
		k.emoteCacheMu.Unlock()
		return entry.sets, nil
	}
	k.emoteCacheMu.Unlock()

	var raw []kickEmoteSetRaw
	if err := k.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/emotes/%s", kickBaseURL, url.PathEscape(slug)), nil, &raw); err != nil {
		return nil, fmt.Errorf("get kick emotes for %s: %w", slug, err)
	}
	sets := make([]KickEmoteSet, 0, len(raw))
	for _, s := range raw {
		name := s.Name
		if name == "" {
			name = s.Slug
		}
		sets = append(sets, KickEmoteSet{Name: name, Slug: s.Slug, Emotes: s.Emotes})
	}
	k.emoteCacheMu.Lock()
	k.emoteCache[slug] = kickEmoteCacheEntry{fetched: time.Now(), sets: sets}
	k.emoteCacheMu.Unlock()
	return sets, nil
}

// parseKickTime parses Kick's "2006-01-02 15:04:05" UTC timestamps.
func parseKickTime(value string) time.Time {
	for _, layout := range []string{"2006-01-02 15:04:05", time.RFC3339} {
		if t, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return t
		}
	}
	return time.Time{}
}

func kickLiveStreamInfo(ch *KickChannel) LiveStreamInfo {
	ls := ch.Livestream
	info := LiveStreamInfo{
		ID:          strconv.FormatInt(ls.ID, 10),
		UserID:      KickChannelExtID(ch.ID),
		UserLogin:   ch.Slug,
		UserName:    ch.User.Username,
		Type:        "live",
		Title:       ls.SessionTitle,
		ViewerCount: ls.ViewerCount,
		StartedAt:   parseKickTime(ls.StartTime),
		Language:    ls.Language,
	}
	if info.StartedAt.IsZero() {
		info.StartedAt = parseKickTime(ls.CreatedAt)
	}
	if info.StartedAt.IsZero() {
		info.StartedAt = time.Now().UTC()
	}
	if len(ls.Categories) > 0 {
		info.GameID = strconv.FormatInt(ls.Categories[0].ID, 10)
		info.GameName = ls.Categories[0].Name
	}
	if ls.Thumbnail != nil {
		info.ThumbnailURL = ls.Thumbnail.URL
	}
	return info
}

func (k *KickConnection) Authenticate(ctx context.Context) (*ConnectionInfo, error) {
	return &ConnectionInfo{}, nil
}

func (k *KickConnection) GetVideo(ctx context.Context, id string, withChapters bool, withMutedSegments bool) (*VideoInfo, error) {
	return nil, fmt.Errorf("kick video archiving: %w", ErrNotSupported)
}

func (k *KickConnection) GetLiveStream(ctx context.Context, channelName string) (*LiveStreamInfo, error) {
	ch, err := k.GetKickChannel(ctx, channelName)
	if err != nil {
		return nil, err
	}
	if ch.Livestream == nil || !ch.Livestream.IsLive {
		return nil, fmt.Errorf("failed to fetch stream for channel %s: %w", channelName, ErrorNoStreamsFound{})
	}
	info := kickLiveStreamInfo(ch)
	// The channel document's stream.kick.com thumbnail is not publicly
	// readable; the livestream endpoint links the images.kick.com renditions.
	if thumb := k.livestreamThumbnail(ctx, ch.Slug); thumb != "" {
		info.ThumbnailURL = thumb
	}
	return &info, nil
}

// livestreamThumbnail returns the largest live thumbnail rendition, or "" if
// unavailable. Thumbnails are cosmetic, so failures are only logged.
func (k *KickConnection) livestreamThumbnail(ctx context.Context, slug string) string {
	ctx, cancel := context.WithTimeout(ctx, kickThumbnailTimeout)
	defer cancel()
	var resp struct {
		Data struct {
			Thumbnail struct {
				Src    string `json:"src"`
				Srcset string `json:"srcset"`
			} `json:"thumbnail"`
		} `json:"data"`
	}
	if err := k.doJSON(ctx, http.MethodGet, fmt.Sprintf("%s/api/v2/channels/%s/livestream", kickBaseURL, url.PathEscape(slug)), nil, &resp); err != nil {
		log.Debug().Err(err).Str("channel", slug).Msg("failed to fetch kick livestream thumbnail")
		return ""
	}
	return largestSrcsetURL(resp.Data.Thumbnail.Srcset, resp.Data.Thumbnail.Src)
}

// largestSrcsetURL picks the widest "url 1920w" candidate, else fallback.
func largestSrcsetURL(srcset, fallback string) string {
	best, bestWidth := fallback, 0
	for _, candidate := range strings.Split(srcset, ",") {
		fields := strings.Fields(strings.TrimSpace(candidate))
		if len(fields) != 2 || !strings.HasSuffix(fields[1], "w") {
			continue
		}
		width, err := strconv.Atoi(strings.TrimSuffix(fields[1], "w"))
		if err == nil && width > bestWidth {
			best, bestWidth = fields[0], width
		}
	}
	return best
}

// GetLiveStreams is not supported because Kick lookups are by slug; use
// GetLiveStream per channel.
func (k *KickConnection) GetLiveStreams(ctx context.Context, channelIDs []string) ([]LiveStreamInfo, error) {
	return nil, fmt.Errorf("kick bulk live lookup: %w", ErrNotSupported)
}

func (k *KickConnection) GetChannel(ctx context.Context, channelName *string, channelID *string) (*ChannelInfo, error) {
	if channelName == nil || *channelName == "" {
		return nil, fmt.Errorf("kick channel lookup by id: %w", ErrNotSupported)
	}
	ch, err := k.GetKickChannel(ctx, *channelName)
	if err != nil {
		return nil, err
	}
	displayName := ch.User.Username
	if displayName == "" {
		displayName = ch.Slug
	}
	info := &ChannelInfo{
		ID:              KickChannelExtID(ch.ID),
		Login:           ch.Slug,
		DisplayName:     displayName,
		Description:     ch.User.Bio,
		ProfileImageURL: ch.User.ProfilePic,
	}
	if ch.BannerImage != nil {
		info.OfflineImageURL = ch.BannerImage.URL
	}
	return info, nil
}

// GetVideos returns no videos: Kick VOD archiving is not implemented, and an
// empty list lets live-archive bookkeeping fall back to capture timestamps.
func (k *KickConnection) GetVideos(ctx context.Context, channelId string, videoType VideoType, withChapters bool, withMutedSegments bool) ([]VideoInfo, error) {
	return []VideoInfo{}, nil
}

func (k *KickConnection) GetCategories(ctx context.Context) ([]Category, error) {
	return []Category{}, nil
}

func (k *KickConnection) GetGlobalBadges(ctx context.Context) ([]Badge, error) {
	return []Badge{}, nil
}

// GetChannelBadges returns subscriber badges. channelId is the Kick slug.
func (k *KickConnection) GetChannelBadges(ctx context.Context, channelId string) ([]Badge, error) {
	ch, err := k.GetKickChannel(ctx, channelId)
	if err != nil {
		return nil, err
	}
	badges := make([]Badge, 0, len(ch.SubscriberBadges))
	for _, b := range ch.SubscriberBadges {
		badges = append(badges, Badge{
			Version:    strconv.Itoa(b.Months),
			Name:       "subscriber",
			Title:      fmt.Sprintf("%d-Month Subscriber", b.Months),
			ImageUrl1X: b.BadgeImage.Src,
			ImageUrl2X: b.BadgeImage.Src,
			ImageUrl4X: b.BadgeImage.Src,
		})
	}
	return badges, nil
}

func (k *KickConnection) GetGlobalEmotes(ctx context.Context) ([]Emote, error) {
	return []Emote{}, nil
}

// GetChannelEmotes returns every emote usable in a channel. channelId is the
// Kick slug.
func (k *KickConnection) GetChannelEmotes(ctx context.Context, channelId string) ([]Emote, error) {
	sets, err := k.GetKickEmoteSets(ctx, channelId)
	if err != nil {
		return nil, err
	}
	var emotes []Emote
	for _, set := range sets {
		for _, e := range set.Emotes {
			id := strconv.FormatInt(e.ID, 10)
			emotes = append(emotes, Emote{
				ID:     id,
				Name:   e.Name,
				URL:    KickEmoteURL(id),
				Format: EmoteFormatStatic,
				Type:   EmoteTypeSubscription,
				Source: "kick",
			})
		}
	}
	return emotes, nil
}

func (k *KickConnection) GetChannelClips(ctx context.Context, channelId string, filter ClipsFilter) ([]ClipInfo, error) {
	return []ClipInfo{}, nil
}

func (k *KickConnection) GetClip(ctx context.Context, id string) (*ClipInfo, error) {
	return nil, fmt.Errorf("kick clip archiving: %w", ErrNotSupported)
}

// CheckIfStreamIsLive fetches the stream's playlist rather than trusting the
// channel document, which keeps reporting a stream as live for a while after
// it ends (the playlist then returns 404).
func (k *KickConnection) CheckIfStreamIsLive(ctx context.Context, channelName string) (bool, error) {
	_, err := k.GetLiveMasterPlaylist(ctx, channelName)
	if err == nil {
		return true, nil
	}
	var noStreams ErrorNoStreamsFound
	if errors.As(err, &noStreams) || errors.Is(err, errKickPlaylistNotFound) {
		return false, nil
	}
	return false, err
}

func (k *KickConnection) GetStreams(ctx context.Context, limit int) ([]LiveStreamInfo, error) {
	return []LiveStreamInfo{}, nil
}

var _ Platform = (*KickConnection)(nil)
