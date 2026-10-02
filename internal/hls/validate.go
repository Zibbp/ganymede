package hls

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/bluenviron/gohlslib/v2/pkg/playlist"
)

// ValidateMediaPlaylistFiles verifies that a local media playlist references usable files.
func ValidateMediaPlaylistFiles(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read HLS playlist: %w", err)
	}
	parsed, err := playlist.Unmarshal(data)
	if err != nil {
		return fmt.Errorf("parse HLS playlist: %w", err)
	}
	media, ok := parsed.(*playlist.Media)
	if !ok || len(media.Segments) == 0 {
		return fmt.Errorf("HLS playlist has no media segments: %s", path)
	}

	check := func(rawURI string) error {
		u, err := url.Parse(rawURI)
		if err != nil || u.IsAbs() || u.Host != "" {
			return fmt.Errorf("invalid local HLS media URI %q", rawURI)
		}
		mediaPath := filepath.Join(filepath.Dir(path), filepath.FromSlash(u.Path))
		info, err := os.Stat(mediaPath)
		if err != nil {
			return fmt.Errorf("missing HLS media file %s: %w", mediaPath, err)
		}
		if !info.Mode().IsRegular() || info.Size() == 0 {
			return fmt.Errorf("empty HLS media file: %s", mediaPath)
		}
		return nil
	}
	if media.Map != nil {
		if err := check(media.Map.URI); err != nil {
			return err
		}
	}
	for _, segment := range media.Segments {
		if !segment.Gap {
			if err := check(segment.URI); err != nil {
				return err
			}
		}
	}
	return nil
}
