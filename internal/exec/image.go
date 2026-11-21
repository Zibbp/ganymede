package exec

import (
	"bytes"
	"context"
	"fmt"
	"os"
	osExec "os/exec"
	"path/filepath"
	"strings"
)

// isWebP reports whether data starts with a RIFF/WEBP header.
func isWebP(data []byte) bool {
	return len(data) >= 12 && bytes.Equal(data[0:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP"))
}

// EnsureImageMatchesExtension re-encodes a WebP image saved under a .jpg/.png
// name into that format. Kick serves WebP, while archives promise JPEG
// thumbnails and PNG profile images to external tools (e.g. NFO consumers).
func EnsureImageMatchesExtension(ctx context.Context, path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	header := make([]byte, 12)
	n, _ := f.Read(header)
	_ = f.Close()
	if !isWebP(header[:n]) {
		return nil
	}

	tmp := strings.TrimSuffix(path, filepath.Ext(path)) + ".convert" + ext
	args := []string{"-hide_banner", "-loglevel", "error", "-y", "-i", path, "-frames:v", "1"}
	if ext != ".png" {
		args = append(args, "-q:v", "2")
	}
	args = append(args, tmp)
	cmd := osExec.CommandContext(ctx, "ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("convert %s from webp: %w: %s", path, err, strings.TrimSpace(string(out)))
	}
	return os.Rename(tmp, path)
}
