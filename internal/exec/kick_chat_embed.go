package exec

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/chat"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/utils"
)

const (
	kickAssetFetchWorkers  = 8
	kickAssetFetchAttempts = 3
	kickAssetMaxBytes      = 4 << 20
	kickDefaultEmoteSize   = 28
)

type kickAsset struct {
	data          string
	width, height int64
}

// EmbedKickChatAssets is the Kick counterpart of TwitchDownloader's
// "chatupdate --embed-missing": it embeds the emote and badge images used in a
// converted chat (tdlPath) as base64 so the archive plays back offline, and
// writes the result to outPath. Badge image URLs are read from the raw live
// chat file because the converted format keeps only badge name/version.
func EmbedKickChatAssets(ctx context.Context, liveChatPath, tdlPath, outPath string) error {
	raw, err := os.ReadFile(tdlPath)
	if err != nil {
		return fmt.Errorf("read converted kick chat: %w", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("decode converted kick chat: %w", err)
	}
	var comments []struct {
		Message struct {
			Fragments []struct {
				Text     string `json:"text"`
				Emoticon *struct {
					EmoticonID string `json:"emoticon_id"`
				} `json:"emoticon"`
			} `json:"fragments"`
			UserBadges []utils.UserBadge `json:"user_badges"`
		} `json:"message"`
	}
	if c, ok := doc["comments"]; ok {
		if err := json.Unmarshal(c, &comments); err != nil {
			return fmt.Errorf("decode kick chat comments: %w", err)
		}
	}

	type badgeKey struct{ name, version string }
	emoteNames := map[string]string{}
	usedBadges := map[badgeKey]struct{}{}
	for _, c := range comments {
		for _, b := range c.Message.UserBadges {
			usedBadges[badgeKey{b.ID, b.Version}] = struct{}{}
		}
		for _, f := range c.Message.Fragments {
			if f.Emoticon != nil && f.Emoticon.EmoticonID != "" {
				if _, seen := emoteNames[f.Emoticon.EmoticonID]; !seen {
					emoteNames[f.Emoticon.EmoticonID] = f.Text
				}
			}
		}
	}
	comments = nil

	type badgeSource struct{ url, title string }
	badgeURLs := map[badgeKey]badgeSource{}
	if err := forEachLiveComment(liveChatPath, func(lc utils.LiveComment) {
		for _, b := range lc.Author.Badges {
			if len(b.Icons) == 0 || b.Icons[0].URL == "" {
				continue
			}
			key := badgeKey{b.Name, fmt.Sprintf("%v", b.Version)}
			if _, used := usedBadges[key]; !used {
				continue
			}
			title := b.Title
			if title == "" {
				title = b.Name
			}
			badgeURLs[key] = badgeSource{b.Icons[0].URL, title}
		}
	}); err != nil {
		log.Warn().Err(err).Msg("could not read kick live chat for badge images; badges will not be embedded")
	}

	urls := make([]string, 0, len(emoteNames)+len(badgeURLs))
	for id := range emoteNames {
		urls = append(urls, platform.KickEmoteURL(id))
	}
	for _, b := range badgeURLs {
		urls = append(urls, b.url)
	}
	assets, err := fetchKickAssets(ctx, urls)
	if err != nil {
		return err
	}

	embedded := chat.Emotes{FirstParty: []chat.Party{}, ThirdParty: []chat.Party{}}
	for id := range emoteNames {
		a, ok := assets[platform.KickEmoteURL(id)]
		if !ok {
			continue
		}
		// Name stays empty on purpose: the chat player keys unnamed emotes by
		// ID, matching the position-based emote fragments. Named embedded
		// emotes are matched by whitespace-separated word instead, which
		// misses emotes glued to text. Fragments still carry the name.
		embedded.FirstParty = append(embedded.FirstParty, chat.Party{
			ID: id, ImageScale: 1, Data: a.data, Width: a.width, Height: a.height,
		})
	}

	badgesByName := map[string]map[string]chat.ChatTwitchBadgeVersion{}
	for k, src := range badgeURLs {
		a, ok := assets[src.url]
		if !ok {
			continue
		}
		if badgesByName[k.name] == nil {
			badgesByName[k.name] = map[string]chat.ChatTwitchBadgeVersion{}
		}
		badgesByName[k.name][k.version] = chat.ChatTwitchBadgeVersion{Title: src.title, Description: src.title, Bytes: a.data}
	}
	badges := make([]chat.ChatTwitchBadge, 0, len(badgesByName))
	for name, versions := range badgesByName {
		badges = append(badges, chat.ChatTwitchBadge{Name: name, Versions: versions})
	}

	embeddedData := struct {
		ThirdParty   []chat.Party           `json:"thirdParty"`
		FirstParty   []chat.Party           `json:"firstParty"`
		TwitchBadges []chat.ChatTwitchBadge `json:"twitchBadges"`
		TwitchBits   []chat.ChatTwitchBit   `json:"twitchBits"`
	}{embedded.ThirdParty, embedded.FirstParty, badges, []chat.ChatTwitchBit{}}
	encoded, err := json.Marshal(embeddedData)
	if err != nil {
		return err
	}
	doc["embeddedData"] = encoded

	out, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	tmp := outPath + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, outPath); err != nil {
		return err
	}
	log.Info().Int("emotes", len(embedded.FirstParty)).Int("badges", len(badges)).Str("chat_file", outPath).Msg("embedded kick chat assets")
	return nil
}

// forEachLiveComment streams a live chat JSON array without loading it whole.
func forEachLiveComment(path string, fn func(utils.LiveComment)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck
	dec := json.NewDecoder(f)
	if _, err := dec.Token(); err != nil {
		return err
	}
	for dec.More() {
		var lc utils.LiveComment
		if err := dec.Decode(&lc); err != nil {
			return err
		}
		fn(lc)
	}
	return nil
}

// fetchKickAssets downloads assets concurrently. Assets that are permanently
// gone (4xx) are skipped; transient failures that survive retries fail the
// whole call so the task retries instead of saving an incomplete archive.
func fetchKickAssets(ctx context.Context, urls []string) (map[string]kickAsset, error) {
	result := make(map[string]kickAsset, len(urls))
	var mu sync.Mutex
	var transientErr error
	jobs := make(chan string)
	var wg sync.WaitGroup
	client := &http.Client{Timeout: 20 * time.Second}
	for range kickAssetFetchWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range jobs {
				a, err := fetchKickAssetWithRetry(ctx, client, u)
				mu.Lock()
				switch {
				case err == nil:
					result[u] = a
				case errors.Is(err, errKickAssetGone):
					log.Debug().Err(err).Str("url", u).Msg("kick chat asset unavailable; skipping")
				case transientErr == nil:
					transientErr = fmt.Errorf("fetch kick chat asset %s: %w", u, err)
				}
				mu.Unlock()
			}
		}()
	}
	for _, u := range urls {
		select {
		case jobs <- u:
		case <-ctx.Done():
		}
		if ctx.Err() != nil {
			break
		}
	}
	close(jobs)
	wg.Wait()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return result, transientErr
}

var errKickAssetGone = errors.New("kick asset not available")

func fetchKickAssetWithRetry(ctx context.Context, client *http.Client, u string) (kickAsset, error) {
	backoff := time.Second
	for attempt := 1; ; attempt++ {
		a, err := fetchKickAsset(ctx, client, u)
		if err == nil || errors.Is(err, errKickAssetGone) || attempt == kickAssetFetchAttempts {
			return a, err
		}
		select {
		case <-ctx.Done():
			return kickAsset{}, ctx.Err()
		case <-time.After(backoff):
		}
		backoff *= 2
	}
}

func fetchKickAsset(ctx context.Context, client *http.Client, u string) (kickAsset, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return kickAsset{}, err
	}
	req.Header.Set("User-Agent", utils.ChromeUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return kickAsset{}, err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
		return kickAsset{}, fmt.Errorf("status %d: %w", resp.StatusCode, errKickAssetGone)
	}
	if resp.StatusCode != http.StatusOK {
		return kickAsset{}, fmt.Errorf("status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, kickAssetMaxBytes+1))
	if err != nil {
		return kickAsset{}, err
	}
	if len(data) > kickAssetMaxBytes {
		return kickAsset{}, fmt.Errorf("larger than %d bytes: %w", kickAssetMaxBytes, errKickAssetGone)
	}
	cfg, _, decodeErr := image.DecodeConfig(bytes.NewReader(data))
	// Formats Go cannot decode (SVG, AVIF) are accepted by content type.
	if decodeErr != nil && !strings.HasPrefix(http.DetectContentType(data), "image/") &&
		!strings.HasPrefix(resp.Header.Get("Content-Type"), "image/") {
		return kickAsset{}, fmt.Errorf("not an image: %w", errKickAssetGone)
	}
	a := kickAsset{data: base64.StdEncoding.EncodeToString(data), width: kickDefaultEmoteSize, height: kickDefaultEmoteSize}
	if decodeErr == nil && cfg.Width > 0 {
		a.width, a.height = int64(cfg.Width), int64(cfg.Height)
	}
	return a, nil
}
