package storage

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCollectVideoFilesOnePrefix checks that the files of one video are collected and that a
// directory holding the files of two videos is refused instead of mixing them.
func TestCollectVideoFilesOnePrefix(t *testing.T) {
	t.Run("one video", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "v-info.json"), 1)
		writeFile(t, filepath.Join(dir, "v-video.mp4"), 10)

		files, err := collectVideoFiles(dir)
		require.NoError(t, err)
		assert.Equal(t, "v", files.fileName)
		assert.Equal(t, filepath.Join(dir, "v-video.mp4"), files.videoPath)
	})

	t.Run("info and video of different videos", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "a-info.json"), 1)
		writeFile(t, filepath.Join(dir, "b-video.mp4"), 10)

		_, err := collectVideoFiles(dir)
		assert.ErrorContains(t, err, "different videos")
	})

	t.Run("two video files", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "v-info.json"), 1)
		writeFile(t, filepath.Join(dir, "v-video.mp4"), 10)
		writeFile(t, filepath.Join(dir, "v-video.mkv"), 10)

		_, err := collectVideoFiles(dir)
		assert.ErrorContains(t, err, "more than one video file")
	})

	t.Run("hls directory of a different video", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "a-info.json"), 1)
		writeFile(t, filepath.Join(dir, "b-video_hls", "b-video.m3u8"), 10)

		_, err := collectVideoFiles(dir)
		assert.ErrorContains(t, err, "different videos")
	})

	t.Run("hls directory of the same video", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "v-info.json"), 1)
		writeFile(t, filepath.Join(dir, "v-video_hls", "v-video.m3u8"), 10)

		files, err := collectVideoFiles(dir)
		require.NoError(t, err)
		assert.Equal(t, "v", files.fileName)
		assert.Equal(t, filepath.Join(dir, "v-video_hls", "v-video.m3u8"), files.videoPath)
	})
}
