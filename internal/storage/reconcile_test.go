package storage

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, make([]byte, size), 0644))
}

// writeVideo writes the files the archiver would write for a video and returns the path of its
// video file.
func writeVideo(t *testing.T, dir string, name string) string {
	t.Helper()
	video := filepath.Join(dir, name+"-video.mp4")
	writeFile(t, video, 10)
	writeFile(t, filepath.Join(dir, name+"-info.json"), 1)
	writeFile(t, filepath.Join(dir, name+"-thumbnail.jpg"), 1)
	return video
}

// age moves the modification time of everything below the root out of the grace period, so
// that the tree looks like an existing library rather than one written a second ago.
func age(t *testing.T, root string) {
	t.Helper()
	old := time.Now().Add(-24 * time.Hour)
	require.NoError(t, filepath.WalkDir(root, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Chtimes(path, old, old)
	}))
}

func scan(t *testing.T, videosDir string, videos []VideoReference, channels []ChannelReference, otherDirectories []string) []string {
	t.Helper()
	referenced := resolve(References{
		VideosDir:        videosDir,
		Videos:           videos,
		Channels:         channels,
		OtherDirectories: otherDirectories,
	})

	orphans, err := findOrphanedDirectories(videosDir, referenced)
	require.NoError(t, err)

	relativePaths := make([]string, 0, len(orphans))
	for _, path := range orphans {
		rel, err := filepath.Rel(videosDir, path)
		require.NoError(t, err)
		relativePaths = append(relativePaths, rel)
	}
	return relativePaths
}

// TestFindOrphanedDirectories checks that a directory is reported when it holds video files,
// sits next to a video that the database and the disk agree on, and belongs to no video.
func TestFindOrphanedDirectories(t *testing.T) {
	videosDir := t.TempDir()

	archivedVideo := writeVideo(t, filepath.Join(videosDir, "channel_a", "video_1"), "video_1")
	writeFile(t, filepath.Join(videosDir, "channel_a", "profile.png"), 5)

	// Left behind when a video was deleted without deleting its files.
	writeVideo(t, filepath.Join(videosDir, "channel_a", "video_2"), "video_2")

	// A live archive keeps its segments in a subdirectory.
	writeFile(t, filepath.Join(videosDir, "channel_a", "video_3", "video_3-video_hls", "video_3-video.m3u8"), 20)

	// Nested storage template.
	nestedVideo := writeVideo(t, filepath.Join(videosDir, "channel_b", "2026", "video_4"), "video_4")
	writeVideo(t, filepath.Join(videosDir, "channel_b", "2026", "video_5"), "video_5")

	age(t, videosDir)

	orphans := scan(t, videosDir, []VideoReference{
		{Paths: []string{archivedVideo}, ChannelFolders: []string{"channel_a"}},
		{Paths: []string{nestedVideo}, ChannelFolders: []string{"channel_b"}},
	}, nil, nil)

	assert.ElementsMatch(t, []string{
		filepath.Join("channel_a", "video_2"),
		filepath.Join("channel_a", "video_3"),
		filepath.Join("channel_b", "2026", "video_5"),
	}, orphans)
}

// TestFindOrphanedDirectoriesWithoutAnchorsReportsNothing checks the case that matters most: an
// empty or restored database must not turn the library into a list of orphans.
func TestFindOrphanedDirectoriesWithoutAnchorsReportsNothing(t *testing.T) {
	videosDir := t.TempDir()

	writeVideo(t, filepath.Join(videosDir, "channel_a", "video_1"), "video_1")
	writeVideo(t, filepath.Join(videosDir, "channel_a", "video_2"), "video_2")
	age(t, videosDir)

	t.Run("empty database", func(t *testing.T) {
		assert.Empty(t, scan(t, videosDir, nil, nil, nil))
	})

	t.Run("every stored path points outside of the videos directory", func(t *testing.T) {
		// The library is somewhere else entirely, so nothing here can be judged.
		assert.Empty(t, scan(t, videosDir, []VideoReference{
			{Paths: []string{"/old/videos/channel_a/video_1/video_1-video.mp4"}, ChannelFolders: []string{"channel_a"}},
			{Paths: []string{"/old/videos/channel_a/video_2/video_2-video.mp4"}, ChannelFolders: []string{"channel_a"}},
		}, nil, nil))
	})

	t.Run("stored paths point at directories that are gone", func(t *testing.T) {
		// The video the database knows about is not on disk, so nothing vouches for the
		// directory it would be in, and its neighbours must not be reported.
		assert.Empty(t, scan(t, videosDir, []VideoReference{
			{
				Paths:          []string{filepath.Join(videosDir, "channel_a", "video_9", "video_9-video.mp4")},
				ChannelFolders: []string{"channel_a"},
			},
		}, nil, nil))
	})

	t.Run("a video that belongs to no channel stops the scan", func(t *testing.T) {
		referenced := resolve(References{
			VideosDir: videosDir,
			Videos:    []VideoReference{{Paths: []string{filepath.Join(videosDir, "gone", "gone-video.mp4")}}},
		})

		_, err := findOrphanedDirectories(videosDir, referenced)
		assert.ErrorContains(t, err, "refusing to scan")
	})
}

// TestFindOrphanedDirectoriesNeverLooksInsideAVideo checks that the directory of a video is
// never a region, not even when the video anchors it twice through its sprites and its
// segments.
func TestFindOrphanedDirectoriesNeverLooksInsideAVideo(t *testing.T) {
	videosDir := t.TempDir()

	videoDir := filepath.Join(videosDir, "channel_a", "video_1")
	playlist := filepath.Join(videoDir, "video_1-video_hls", "video_1-video.m3u8")
	writeFile(t, playlist, 10)
	sprite := filepath.Join(videoDir, "sprites", "sprite_0.jpg")
	writeFile(t, sprite, 10)

	// A filesystem cache names its entries after the media files, and an operator can keep
	// anything in there. Neither is a leftover.
	writeFile(t, filepath.Join(videoDir, "@eaDir", "video_1-video.mp4", "SYNOPHOTO_THUMB.jpg"), 20)
	writeVideo(t, filepath.Join(videoDir, "extras"), "bonus")

	writeVideo(t, filepath.Join(videosDir, "channel_a", "video_2"), "video_2")
	age(t, videosDir)

	orphans := scan(t, videosDir, []VideoReference{
		{Paths: []string{playlist, filepath.Dir(playlist), sprite}},
	}, nil, nil)

	assert.Equal(t, []string{filepath.Join("channel_a", "video_2")}, orphans)
}

// TestFindOrphanedDirectoriesProtectsVideoWithoutResolvingPaths checks that a video whose
// stored paths no longer resolve still shields the directory it would live in today. The folder
// name of a nested storage template contains separators, so it has to be treated as a path.
func TestFindOrphanedDirectoriesProtectsVideoWithoutResolvingPaths(t *testing.T) {
	videosDir := t.TempDir()

	archivedVideo := writeVideo(t, filepath.Join(videosDir, "channel_a", "2026", "video_1"), "video_1")
	writeVideo(t, filepath.Join(videosDir, "channel_a", "2026", "video_2"), "video_2")
	writeVideo(t, filepath.Join(videosDir, "channel_a", "2026", "video_3"), "video_3")
	age(t, videosDir)

	orphans := scan(t, videosDir, []VideoReference{
		{Paths: []string{archivedVideo}},
		{
			// Migrated on disk but not in the database.
			Paths:          []string{"/old/videos/channel_a/2026/video_2/video_2-video.mp4"},
			FolderName:     filepath.Join("2026", "video_2"),
			ChannelFolders: []string{"channel_a"},
		},
	}, nil, nil)

	assert.Equal(t, []string{filepath.Join("channel_a", "2026", "video_3")}, orphans)
}

// TestFindOrphanedDirectoriesProtectsKnownFiles checks that a channel the database and the disk
// disagree about is left alone entirely. A storage migration that did not finish moves the
// files of a video and leaves its row pointing at the old directory, and then nothing in that
// channel can be called a leftover.
func TestFindOrphanedDirectoriesProtectsKnownFiles(t *testing.T) {
	videosDir := t.TempDir()

	archivedVideo := writeVideo(t, filepath.Join(videosDir, "channel_a", "video_1"), "video_1")
	// The files of video_2 were moved here, its row still points at the old directory.
	writeVideo(t, filepath.Join(videosDir, "channel_a", "2026-01-02-video_2"), "video_2")
	writeVideo(t, filepath.Join(videosDir, "channel_a", "video_3"), "video_3")
	// A channel the database and the disk agree on reports its leftover as usual.
	otherVideo := writeVideo(t, filepath.Join(videosDir, "channel_b", "video_4"), "video_4")
	writeVideo(t, filepath.Join(videosDir, "channel_b", "video_5"), "video_5")
	age(t, videosDir)

	orphans := scan(t, videosDir, []VideoReference{
		{Paths: []string{archivedVideo}, FileName: "video_1", ChannelFolders: []string{"channel_a"}},
		{
			Paths:          []string{filepath.Join(videosDir, "channel_a", "video_2", "video_2-video.mp4")},
			FileName:       "video_2",
			ChannelFolders: []string{"channel_a"},
		},
		{Paths: []string{otherVideo}, FileName: "video_4", ChannelFolders: []string{"channel_b"}},
	}, nil, nil)

	assert.Equal(t, []string{filepath.Join("channel_b", "video_5")}, orphans)
}

// TestFindOrphanedDirectoriesRecognisesAVideoFileAlone checks the marker that matters most: a
// directory holding nothing but the video file.
func TestFindOrphanedDirectoriesRecognisesAVideoFileAlone(t *testing.T) {
	videosDir := t.TempDir()

	archivedVideo := writeVideo(t, filepath.Join(videosDir, "channel_a", "video_1"), "video_1")
	writeFile(t, filepath.Join(videosDir, "channel_a", "mp4_only", "video_2-video.mp4"), 10)
	writeFile(t, filepath.Join(videosDir, "channel_a", "ts_only", "video_3-video.ts"), 10)
	age(t, videosDir)

	orphans := scan(t, videosDir, []VideoReference{
		{Paths: []string{archivedVideo}, ChannelFolders: []string{"channel_a"}},
	}, nil, nil)

	assert.ElementsMatch(t, []string{
		filepath.Join("channel_a", "mp4_only"),
		filepath.Join("channel_a", "ts_only"),
	}, orphans)
}

// TestFindOrphanedDirectoriesProtectsImportedHlsVideo checks the video whose only stored path
// points into its own subdirectory, which is what the API creates. Nothing but that path tells
// the scan where the video lives.
func TestFindOrphanedDirectoriesProtectsImportedHlsVideo(t *testing.T) {
	videosDir := t.TempDir()

	playlist := filepath.Join(videosDir, "channel_a", "video_1", "video_1-video_hls", "video_1-video.m3u8")
	writeFile(t, playlist, 10)
	writeFile(t, filepath.Join(videosDir, "channel_a", "video_1", "extras", "bonus-video.mp4"), 10)
	writeVideo(t, filepath.Join(videosDir, "channel_a", "video_2"), "video_2")
	age(t, videosDir)

	// No video_hls_path, no folder name, exactly what POST /vod stores.
	orphans := scan(t, videosDir, []VideoReference{
		{Paths: []string{playlist}, ChannelFolders: []string{"channel_a"}},
	}, nil, nil)

	assert.Equal(t, []string{filepath.Join("channel_a", "video_2")}, orphans)
}

// TestFindOrphanedDirectoriesIgnoresFiles checks that a file is never reported, however it is
// named.
func TestFindOrphanedDirectoriesIgnoresFiles(t *testing.T) {
	videosDir := t.TempDir()

	archivedVideo := writeVideo(t, filepath.Join(videosDir, "channel_a", "video_1"), "video_1")
	writeFile(t, filepath.Join(videosDir, "channel_a", "stray-video.mp4"), 10)
	writeFile(t, filepath.Join(videosDir, "channel_a", "stray-info.json"), 10)
	age(t, videosDir)

	orphans := scan(t, videosDir, []VideoReference{
		{Paths: []string{archivedVideo}, ChannelFolders: []string{"channel_a"}},
	}, nil, nil)

	assert.Empty(t, orphans)
}

// TestFindOrphanedDirectoriesSkipsChannelsWithMissingVideos checks the same rule for a video
// whose files are simply gone: the database still knows it, so its channel is not scanned.
func TestFindOrphanedDirectoriesSkipsChannelsWithMissingVideos(t *testing.T) {
	videosDir := t.TempDir()

	archivedVideo := writeVideo(t, filepath.Join(videosDir, "channel_a", "video_1"), "video_1")
	writeVideo(t, filepath.Join(videosDir, "channel_a", "video_2"), "video_2")
	age(t, videosDir)

	orphans := scan(t, videosDir, []VideoReference{
		{Paths: []string{archivedVideo}, ChannelFolders: []string{"channel_a"}},
		{
			// Deleted from disk by hand, the row is still there.
			Paths:          []string{filepath.Join(videosDir, "channel_a", "video_9", "video_9-video.mp4")},
			ChannelFolders: []string{"channel_a"},
		},
	}, nil, nil)

	assert.Empty(t, orphans)
}

// TestFindOrphanedDirectoriesFindsChannelWithoutVideos checks the other common case: every
// video of a channel was deleted without its files, so nothing inside the channel directory
// anchors the scan. Another video that the database and the disk agree on is what makes it
// safe to look into the channel directory at all.
func TestFindOrphanedDirectoriesFindsChannelWithoutVideos(t *testing.T) {
	videosDir := t.TempDir()

	writeVideo(t, filepath.Join(videosDir, "channel_a", "video_1"), "video_1")
	writeVideo(t, filepath.Join(videosDir, "channel_a", "video_2"), "video_2")
	writeFile(t, filepath.Join(videosDir, "channel_a", "profile.png"), 5)
	archivedVideo := writeVideo(t, filepath.Join(videosDir, "channel_b", "video_3"), "video_3")
	// A channel that is not in the database either is not touched.
	writeVideo(t, filepath.Join(videosDir, "channel_c", "video_4"), "video_4")
	age(t, videosDir)

	channels := []ChannelReference{
		{Folders: []string{"channel_a"}},
		{Folders: []string{"channel_b"}},
	}

	t.Run("with a video the database and the disk agree on", func(t *testing.T) {
		orphans := scan(t, videosDir, []VideoReference{{Paths: []string{archivedVideo}}}, channels, nil)

		assert.ElementsMatch(t, []string{
			filepath.Join("channel_a", "video_1"),
			filepath.Join("channel_a", "video_2"),
		}, orphans)
	})

	t.Run("without any video in the database", func(t *testing.T) {
		// A database that was restored empty next to a full library must not report it.
		assert.Empty(t, scan(t, videosDir, nil, channels, nil))
	})
}

// TestFindOrphanedDirectoriesNeverReportsTopLevelDirectories checks that the entries of the
// videos directory itself, such as a NAS recycle bin or another library, are never reported,
// not even when a video sits directly in one of them.
func TestFindOrphanedDirectoriesNeverReportsTopLevelDirectories(t *testing.T) {
	videosDir := t.TempDir()

	// A video directly below the videos directory, so its region would be the videos directory.
	topLevelVideo := writeVideo(t, filepath.Join(videosDir, "video_1"), "video_1")
	writeVideo(t, filepath.Join(videosDir, "#recycle"), "deleted_video")
	writeVideo(t, filepath.Join(videosDir, "lost+found"), "recovered")
	writeVideo(t, filepath.Join(videosDir, "channel_b"), "hand_placed")
	age(t, videosDir)

	orphans := scan(t, videosDir, []VideoReference{{Paths: []string{topLevelVideo}}}, nil, nil)

	assert.Empty(t, orphans)
}

// TestFindOrphanedDirectoriesNeverReportsARegion checks that a directory that holds videos of
// its own is not reported because it happens to hold video files as well.
func TestFindOrphanedDirectoriesNeverReportsARegion(t *testing.T) {
	videosDir := t.TempDir()

	archivedVideo := writeVideo(t, filepath.Join(videosDir, "channel_a", "video_1"), "video_1")

	// An older layout where a video was stored directly in the year directory, which now also
	// holds a video of its own.
	yearDir := filepath.Join(videosDir, "channel_a", "2026")
	writeVideo(t, yearDir, "loose_video")
	nestedVideo := writeVideo(t, filepath.Join(yearDir, "video_2"), "video_2")
	age(t, videosDir)

	orphans := scan(t, videosDir, []VideoReference{
		{Paths: []string{archivedVideo}},
		{Paths: []string{nestedVideo}},
	}, nil, nil)

	assert.Empty(t, orphans)
}

// TestFindOrphanedDirectoriesIgnoresForeignDirectories checks that directories that do not hold
// the files of a video are not reported.
func TestFindOrphanedDirectoriesIgnoresForeignDirectories(t *testing.T) {
	videosDir := t.TempDir()

	archivedVideo := writeVideo(t, filepath.Join(videosDir, "channel_a", "video_1"), "video_1")
	writeFile(t, filepath.Join(videosDir, "channel_a", "notes", "readme.txt"), 10)
	writeFile(t, filepath.Join(videosDir, "channel_a", "logs", "irc-chat.log"), 10)
	writeFile(t, filepath.Join(videosDir, "channel_a", "@eaDir", "video_1-thumbnail.jpg", "SYNOPHOTO_THUMB.jpg"), 10)
	age(t, videosDir)

	orphans := scan(t, videosDir, []VideoReference{{Paths: []string{archivedVideo}}}, nil, nil)

	assert.Empty(t, orphans)
}

// TestFindOrphanedDirectoriesSkipsOtherDataDirectories checks that another configured data
// directory is left alone even when it sits next to the videos.
func TestFindOrphanedDirectoriesSkipsOtherDataDirectories(t *testing.T) {
	videosDir := t.TempDir()

	archivedVideo := writeVideo(t, filepath.Join(videosDir, "channel_a", "video_1"), "video_1")
	tempDir := filepath.Join(videosDir, "channel_a", "temp")
	writeVideo(t, tempDir, "live")
	age(t, videosDir)

	orphans := scan(t, videosDir, []VideoReference{{Paths: []string{archivedVideo}}}, nil, []string{tempDir})

	assert.Empty(t, orphans)
}

// TestFindOrphanedDirectoriesSkipsRecentDirectories checks that a directory that is being
// written to is not reported, however deep the write happens.
func TestFindOrphanedDirectoriesSkipsRecentDirectories(t *testing.T) {
	videosDir := t.TempDir()

	archivedVideo := writeVideo(t, filepath.Join(videosDir, "channel_a", "video_1"), "video_1")
	writeVideo(t, filepath.Join(videosDir, "channel_a", "video_2"), "video_2")
	age(t, videosDir)

	// Written now, several levels down, so the directory is within the grace period.
	writeFile(t, filepath.Join(videosDir, "channel_a", "video_2", "video_2-video_hls", "0.ts"), 10)

	orphans := scan(t, videosDir, []VideoReference{{Paths: []string{archivedVideo}}}, nil, nil)

	assert.Empty(t, orphans)
}

// TestFindOrphanedDirectoriesReportsFutureTimestamps checks that a broken modification time
// cannot hide a directory forever.
func TestFindOrphanedDirectoriesReportsFutureTimestamps(t *testing.T) {
	videosDir := t.TempDir()

	archivedVideo := writeVideo(t, filepath.Join(videosDir, "channel_a", "video_1"), "video_1")
	orphan := filepath.Join(videosDir, "channel_a", "video_2")
	writeVideo(t, orphan, "video_2")
	age(t, videosDir)

	future := time.Now().Add(72 * time.Hour)
	require.NoError(t, os.Chtimes(orphan, future, future))

	orphans := scan(t, videosDir, []VideoReference{{Paths: []string{archivedVideo}}}, nil, nil)

	assert.Equal(t, []string{filepath.Join("channel_a", "video_2")}, orphans)
}

// TestFindOrphanedDirectoriesSkipsSymlinkedChannel checks that a channel directory that is a
// symlink out of the videos directory is not scanned: everything below it would be reported
// with a path that looks inside but is not.
func TestFindOrphanedDirectoriesSkipsSymlinkedChannel(t *testing.T) {
	videosDir := t.TempDir()
	elsewhere := t.TempDir()

	archivedVideo := writeVideo(t, filepath.Join(videosDir, "channel_a", "video_1"), "video_1")
	writeVideo(t, filepath.Join(elsewhere, "video_2"), "video_2")
	require.NoError(t, os.Symlink(elsewhere, filepath.Join(videosDir, "channel_b")))
	age(t, videosDir)
	age(t, elsewhere)

	orphans := scan(t, videosDir, []VideoReference{
		{Paths: []string{archivedVideo}, ChannelFolders: []string{"channel_a"}},
	}, []ChannelReference{
		{Folders: []string{"channel_a"}},
		{Folders: []string{"channel_b"}},
	}, nil)

	assert.Empty(t, orphans)
}

// TestFindOrphanedDirectoriesIgnoresSymlinks checks that a symlink is not reported. Note that
// the grace period would hide it as well, because the modification time of a symlink cannot be
// changed from the standard library.
func TestFindOrphanedDirectoriesIgnoresSymlinks(t *testing.T) {
	videosDir := t.TempDir()

	archivedVideo := writeVideo(t, filepath.Join(videosDir, "channel_a", "video_1"), "video_1")

	target := t.TempDir()
	writeVideo(t, target, "elsewhere")
	require.NoError(t, os.Symlink(target, filepath.Join(videosDir, "channel_a", "linked_video")))
	age(t, videosDir)

	orphans := scan(t, videosDir, []VideoReference{{Paths: []string{archivedVideo}}}, nil, nil)

	assert.Empty(t, orphans)
}

// TestFindOrphanedDirectoriesNeverReportsAnAncestorOfAVideo checks that a directory is not
// reported when a video lives below it, because deleting it would take that video with it.
func TestFindOrphanedDirectoriesNeverReportsAnAncestorOfAVideo(t *testing.T) {
	videosDir := t.TempDir()

	archivedVideo := writeVideo(t, filepath.Join(videosDir, "channel_a", "video_1"), "video_1")

	// The template changed from one level to two. The old year directory still holds the files
	// of a deleted video, and a video that is still in the database now lives below it.
	yearDir := filepath.Join(videosDir, "channel_a", "2026")
	writeVideo(t, yearDir, "left_over")
	nestedVideo := writeVideo(t, filepath.Join(yearDir, "03", "video_2"), "video_2")
	age(t, videosDir)

	orphans := scan(t, videosDir, []VideoReference{
		{Paths: []string{archivedVideo}},
		{Paths: []string{nestedVideo}},
	}, nil, nil)

	assert.Empty(t, orphans)
}

// TestFindOrphanedDirectoriesRecognisesEveryKindOfLeftover checks the files an archive can
// leave behind when it never got as far as writing the video itself.
func TestFindOrphanedDirectoriesRecognisesEveryKindOfLeftover(t *testing.T) {
	videosDir := t.TempDir()

	archivedVideo := writeVideo(t, filepath.Join(videosDir, "channel_a", "video_1"), "video_1")

	writeFile(t, filepath.Join(videosDir, "channel_a", "web_thumbnail_only", "v-web_thumbnail.jpg"), 10)
	writeFile(t, filepath.Join(videosDir, "channel_a", "caption_only", "v-caption.vtt"), 10)
	writeFile(t, filepath.Join(videosDir, "channel_a", "sprites_only", "sprites", "sprite-000.jpg"), 10)
	writeFile(t, filepath.Join(videosDir, "channel_a", "converted_chat_only", "v-chat-convert.json"), 10)
	age(t, videosDir)

	orphans := scan(t, videosDir, []VideoReference{{Paths: []string{archivedVideo}}}, nil, nil)

	assert.ElementsMatch(t, []string{
		filepath.Join("channel_a", "web_thumbnail_only"),
		filepath.Join("channel_a", "caption_only"),
		filepath.Join("channel_a", "sprites_only"),
		filepath.Join("channel_a", "converted_chat_only"),
	}, orphans)
}

// TestIsOutside checks that a directory whose name starts with dots is not mistaken for a path
// that leaves the videos directory.
func TestIsOutside(t *testing.T) {
	assert.True(t, isOutside(".."))
	assert.True(t, isOutside(filepath.Join("..", "elsewhere")))
	assert.False(t, isOutside("..channel"))
	assert.False(t, isOutside(filepath.Join("..channel", "video")))
	assert.False(t, isOutside("channel"))
}

// TestFindOrphanedDirectoriesToleratesVanishedRegion checks that a directory deleted between
// being resolved as a region and being read does not fail the whole scan. The library keeps
// being written to while it is walked.
func TestFindOrphanedDirectoriesToleratesVanishedRegion(t *testing.T) {
	videosDir := t.TempDir()

	archivedVideo := writeVideo(t, filepath.Join(videosDir, "channel_a", "video_1"), "video_1")
	writeVideo(t, filepath.Join(videosDir, "channel_b", "video_2"), "video_2")
	age(t, videosDir)

	referenced := resolve(References{
		VideosDir: videosDir,
		Videos: []VideoReference{
			{Paths: []string{archivedVideo}, ChannelFolders: []string{"channel_a"}},
		},
		Channels: []ChannelReference{
			{Folders: []string{"channel_a"}},
			{Folders: []string{"channel_b"}},
		},
	})

	require.NoError(t, os.RemoveAll(filepath.Join(videosDir, "channel_b")))

	orphans, err := findOrphanedDirectories(videosDir, referenced)
	require.NoError(t, err)
	assert.Empty(t, orphans)
}
