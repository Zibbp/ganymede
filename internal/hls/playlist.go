package hls

import (
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/bluenviron/gohlslib/v2/pkg/playlist"
	"github.com/bluenviron/gohlslib/v2/pkg/playlist/primitives"
)

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

	return os.WriteFile(path, []byte(playlistText), 0o644)
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
