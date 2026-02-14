package tasks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateNonEmptyFile(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		err := validateNonEmptyFile(filepath.Join(t.TempDir(), "missing.mp4"), "test file")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.mp4")
		if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
			t.Fatalf("failed to create empty file: %v", err)
		}

		err := validateNonEmptyFile(path, "test file")
		if err == nil {
			t.Fatal("expected error for empty file")
		}
	})

	t.Run("non-empty file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "ok.mp4")
		if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
			t.Fatalf("failed to create non-empty file: %v", err)
		}

		err := validateNonEmptyFile(path, "test file")
		if err != nil {
			t.Fatalf("expected nil error for non-empty file, got: %v", err)
		}
	})
}
