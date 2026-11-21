package exec

import (
	"context"
	"os"
	osExec "os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnsureImageMatchesExtension(t *testing.T) {
	if _, err := osExec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	dir := t.TempDir()
	webp := filepath.Join(dir, "src.webp")
	out, err := osExec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=red:s=64x36", "-frames:v", "1", webp).CombinedOutput()
	if err != nil {
		t.Skipf("ffmpeg cannot encode webp: %v %s", err, out)
	}
	data, err := os.ReadFile(webp)
	require.NoError(t, err)
	require.True(t, isWebP(data))

	for _, name := range []string{"thumb.jpg", "profile.png"} {
		target := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(target, data, 0o644))
		require.NoError(t, EnsureImageMatchesExtension(context.Background(), target))
		got, err := os.ReadFile(target)
		require.NoError(t, err)
		require.False(t, isWebP(got), name)
		if name == "thumb.jpg" {
			require.Equal(t, []byte{0xFF, 0xD8}, got[:2])
		} else {
			require.Equal(t, []byte("\x89PNG"), got[:4])
		}
	}

	// Non-WebP files are left untouched.
	jpg := filepath.Join(dir, "thumb.jpg")
	before, _ := os.ReadFile(jpg)
	require.NoError(t, EnsureImageMatchesExtension(context.Background(), jpg))
	after, _ := os.ReadFile(jpg)
	require.Equal(t, before, after)
}
