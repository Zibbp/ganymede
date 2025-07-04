package utils

import (
	"os"
	"path/filepath"
	"testing"
)

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
