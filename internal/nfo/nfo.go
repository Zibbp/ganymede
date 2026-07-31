// Package nfo creates Kodi-compatible sidecar metadata files for archived media.
package nfo

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MovieMetadata contains the archive metadata written to a movie NFO sidecar.
type MovieMetadata struct {
	Title      string
	Premiered  time.Time
	Studio     string
	Genres     []string
	Platform   string
	ExternalID string
}

type movie struct {
	XMLName   xml.Name  `xml:"movie"`
	Title     string    `xml:"title"`
	Premiered string    `xml:"premiered,omitempty"`
	Studio    string    `xml:"studio,omitempty"`
	Genres    []string  `xml:"genre,omitempty"`
	UniqueID  *uniqueID `xml:"uniqueid,omitempty"`
}

type uniqueID struct {
	Type    string `xml:"type,attr"`
	Default bool   `xml:"default,attr"`
	Value   string `xml:",chardata"`
}

// SidecarPath returns the NFO path matching an MP4 file or HLS playlist.
func SidecarPath(mediaPath string) (string, error) {
	ext := filepath.Ext(mediaPath)
	switch strings.ToLower(ext) {
	case ".mp4", ".m3u8":
		return strings.TrimSuffix(mediaPath, ext) + ".nfo", nil
	default:
		return "", fmt.Errorf("unsupported media extension %q", ext)
	}
}

// MarshalMovie serializes archive metadata as a Kodi-compatible movie NFO.
func MarshalMovie(metadata MovieMetadata) ([]byte, error) {
	nfo := movie{
		Title:     metadata.Title,
		Studio:    metadata.Studio,
		Genres:    metadata.Genres,
		Premiered: metadata.Premiered.UTC().Format("2006-01-02"),
	}
	if metadata.Premiered.IsZero() {
		nfo.Premiered = ""
	}
	if metadata.Platform != "" && metadata.ExternalID != "" {
		nfo.UniqueID = &uniqueID{
			Type:    strings.ToLower(metadata.Platform),
			Default: true,
			Value:   metadata.ExternalID,
		}
	}

	data, err := xml.MarshalIndent(nfo, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal movie NFO: %w", err)
	}

	result := make([]byte, 0, len(xml.Header)+len(data)+1)
	result = append(result, xml.Header...)
	result = append(result, data...)
	result = append(result, '\n')
	return result, nil
}

// CreateMovieIfMissing writes an NFO atomically without replacing an existing
// sidecar. It returns true only when a new file was created.
func CreateMovieIfMissing(path string, metadata MovieMetadata) (bool, error) {
	if _, err := os.Lstat(path); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("stat NFO path %s: %w", path, err)
	}

	data, err := MarshalMovie(metadata)
	if err != nil {
		return false, err
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp.*")
	if err != nil {
		return false, fmt.Errorf("create temporary NFO in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return false, fmt.Errorf("write temporary NFO %s: %w", tmpPath, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return false, fmt.Errorf("sync temporary NFO %s: %w", tmpPath, err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return false, fmt.Errorf("set temporary NFO permissions %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return false, fmt.Errorf("close temporary NFO %s: %w", tmpPath, err)
	}

	// A hard link publishes the completed temporary file atomically and fails
	// with os.ErrExist if another task or a user created the sidecar first.
	if err := os.Link(tmpPath, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return false, nil
		}
		return false, fmt.Errorf("publish NFO %s: %w", path, err)
	}

	return true, nil
}
