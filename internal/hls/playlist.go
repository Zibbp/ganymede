package hls

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/bluenviron/gohlslib/v2/pkg/playlist"
	"github.com/bluenviron/gohlslib/v2/pkg/playlist/primitives"
	"github.com/rs/zerolog/log"
)

// URI returns a percent-encoded HLS URI for a filesystem path.
func URI(path string) string {
	return (&url.URL{Path: filepath.ToSlash(path)}).String()
}

const maxPlaylistSize = 1 * 1024 * 1024

// MaxPlaylistSize is the maximum allowed size for an HLS playlist.
const MaxPlaylistSize = maxPlaylistSize

// Multivariant is a parsed HLS multivariant playlist.
type Multivariant = playlist.Multivariant

// DecodeMultivariant reads and parses an HLS multivariant playlist.
func DecodeMultivariant(r io.Reader) (*Multivariant, error) {
	byts, err := io.ReadAll(io.LimitReader(r, maxPlaylistSize+1))
	if err != nil {
		return nil, err
	}
	if len(byts) > maxPlaylistSize {
		return nil, fmt.Errorf("playlist exceeds maximum size of %d bytes", maxPlaylistSize)
	}

	byts, err = normalizeTwitchMultivariant(byts)
	if err != nil {
		return nil, err
	}

	pl, err := playlist.Unmarshal(byts)
	if err != nil {
		return nil, err
	}

	multivariant, ok := pl.(*playlist.Multivariant)
	if !ok {
		return nil, fmt.Errorf("playlist is %T, not *playlist.Multivariant", pl)
	}

	return multivariant, nil
}

// FinalizeMediaPlaylist makes an interrupted live/event media playlist look
// like a completed VOD playlist. It is safe to call repeatedly.
func FinalizeMediaPlaylist(path string) error {
	byts, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	playlistText := string(byts)
	playlistText = strings.ReplaceAll(playlistText, "#EXT-X-PLAYLIST-TYPE:EVENT", "#EXT-X-PLAYLIST-TYPE:VOD")

	trimmed := strings.TrimRight(playlistText, "\r\n")
	if !strings.Contains(trimmed, "#EXT-X-ENDLIST") {
		playlistText = trimmed + "\n#EXT-X-ENDLIST\n"
	} else if playlistText != "" && !strings.HasSuffix(playlistText, "\n") {
		playlistText += "\n"
	}

	return writeFileAtomic(path, []byte(playlistText))
}

// SegmentProbe reports a media segment's duration in seconds.
type SegmentProbe func(ctx context.Context, path string) (float64, error)

var segmentIndexPattern = regexp.MustCompile(`(\d+)(?:\.[^.]+)?$`)

// segmentIndex extracts the trailing numeric index from a segment filename so
// zero-padded (%06d) and unpadded (%d) sequences both sort numerically.
func segmentIndex(name string) int {
	m := segmentIndexPattern.FindStringSubmatch(name)
	if m == nil {
		return -1
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return -1
	}
	return n
}

// LiveCaptureID returns the immutable HLS capture prefix for a live archive.
// Capture files are named with the stream ID present when capture starts,
// but Vod.ExtID is later replaced with the Twitch VOD ID by the stream video
// ID update. Prefer ExtStreamID, then the persisted TmpVideoDownloadPath
// basename, then ExtID for legacy rows.
func LiveCaptureID(extID, extStreamID, tmpVideoDownloadPath string) string {
	if extStreamID != "" {
		return extStreamID
	}
	if base := filepath.Base(tmpVideoDownloadPath); strings.HasSuffix(base, "-video.m3u8") {
		if id := strings.TrimSuffix(base, "-video.m3u8"); id != "" {
			return id
		}
	}
	return extID
}

// HasRecoverableSegments reports whether the dir holds segments for rebuild.
// Zero-byte partials are ignored.
func HasRecoverableSegments(dir, extID string) bool {
	segments, _, _ := recoverableSegments(dir, extID)
	return len(segments) > 0
}

// recoverableSegments lists non-empty segments, fmp4 with init first, then TS.
func recoverableSegments(dir, extID string) (segments []string, fmp4 bool, err error) {
	if dir == "" || extID == "" {
		return nil, false, nil
	}
	collect := func(pattern string) ([]string, error) {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			return nil, err
		}
		var out []string
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil || info.Size() == 0 || !info.Mode().IsRegular() {
				continue
			}
			out = append(out, match)
		}
		sort.Slice(out, func(i, j int) bool {
			ii, jj := segmentIndex(filepath.Base(out[i])), segmentIndex(filepath.Base(out[j]))
			if ii != jj {
				return ii < jj
			}
			return out[i] < out[j]
		})
		return out, nil
	}

	if fmp4Init := filepath.Join(dir, extID+"_init.mp4"); fileExists(fmp4Init) {
		segments, err := collect(extID + "_segment*.m4s")
		if err != nil {
			return nil, false, err
		}
		if len(segments) > 0 {
			return segments, true, nil
		}
	}
	segments, err = collect(extID + "_segment*.ts")
	if err != nil {
		return nil, false, err
	}
	return segments, false, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// RebuildMediaPlaylistFromSegments regenerates a finalized VOD playlist from
// segments on disk. Unprobeable segments are skipped with a discontinuity.
func RebuildMediaPlaylistFromSegments(ctx context.Context, dir, extID, playlistPath string, probe SegmentProbe) error {
	segments, fmp4, err := recoverableSegments(dir, extID)
	if err != nil {
		return err
	}
	if len(segments) == 0 {
		return fmt.Errorf("no recoverable segments in %s", dir)
	}

	type entry struct {
		name     string
		duration float64
	}
	entries := make([]entry, 0, len(segments))
	maxDuration := 0.0
	for _, segment := range segments {
		if err := ctx.Err(); err != nil {
			return err
		}
		duration, err := probe(ctx, segment)
		if err != nil {
			// Skip truncated segments; the index gap emits a discontinuity.
			log.Warn().Err(err).Str("segment", filepath.Base(segment)).Msg("skipping unprobeable HLS segment during playlist rebuild")
			continue
		}
		if duration <= 0 {
			log.Warn().Str("segment", filepath.Base(segment)).Msg("skipping HLS segment with no measurable duration during playlist rebuild")
			continue
		}
		entries = append(entries, entry{name: filepath.Base(segment), duration: duration})
		maxDuration = math.Max(maxDuration, duration)
	}
	if len(entries) == 0 {
		return fmt.Errorf("no probeable segments in %s", dir)
	}
	targetDuration := int(math.Ceil(maxDuration))
	if targetDuration < 1 {
		targetDuration = 1
	}

	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	if fmp4 {
		b.WriteString("#EXT-X-VERSION:7\n")
	} else {
		b.WriteString("#EXT-X-VERSION:3\n")
	}
	fmt.Fprintf(&b, "#EXT-X-TARGETDURATION:%d\n", targetDuration)
	b.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")
	b.WriteString("#EXT-X-PLAYLIST-TYPE:VOD\n")
	b.WriteString("#EXT-X-INDEPENDENT-SEGMENTS\n")
	if fmp4 {
		fmt.Fprintf(&b, "#EXT-X-MAP:URI=%q\n", URI(extID+"_init.mp4"))
	}
	previousIndex := -2
	for _, e := range entries {
		index := segmentIndex(e.name)
		if previousIndex >= 0 && index > previousIndex+1 {
			b.WriteString("#EXT-X-DISCONTINUITY\n")
		}
		previousIndex = index
		fmt.Fprintf(&b, "#EXTINF:%.6f,\n%s\n", e.duration, URI(e.name))
	}
	b.WriteString("#EXT-X-ENDLIST\n")

	if err := writeFileAtomic(playlistPath, []byte(b.String())); err != nil {
		// Playlist may be missing after a mid-rewrite kill; create it.
		if err := writeFileAtomicCreate(playlistPath, []byte(b.String())); err != nil {
			return err
		}
	}
	return nil
}

func writeFileAtomic(path string, data []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp.*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	renamed := false
	defer func() {
		if !renamed {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return errors.Join(err, tmpFile.Close())
	}
	if err := tmpFile.Chmod(info.Mode()); err != nil {
		return errors.Join(err, tmpFile.Close())
	}
	if err := tmpFile.Sync(); err != nil {
		return errors.Join(err, tmpFile.Close())
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	renamed = true

	return syncDir(dir)
}

// writeFileAtomicCreate is writeFileAtomic for paths that may not exist yet.
func writeFileAtomicCreate(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp.*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	renamed := false
	defer func() {
		if !renamed {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return errors.Join(err, tmpFile.Close())
	}
	if err := tmpFile.Chmod(0o644); err != nil {
		return errors.Join(err, tmpFile.Close())
	}
	if err := tmpFile.Sync(); err != nil {
		return errors.Join(err, tmpFile.Close())
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	renamed = true

	return syncDir(dir)
}

func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close() //nolint:errcheck

	return dir.Sync()
}

func normalizeTwitchMultivariant(byts []byte) ([]byte, error) {
	const prefix = "#EXT-X-STREAM-INF:"

	if !strings.Contains(string(byts), prefix) {
		return byts, nil
	}

	var b strings.Builder
	lines := strings.SplitAfter(string(byts), "\n")
	for _, line := range lines {
		lineWithoutNewline := strings.TrimSuffix(line, "\n")
		newline := line[len(lineWithoutNewline):]
		lineWithoutCR := strings.TrimSuffix(lineWithoutNewline, "\r")
		cr := lineWithoutNewline[len(lineWithoutCR):]

		if strings.HasPrefix(lineWithoutCR, prefix) {
			attrs := lineWithoutCR[len(prefix):]
			normalizedAttrs, err := normalizeStreamInfAttributes(attrs)
			if err != nil {
				return nil, err
			}
			line = prefix + normalizedAttrs + cr + newline
		}

		b.WriteString(line)
	}

	return []byte(b.String()), nil
}

func normalizeStreamInfAttributes(attrsText string) (string, error) {
	var attrs primitives.Attributes
	if err := attrs.Unmarshal(attrsText); err != nil {
		return "", fmt.Errorf("invalid #EXT-X-STREAM-INF attributes: %w", err)
	}

	if _, ok := attrs["VIDEO"]; ok {
		return attrsText, nil
	}

	// https://eu.luminous.dev doesn't return a VIDEO attribute, but it can be derived from other attributes, so we add it if missing
	label := twitchVariantVideoLabel(attrs)
	if label == "" {
		return attrsText, nil
	}

	separator := ","
	if strings.TrimSpace(attrsText) == "" {
		separator = ""
	}

	return attrsText + separator + `VIDEO="` + sanitizeHLSQuotedString(label) + `"`, nil
}

// twitchVariantVideoLabel generates a video label for a Twitch HLS variant based on its attributes
// https://eu.luminous.dev doesn't return a VIDEO attribute, but it can be derived from other attributes, so we add it if missing
func twitchVariantVideoLabel(attrs primitives.Attributes) string {
	if label := attrs["STABLE-VARIANT-ID"]; label != "" {
		return label
	}
	if label := attrs["IVS-NAME"]; label != "" {
		return label
	}

	resolution := strings.TrimSpace(attrs["RESOLUTION"])
	if resolution == "" {
		if isAudioOnlyVariant(attrs) {
			return "audio_only"
		}
		return ""
	}

	height := resolution
	if parts := strings.SplitN(resolution, "x", 2); len(parts) == 2 {
		height = parts[1]
	}
	height = strings.TrimSpace(height)
	if height == "" {
		return ""
	}

	label := height + "p"
	if fps := roundedFrameRate(attrs["FRAME-RATE"]); fps != "" {
		label += fps
	}

	return label
}

func isAudioOnlyVariant(attrs primitives.Attributes) bool {
	codecs := attrs["CODECS"]
	if codecs == "" {
		return false
	}

	for _, codec := range strings.Split(codecs, ",") {
		if isVideoCodec(strings.ToLower(strings.TrimSpace(codec))) {
			return false
		}
	}

	return true
}

func isVideoCodec(codec string) bool {
	return strings.HasPrefix(codec, "avc") ||
		strings.HasPrefix(codec, "hvc") ||
		strings.HasPrefix(codec, "hev") ||
		strings.HasPrefix(codec, "av01") ||
		strings.HasPrefix(codec, "vp09") ||
		strings.HasPrefix(codec, "vp8")
}

func roundedFrameRate(frameRate string) string {
	if frameRate == "" {
		return ""
	}

	fps, err := strconv.ParseFloat(frameRate, 64)
	if err != nil {
		return ""
	}

	rounded := int(math.Round(fps))
	if rounded <= 0 {
		return ""
	}

	return strconv.Itoa(rounded)
}

func sanitizeHLSQuotedString(s string) string {
	replacer := strings.NewReplacer(`"`, "", "\r", "", "\n", "")
	return replacer.Replace(s)
}
