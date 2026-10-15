package storage_test

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/ent"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/server"
	"github.com/zibbp/ganymede/internal/storage"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/tests"
)

type StorageTest struct {
	App       *server.Application
	Service   *storage.Service
	VideosDir string
	Channel   *ent.Channel
}

// TestStorage runs the database backed storage tests against one container.
func TestStorage(t *testing.T) {
	app, err := tests.Setup(t)
	require.NoError(t, err)

	videosDir := os.Getenv("VIDEOS_DIR")
	require.NotEmpty(t, videosDir)

	dbChannel, err := app.ChannelService.CreateChannel(channel.Channel{
		ExtID:       "123456789",
		Name:        "test_channel",
		DisplayName: "Test Channel",
		ImagePath:   filepath.Join(videosDir, "test_channel", "profile.png"),
	})
	require.NoError(t, err)
	writeFile(t, dbChannel.ImagePath, 1)

	storageTest := StorageTest{
		App:       app,
		Service:   storage.NewService(app.Database, app.RiverClient, app.PlatformTwitch),
		VideosDir: videosDir,
		Channel:   dbChannel,
	}

	t.Run("TestReconcileAndStore", storageTest.ReconcileAndStoreTest)
	t.Run("TestVerifyFinding", storageTest.VerifyFindingTest)
	t.Run("TestDeleteFindings", storageTest.DeleteFindingsTest)
	t.Run("TestImportFindings", storageTest.ImportFindingsTest)
	t.Run("TestClaimedFinding", storageTest.ClaimedFindingTest)
}

func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, make([]byte, size), 0644))
}

func writeVideoFiles(t *testing.T, dir string, name string) string {
	t.Helper()
	video := filepath.Join(dir, name+"-video.mp4")
	writeFile(t, video, 10)
	writeFile(t, filepath.Join(dir, name+"-web_thumbnail.jpg"), 1)
	return video
}

// age moves everything below the root out of the grace period.
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

// archiveVideo creates a video row that points at files on disk, the way the archiver does.
func (s *StorageTest) archiveVideo(t *testing.T, extID string, folderName string) *ent.Vod {
	t.Helper()
	dir := filepath.Join(s.VideosDir, "test_channel", folderName)
	video := writeVideoFiles(t, dir, folderName)

	row, err := s.App.Database.Client.Vod.Create().
		SetChannel(s.Channel).
		SetExtID(extID).
		SetPlatform(utils.PlatformTwitch).
		SetType(utils.Archive).
		SetTitle("Test " + folderName).
		SetDuration(60).
		SetViews(1).
		SetVideoPath(video).
		SetWebThumbnailPath(filepath.Join(dir, folderName+"-web_thumbnail.jpg")).
		SetStreamedAt(time.Now().Add(-48 * time.Hour)).
		SetFolderName(folderName).
		SetFileName(folderName).
		Save(context.Background())
	require.NoError(t, err)
	return row
}

func (s *StorageTest) findings(t *testing.T) []*ent.StorageFinding {
	t.Helper()
	rows, err := s.App.Database.Client.StorageFinding.Query().All(context.Background())
	require.NoError(t, err)
	return rows
}

// ReconcileAndStoreTest checks that findings are stored, keep their id across runs, and are
// removed once the directory is gone.
func (s *StorageTest) ReconcileAndStoreTest(t *testing.T) {
	ctx := context.Background()

	s.archiveVideo(t, "1001", "video_1")
	orphan := filepath.Join(s.VideosDir, "test_channel", "video_2")
	writeVideoFiles(t, orphan, "video_2")
	age(t, s.VideosDir)

	count, err := storage.ReconcileAndStore(ctx, s.App.Database)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	rows := s.findings(t)
	require.Len(t, rows, 1)
	assert.Equal(t, orphan, rows[0].Path)
	assert.Equal(t, int64(11), rows[0].SizeBytes)
	id := rows[0].ID

	// A second run keeps the finding and its id.
	writeFile(t, filepath.Join(orphan, "video_2-chat.json"), 5)
	age(t, orphan)
	_, err = storage.ReconcileAndStore(ctx, s.App.Database)
	require.NoError(t, err)

	rows = s.findings(t)
	require.Len(t, rows, 1)
	assert.Equal(t, id, rows[0].ID)
	assert.Equal(t, int64(16), rows[0].SizeBytes)

	// Once the directory is gone, so is the finding.
	require.NoError(t, os.RemoveAll(orphan))
	count, err = storage.ReconcileAndStore(ctx, s.App.Database)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Empty(t, s.findings(t))
}

// VerifyFindingTest checks the targeted verification that runs before an action.
func (s *StorageTest) VerifyFindingTest(t *testing.T) {
	ctx := context.Background()

	video := s.archiveVideo(t, "1002", "video_3")
	orphan := filepath.Join(s.VideosDir, "test_channel", "video_4")
	writeVideoFiles(t, orphan, "video_4")
	writeVideoFiles(t, filepath.Join(filepath.Dir(video.VideoPath), "extras"), "bonus")
	age(t, s.VideosDir)

	assert.NoError(t, storage.VerifyFinding(ctx, s.App.Database, s.VideosDir, orphan))

	err := storage.VerifyFinding(ctx, s.App.Database, s.VideosDir, filepath.Dir(video.VideoPath))
	assert.ErrorContains(t, err, "references this directory")

	err = storage.VerifyFinding(ctx, s.App.Database, s.VideosDir, filepath.Join(s.VideosDir, "test_channel"))
	assert.ErrorContains(t, err, "top level")

	err = storage.VerifyFinding(ctx, s.App.Database, s.VideosDir, filepath.Join(s.VideosDir, "..", "elsewhere"))
	assert.ErrorContains(t, err, "outside")

	err = storage.VerifyFinding(ctx, s.App.Database, s.VideosDir, filepath.Join(filepath.Dir(video.VideoPath), "extras"))
	assert.ErrorContains(t, err, "inside the directory of a video")

	err = storage.VerifyFinding(ctx, s.App.Database, s.VideosDir, "test_channel/video_4")
	assert.ErrorContains(t, err, "not absolute")

	err = storage.VerifyFinding(ctx, s.App.Database, s.VideosDir, video.VideoPath)
	assert.ErrorContains(t, err, "not a directory")

	err = storage.VerifyFinding(ctx, s.App.Database, s.VideosDir, filepath.Join(os.Getenv("TEMP_DIR"), "anything"))
	assert.ErrorContains(t, err, "outside")

	linked := filepath.Join(s.VideosDir, "test_channel", "linked")
	require.NoError(t, os.Symlink(orphan, linked))
	err = storage.VerifyFinding(ctx, s.App.Database, s.VideosDir, linked)
	assert.ErrorContains(t, err, "not a directory")
	require.NoError(t, os.Remove(linked))

	// Files of a known video shield a directory the database does not point at.
	moved := filepath.Join(s.VideosDir, "test_channel", "moved")
	writeVideoFiles(t, moved, "video_3")
	age(t, moved)
	err = storage.VerifyFinding(ctx, s.App.Database, s.VideosDir, moved)
	assert.ErrorContains(t, err, "no longer a finding")

	require.NoError(t, os.RemoveAll(orphan))
	require.NoError(t, os.RemoveAll(moved))
}

// DeleteFindingsTest checks that a finding is deleted from disk and from the table, and that
// a stale finding is refused.
func (s *StorageTest) DeleteFindingsTest(t *testing.T) {
	ctx := context.Background()

	s.archiveVideo(t, "1003", "video_5")
	orphan := filepath.Join(s.VideosDir, "test_channel", "video_6")
	writeVideoFiles(t, orphan, "video_6")
	age(t, s.VideosDir)

	_, err := storage.ReconcileAndStore(ctx, s.App.Database)
	require.NoError(t, err)
	rows := s.findings(t)
	require.Len(t, rows, 1)

	// The directory was re-archived in the meantime, so the finding is stale.
	stale, err := s.App.Database.Client.StorageFinding.Create().
		SetKind("orphaned_directory").
		SetPath(filepath.Join(s.VideosDir, "test_channel", "video_5")).
		Save(ctx)
	require.NoError(t, err)

	result, err := s.Service.DeleteFindings(ctx, []uuid.UUID{rows[0].ID, stale.ID}, "tester")
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{rows[0].ID}, result.Done)
	require.Len(t, result.Failed, 1)
	assert.Equal(t, stale.ID, result.Failed[0].ID)

	assert.NoDirExists(t, orphan)
	assert.FileExists(t, filepath.Join(s.VideosDir, "test_channel", "video_5", "video_5-video.mp4"))

	// Both findings were claimed, so the table is empty either way. The stale one comes back
	// with the next scan if it still qualifies.
	assert.Empty(t, s.findings(t))
}

// ClaimedFindingTest checks that a finding can only be acted on once, so two administrators
// cannot delete and import the same directory at the same time.
func (s *StorageTest) ClaimedFindingTest(t *testing.T) {
	ctx := context.Background()

	s.archiveVideo(t, "1006", "video_9")
	orphan := filepath.Join(s.VideosDir, "test_channel", "video_10")
	writeVideoFiles(t, orphan, "video_10")
	age(t, s.VideosDir)

	_, err := storage.ReconcileAndStore(ctx, s.App.Database)
	require.NoError(t, err)
	rows := s.findings(t)
	require.Len(t, rows, 1)
	id := rows[0].ID

	first, err := s.Service.DeleteFindings(ctx, []uuid.UUID{id}, "tester")
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{id}, first.Done)

	// The same finding a second time belongs to nobody any more.
	second, err := s.Service.DeleteFindings(ctx, []uuid.UUID{id}, "tester")
	require.NoError(t, err)
	assert.Empty(t, second.Done)
	require.Len(t, second.Failed, 1)
	assert.Equal(t, "finding not found", second.Failed[0].Error)

	assert.NoDirExists(t, orphan)
}

// ImportFindingsTest checks that a directory with an info file becomes a video again.
func (s *StorageTest) ImportFindingsTest(t *testing.T) {
	ctx := context.Background()

	s.archiveVideo(t, "1004", "video_7")
	orphan := filepath.Join(s.VideosDir, "test_channel", "video_8")
	writeVideoFiles(t, orphan, "video_8")
	writeFile(t, filepath.Join(orphan, "video_8-chat.json"), 3)

	info := platform.VideoInfo{
		ID:        "1005",
		UserID:    s.Channel.ExtID,
		UserLogin: s.Channel.Name,
		UserName:  s.Channel.DisplayName,
		Title:     "Imported video",
		URL:       "https://www.twitch.tv/videos/1005",
		Type:      "archive",
		CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		ViewCount: 42,
		Duration:  90 * time.Minute,
		MutedSegments: []platform.MutedSegment{
			{Offset: 10, Duration: 20},
		},
	}
	data, err := json.Marshal(info)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(orphan, "video_8-info.json"), data, 0644))
	age(t, s.VideosDir)

	_, err = storage.ReconcileAndStore(ctx, s.App.Database)
	require.NoError(t, err)
	rows := s.findings(t)
	require.Len(t, rows, 1)

	result, err := s.Service.ImportFindings(ctx, []uuid.UUID{rows[0].ID}, "tester")
	require.NoError(t, err)
	assert.Empty(t, result.Failed)
	assert.Equal(t, []uuid.UUID{rows[0].ID}, result.Done)

	imported, err := s.App.Database.Client.Vod.Query().Where(entVod.ExtID("1005")).WithChannel().WithMutedSegments().Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Imported video", imported.Title)
	assert.Equal(t, utils.Archive, imported.Type)
	assert.Equal(t, 5400, imported.Duration)
	assert.Equal(t, 42, imported.Views)
	assert.False(t, imported.Processing)
	assert.Equal(t, filepath.Join(orphan, "video_8-video.mp4"), imported.VideoPath)
	assert.Equal(t, filepath.Join(orphan, "video_8-web_thumbnail.jpg"), imported.WebThumbnailPath)
	assert.Equal(t, filepath.Join(orphan, "video_8-chat.json"), imported.ChatPath)
	assert.Equal(t, filepath.Join(orphan, "video_8-info.json"), imported.InfoPath)
	assert.Equal(t, "video_8", imported.FolderName)
	assert.Equal(t, "video_8", imported.FileName)
	assert.Equal(t, s.Channel.ID, imported.Edges.Channel.ID)
	require.Len(t, imported.Edges.MutedSegments, 1)
	assert.Equal(t, 30, imported.Edges.MutedSegments[0].End)

	assert.Empty(t, s.findings(t))

	// Importing the same video twice is refused.
	again, err := s.App.Database.Client.StorageFinding.Create().SetKind("orphaned_directory").SetPath(orphan).Save(ctx)
	require.NoError(t, err)
	result, err = s.Service.ImportFindings(ctx, []uuid.UUID{again.ID}, "tester")
	require.NoError(t, err)
	require.Len(t, result.Failed, 1)
	assert.Contains(t, result.Failed[0].Error, "references this directory")
}
