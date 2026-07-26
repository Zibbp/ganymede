package utils

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMoveFileCopyFallback(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.mp4")
	dest := filepath.Join(dir, "dest.mp4")
	content := []byte("video data")
	if err := os.WriteFile(source, content, 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	renameErr := errors.New("cross-device link")
	err := moveFile(context.Background(), source, dest, func(string, string) error {
		return renameErr
	})
	if err != nil {
		t.Fatalf("moveFile returned error: %v", err)
	}

	if _, err := os.Stat(source); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("source should be removed after successful copy, stat error: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read destination file: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("destination content = %q; want %q", got, content)
	}
}

func TestMoveFilePreservesCopyError(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.mp4")
	dest := filepath.Join(dir, "dest.mp4")
	if err := os.WriteFile(source, []byte("video data"), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := moveFile(ctx, source, dest, func(string, string) error {
		return errors.New("rename failed")
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("moveFile error = %v; want context.Canceled", err)
	}
	if strings.Contains(err.Error(), "%!w(<nil>)") {
		t.Fatalf("moveFile returned a nil wrapped error: %v", err)
	}
	if _, statErr := os.Stat(source); statErr != nil {
		t.Fatalf("source should remain after a failed copy: %v", statErr)
	}
	if _, statErr := os.Stat(dest); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("partial destination should be removed, stat error: %v", statErr)
	}
}

func TestGetSizeOfDirectory(t *testing.T) {
	dir := t.TempDir()

	// Create files with known sizes
	files := []struct {
		name string
		size int
	}{
		{"file1.txt", 100},
		{"file2.txt", 200},
		{"subdir/file3.txt", 300},
	}

	for _, f := range files {
		fullPath := filepath.Join(dir, f.name)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		data := make([]byte, f.size)
		if err := os.WriteFile(fullPath, data, 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	expectedSize := int64(100 + 200 + 300)
	size, err := GetSizeOfDirectory(dir)
	if err != nil {
		t.Fatalf("GetSizeOfDirectory returned error: %v", err)
	}
	if size != expectedSize {
		t.Errorf("expected size %d, got %d", expectedSize, size)
	}
}

func TestGetSizeOfDirectory_Empty(t *testing.T) {
	dir := t.TempDir()
	size, err := GetSizeOfDirectory(dir)
	if err != nil {
		t.Fatalf("GetSizeOfDirectory returned error: %v", err)
	}
	if size != 0 {
		t.Errorf("expected size 0, got %d", size)
	}
}

func TestGetSizeOfDirectory_NonExistent(t *testing.T) {
	_, err := GetSizeOfDirectory("/nonexistent/path/shouldfail")
	if err == nil {
		t.Error("expected error for non-existent directory, got nil")
	}
}

func TestGetFreeSpaceOfDirectory(t *testing.T) {
	dir := t.TempDir()
	free, err := GetFreeSpaceOfDirectory(dir)
	if err != nil {
		t.Fatalf("GetFreeSpaceOfDirectory returned error: %v", err)
	}
	if free <= 0 {
		t.Errorf("expected free space > 0, got %d", free)
	}
}

func TestGetFreeSpaceOfDirectory_NonExistent(t *testing.T) {
	_, err := GetFreeSpaceOfDirectory("/nonexistent/path/shouldfail")
	if err == nil {
		t.Error("expected error for non-existent directory, got nil")
	}
}
