package exec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergePendingNDJSONIntoJSONArray(t *testing.T) {
	t.Parallel()

	t.Run("empty array", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		main := filepath.Join(dir, "chat.json")
		pending := main + liveChatPendingFileSuffix
		require.NoError(t, os.WriteFile(main, []byte("[]\n"), 0o644))
		require.NoError(t, os.WriteFile(pending, []byte("{\"id\":\"1\"}\n{\"id\":\"2\"}\n"), 0o644))

		merged, skipped, err := mergePendingNDJSONIntoJSONArray(main, pending)
		require.NoError(t, err)
		require.Equal(t, 2, merged)
		require.Equal(t, 0, skipped)

		var got []map[string]string
		data, err := os.ReadFile(main)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(data, &got))
		require.Equal(t, []map[string]string{{"id": "1"}, {"id": "2"}}, got)
	})

	t.Run("repeated merges", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		main := filepath.Join(dir, "chat.json")
		pending := main + liveChatPendingFileSuffix
		require.NoError(t, os.WriteFile(main, []byte("[]\n"), 0o644))

		for i := 0; i < 12; i++ {
			require.NoError(t, os.WriteFile(pending, []byte("{\"id\":\"x\"}\n"), 0o644))
			_, _, err := mergePendingNDJSONIntoJSONArray(main, pending)
			require.NoError(t, err)
			require.NoError(t, os.Remove(pending))
		}

		var got []map[string]string
		data, err := os.ReadFile(main)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(data, &got))
		require.Len(t, got, 12)
	})

	t.Run("repairs truncated tail", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		main := filepath.Join(dir, "chat.json")
		pending := main + liveChatPendingFileSuffix
		require.NoError(t, os.WriteFile(main, []byte("[\n{\"id\":\"old\"},"), 0o644))
		require.NoError(t, os.WriteFile(pending, []byte("{\"id\":\"new\"}\n"), 0o644))

		_, _, err := mergePendingNDJSONIntoJSONArray(main, pending)
		require.NoError(t, err)

		data, err := os.ReadFile(main)
		require.NoError(t, err)
		require.True(t, json.Valid(data), "chat file is not valid JSON: %s", data)
	})

	t.Run("skips invalid pending lines", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		main := filepath.Join(dir, "chat.json")
		pending := main + liveChatPendingFileSuffix
		require.NoError(t, os.WriteFile(main, []byte("[]\n"), 0o644))
		require.NoError(t, os.WriteFile(pending, []byte("{\"id\":\"1\"}\nnot json\n"), 0o644))

		merged, skipped, err := mergePendingNDJSONIntoJSONArray(main, pending)
		require.NoError(t, err)
		require.Equal(t, 1, merged)
		require.Equal(t, 1, skipped)
	})

	t.Run("keeps permissions and removes temp file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		main := filepath.Join(dir, "chat.json")
		pending := main + liveChatPendingFileSuffix
		require.NoError(t, os.WriteFile(main, []byte("[]\n"), 0o644))
		require.NoError(t, os.WriteFile(pending, []byte("{\"id\":\"1\"}\n"), 0o644))

		_, _, err := mergePendingNDJSONIntoJSONArray(main, pending)
		require.NoError(t, err)

		info, err := os.Stat(main)
		require.NoError(t, err)
		// Compare against a freshly written file so this also works on SMB
		// mounts, where the mount options control the reported mode.
		control := filepath.Join(dir, "control.file")
		require.NoError(t, os.WriteFile(control, []byte("control"), 0o644))
		controlInfo, err := os.Stat(control)
		require.NoError(t, err)
		require.Equal(t, controlInfo.Mode().Perm(), info.Mode().Perm())

		matches, err := filepath.Glob(main + ".tmp-*")
		require.NoError(t, err)
		require.Empty(t, matches)
	})

	t.Run("missing pending file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		main := filepath.Join(dir, "chat.json")

		merged, skipped, err := mergePendingNDJSONIntoJSONArray(main, main+liveChatPendingFileSuffix)
		require.NoError(t, err)
		require.Zero(t, merged)
		require.Zero(t, skipped)
	})
}
