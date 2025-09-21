package archive_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/queue"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/server"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/tests"
	tests_shared "github.com/zibbp/ganymede/tests/shared"
)

type ArchiveTest struct {
	App *server.Application
}

var (
	TestTwitchChannelName        = "sodapoppin"
	TestTwitchChannelDisplayName = "sodapoppin"
	TestTwitchChannelExtId       = "26301881"
	TestTwitchVideoId            = "1989753443"
	TestTwitchClipId             = "SarcasticDarkPanCoolCat-rgyYByzzfGqIwbWd"
	TestArchiveTimeout           = 300 * time.Second
)

// TestArchive runs all archive tests
func TestArchive(t *testing.T) {
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	archiveTest := ArchiveTest{App: app}

	t.Run("TestArchiveChannel", archiveTest.ArchiveChannelTest)

}

// ArchiveChannelTest tests the ArchiveChannel function
func (s *ArchiveTest) ArchiveChannelTest(t *testing.T) {
	archivedPlatformChannel, err := s.App.ArchiveService.ArchiveChannel(context.Background(), TestTwitchChannelName)
	assert.NoError(t, err)

	assert.Equal(t, TestTwitchChannelName, archivedPlatformChannel.Name)
	assert.Equal(t, TestTwitchChannelDisplayName, archivedPlatformChannel.DisplayName)
	assert.Equal(t, TestTwitchChannelExtId, archivedPlatformChannel.ExtID)

	// Check if profile image was download
	assert.FileExists(t, archivedPlatformChannel.ImagePath)

	// Check if profile image is not empty
	fileInfo, err := os.Stat(archivedPlatformChannel.ImagePath)
	assert.NoError(t, err)
	assert.NotEqual(t, 0, fileInfo.Size())
}

// ArchiveVideo tests the full archive process for a video with chat downloading, processing, and rendering
// This asserts the files are created, the queue item is created, and the video is playable.
// It also tests the deletion of the video and its associated files.
func TestArchiveVideo(t *testing.T) {
	// Setup the application
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	// Archive the video
	_, err = app.ArchiveService.ArchiveVideo(context.Background(), archive.ArchiveVideoInput{
		VideoId:     TestTwitchVideoId,
		Quality:     utils.R720,
		ArchiveChat: true,
		RenderChat:  true,
	})
	assert.NoError(t, err)

	// Assert video was created
	v, err := app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchVideoId)).WithChapters().Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	// Assert queue item was created
	q, err := app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	assert.Equal(t, true, q.ChatProcessing)
	assert.Equal(t, true, q.VideoProcessing)
	assert.Equal(t, true, q.RenderChat)
	assert.Equal(t, true, q.ArchiveChat)
	assert.NotNil(t, q.WorkflowID)
	assert.NotNil(t, q.WorkflowRunID)
	assert.Equal(t, utils.Pending, q.TaskChatDownload)
	assert.Equal(t, utils.Pending, q.TaskChatRender)
	assert.Equal(t, utils.Pending, q.TaskChatMove)
	assert.Equal(t, utils.Pending, q.TaskVideoDownload)
	assert.Equal(t, utils.Pending, q.TaskVideoConvert)
	assert.Equal(t, utils.Pending, q.TaskVideoMove)

	// Wait for the video to be archived
	tests_shared.WaitForArchiveCompletion(t, app, v.ID, TestArchiveTimeout)

	// Requery video
	v, err = app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchVideoId)).WithChapters().Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	// Assert queue item was updated
	q, err = app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, q)
	assert.Equal(t, false, q.ChatProcessing)
	assert.Equal(t, false, q.VideoProcessing)
	assert.Equal(t, utils.Success, q.TaskChatDownload)
	assert.Equal(t, utils.Success, q.TaskChatRender)
	assert.Equal(t, utils.Success, q.TaskChatMove)
	assert.Equal(t, utils.Success, q.TaskVideoDownload)
	assert.Equal(t, utils.Success, q.TaskVideoConvert)
	assert.Equal(t, utils.Success, q.TaskVideoMove)

	// Assert files exist
	assert.FileExists(t, v.ThumbnailPath)
	assert.FileExists(t, v.WebThumbnailPath)
	assert.FileExists(t, v.VideoPath)
	assert.FileExists(t, v.ChatPath)
	assert.FileExists(t, v.ChatVideoPath)

	assert.NotEqual(t, 0, v.StorageSizeBytes)

	// Assert video is playable
	assert.True(t, tests_shared.IsPlayableVideo(v.VideoPath), "Video file is not playable")
	assert.True(t, tests_shared.IsPlayableVideo(v.ChatVideoPath), "Video file is not playable")

	// Assert chat files is greater than 0 bytes
	chatFileInfo, err := os.Stat(v.ChatPath)
	assert.NoError(t, err)
	assert.Greater(t, chatFileInfo.Size(), int64(0), "Chat file should not be empty")

	// Assert info file is greater than 0 bytes
	infoFileInfo, err := os.Stat(v.InfoPath)
	assert.NoError(t, err)
	assert.Greater(t, infoFileInfo.Size(), int64(0), "Info file should not be empty")

	// Assert thumbnail is greater than 0 bytes
	thumbnailFileInfo, err := os.Stat(v.ThumbnailPath)
	assert.NoError(t, err)
	assert.Greater(t, thumbnailFileInfo.Size(), int64(0), "Thumbnail file should not be empty")

	// Assert web thumbnail is greater than 0 bytes
	webThumbnailFileInfo, err := os.Stat(v.WebThumbnailPath)
	assert.NoError(t, err)
	assert.Greater(t, webThumbnailFileInfo.Size(), int64(0), "Web thumbnail file should not be empty")

	// Assert sprite thumbnail facts
	v, err = app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchVideoId)).WithChapters().Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)
	assert.Len(t, v.SpriteThumbnailsImages, 1, "Sprite thumbnails should be generated for videos")

	// Assert at least one chapter exists
	assert.NotEmpty(t, v.Edges.Chapters, "Expected at least one chapter to be present")

	// Test delete and ensure directory is removed
	videoDirectory := filepath.Dir(filepath.Clean(v.VideoPath))

	err = app.VodService.DeleteVod(t.Context(), v.ID, true)
	assert.NoError(t, err)

	// Assert video directory is removed
	_, err = os.Stat(videoDirectory)
	assert.Error(t, err)
	if !os.IsNotExist(err) {
		t.Fatalf("Expected video directory %s to be removed, but it still exists: %v", videoDirectory, err)
	}

	// Assert video was deleted from database
	_, err = app.Database.Client.Vod.Query().Where(vod.ID(v.ID)).Only(context.Background())
	assert.Error(t, err)
	if _, ok := err.(*ent.NotFoundError); !ok {
		t.Fatalf("Expected vod to be deleted, but it still exists: %v", err)
	}

	// Assert queue item was deleted
	_, err = app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
	assert.Error(t, err)
	if _, ok := err.(*ent.NotFoundError); !ok {
		t.Fatalf("Expected queue item to be deleted, but it still exists: %v", err)
	}

}

// ArchiveVideo tests the full archive process for a video without chat downloading, processing, and rendering
func TestArchiveVideoNoChat(t *testing.T) {
	// Setup the application
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	// Archive the video
	_, err = app.ArchiveService.ArchiveVideo(context.Background(), archive.ArchiveVideoInput{
		VideoId:     TestTwitchVideoId,
		Quality:     utils.R720,
		ArchiveChat: false,
		RenderChat:  false,
	})
	assert.NoError(t, err)

	// Assert video was created
	v, err := app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchVideoId)).WithChapters().Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	// Assert queue item was created
	q, err := app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	assert.Equal(t, false, q.ChatProcessing)
	assert.Equal(t, true, q.VideoProcessing)
	assert.Equal(t, false, q.RenderChat)
	assert.Equal(t, false, q.ArchiveChat)
	assert.NotNil(t, q.WorkflowID)
	assert.NotNil(t, q.WorkflowRunID)
	assert.Equal(t, utils.Success, q.TaskChatDownload)
	assert.Equal(t, utils.Success, q.TaskChatRender)
	assert.Equal(t, utils.Success, q.TaskChatMove)
	assert.Equal(t, utils.Pending, q.TaskVideoDownload)
	assert.Equal(t, utils.Pending, q.TaskVideoConvert)
	assert.Equal(t, utils.Pending, q.TaskVideoMove)

	// Wait for the video to be archived
	tests_shared.WaitForArchiveCompletion(t, app, v.ID, TestArchiveTimeout)

	// Requery video
	v, err = app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchVideoId)).WithChapters().Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	// Assert queue item was updated
	q, err = app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, false, q.ChatProcessing)
	assert.Equal(t, false, q.VideoProcessing)
	assert.Equal(t, utils.Success, q.TaskChatDownload)
	assert.Equal(t, utils.Success, q.TaskChatRender)
	assert.Equal(t, utils.Success, q.TaskChatMove)
	assert.Equal(t, utils.Success, q.TaskVideoDownload)
	assert.Equal(t, utils.Success, q.TaskVideoConvert)
	assert.Equal(t, utils.Success, q.TaskVideoMove)

	// Assert files exist
	assert.FileExists(t, v.ThumbnailPath)
	assert.FileExists(t, v.WebThumbnailPath)
	assert.FileExists(t, v.VideoPath)
	assert.NoFileExists(t, v.ChatPath)
	assert.NoFileExists(t, v.ChatVideoPath)

	// Assert at least one chapter exists
	assert.NotEmpty(t, v.Edges.Chapters, "Expected at least one chapter to be present")

	assert.NotEqual(t, 0, v.StorageSizeBytes)

	// Assert video is playable
	assert.True(t, tests_shared.IsPlayableVideo(v.VideoPath), "Video file is not playable")
}

// ArchiveVideo tests the full archive process for a video without chat rendering
func TestArchiveVideoNoChatRender(t *testing.T) {
	// Setup the application
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	// Archive the video
	_, err = app.ArchiveService.ArchiveVideo(context.Background(), archive.ArchiveVideoInput{
		VideoId:     TestTwitchVideoId,
		Quality:     utils.R720,
		ArchiveChat: true,
		RenderChat:  false,
	})
	assert.NoError(t, err)

	// Assert video was created
	v, err := app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchVideoId)).WithChapters().Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	// Assert queue item was created
	q, err := app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	assert.Equal(t, true, q.ChatProcessing)
	assert.Equal(t, true, q.VideoProcessing)
	assert.Equal(t, false, q.RenderChat)
	assert.Equal(t, true, q.ArchiveChat)
	assert.NotNil(t, q.WorkflowID)
	assert.NotNil(t, q.WorkflowRunID)
	assert.Equal(t, utils.Pending, q.TaskChatDownload)
	assert.Equal(t, utils.Success, q.TaskChatRender)
	assert.Equal(t, utils.Pending, q.TaskChatMove)
	assert.Equal(t, utils.Pending, q.TaskVideoDownload)
	assert.Equal(t, utils.Pending, q.TaskVideoConvert)
	assert.Equal(t, utils.Pending, q.TaskVideoMove)

	// Wait for the video to be archived
	tests_shared.WaitForArchiveCompletion(t, app, v.ID, TestArchiveTimeout)

	// Requery video
	v, err = app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchVideoId)).WithChapters().Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	// Assert queue item was updated
	q, err = app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, false, q.ChatProcessing)
	assert.Equal(t, false, q.VideoProcessing)
	assert.Equal(t, utils.Success, q.TaskChatDownload)
	assert.Equal(t, utils.Success, q.TaskChatRender)
	assert.Equal(t, utils.Success, q.TaskChatMove)
	assert.Equal(t, utils.Success, q.TaskVideoDownload)
	assert.Equal(t, utils.Success, q.TaskVideoConvert)
	assert.Equal(t, utils.Success, q.TaskVideoMove)

	// Assert files exist
	assert.FileExists(t, v.ThumbnailPath)
	assert.FileExists(t, v.WebThumbnailPath)
	assert.FileExists(t, v.VideoPath)
	assert.FileExists(t, v.ChatPath)
	assert.NoFileExists(t, v.ChatVideoPath)

	// Assert at least one chapter exists
	assert.NotEmpty(t, v.Edges.Chapters, "Expected at least one chapter to be present")

	assert.NotEqual(t, 0, v.StorageSizeBytes)

	// Assert video is playable
	assert.True(t, tests_shared.IsPlayableVideo(v.VideoPath), "Video file is not playable")
}

// TestArchiveVideoHLS tests the full archive process for a video without chat downloading, processing, and rendering converting to HLS
func TestArchiveVideoHLS(t *testing.T) {
	// Setup the application
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	// Update config to save as HLS
	c := config.Get()
	c.Archive.SaveAsHls = true
	assert.NoError(t, config.UpdateConfig(c))

	// Archive the video
	_, err = app.ArchiveService.ArchiveVideo(context.Background(), archive.ArchiveVideoInput{
		VideoId:     TestTwitchVideoId,
		Quality:     utils.R720,
		ArchiveChat: false,
		RenderChat:  false,
	})
	assert.NoError(t, err)

	// Assert video was created
	v, err := app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchVideoId)).WithChapters().Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	assert.NotNil(t, v.TmpVideoHlsPath)

	// Assert queue item was created
	q, err := app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	assert.Equal(t, false, q.ChatProcessing)
	assert.Equal(t, true, q.VideoProcessing)
	assert.Equal(t, false, q.RenderChat)
	assert.Equal(t, false, q.ArchiveChat)
	assert.NotNil(t, q.WorkflowID)
	assert.NotNil(t, q.WorkflowRunID)
	assert.Equal(t, utils.Success, q.TaskChatDownload)
	assert.Equal(t, utils.Success, q.TaskChatRender)
	assert.Equal(t, utils.Success, q.TaskChatMove)
	assert.Equal(t, utils.Pending, q.TaskVideoDownload)
	assert.Equal(t, utils.Pending, q.TaskVideoConvert)
	assert.Equal(t, utils.Pending, q.TaskVideoMove)

	// Wait for the video to be archived
	tests_shared.WaitForArchiveCompletion(t, app, v.ID, TestArchiveTimeout)

	// Requery video
	v, err = app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchVideoId)).WithChapters().Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	// Assert queue item was updated
	q, err = app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, false, q.ChatProcessing)
	assert.Equal(t, false, q.VideoProcessing)
	assert.Equal(t, utils.Success, q.TaskChatDownload)
	assert.Equal(t, utils.Success, q.TaskChatRender)
	assert.Equal(t, utils.Success, q.TaskChatMove)
	assert.Equal(t, utils.Success, q.TaskVideoDownload)
	assert.Equal(t, utils.Success, q.TaskVideoConvert)
	assert.Equal(t, utils.Success, q.TaskVideoMove)

	// Assert files exist
	assert.FileExists(t, v.ThumbnailPath)
	assert.FileExists(t, v.WebThumbnailPath)
	assert.NoFileExists(t, v.ChatPath)
	assert.NoFileExists(t, v.ChatVideoPath)

	assert.NotEqual(t, 0, v.StorageSizeBytes)

	// Assert at least one chapter exists
	assert.NotEmpty(t, v.Edges.Chapters, "Expected at least one chapter to be present")

	// Assert video is playable
	assert.True(t, tests_shared.IsPlayableVideo(v.VideoPath), "Video file is not playable")

	assert.DirExists(t, v.VideoHlsPath)

	// Assert number of files in HLS directory is greater than 0
	files, err := os.ReadDir(v.VideoHlsPath)
	assert.NoError(t, err)
	assert.Greater(t, len(files), 0)

	// Test delete and ensure directory is removed
	videoDirectory := filepath.Dir(filepath.Clean(v.VideoHlsPath))

	err = app.VodService.DeleteVod(t.Context(), v.ID, true)
	assert.NoError(t, err)

	// Assert video directory is removed
	_, err = os.Stat(videoDirectory)
	assert.Error(t, err)
	if !os.IsNotExist(err) {
		t.Fatalf("Expected video directory %s to be removed, but it still exists: %v", videoDirectory, err)
	}

	// Assert video was deleted from database
	_, err = app.Database.Client.Vod.Query().Where(vod.ID(v.ID)).Only(context.Background())
	assert.Error(t, err)
	if _, ok := err.(*ent.NotFoundError); !ok {
		t.Fatalf("Expected vod to be deleted, but it still exists: %v", err)
	}

	// Assert queue item was deleted
	_, err = app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
	assert.Error(t, err)
	if _, ok := err.(*ent.NotFoundError); !ok {
		t.Fatalf("Expected queue item to be deleted, but it still exists: %v", err)
	}
}

// ArchiveVideo tests the full archive process for a video with chat downloading, processing, and rendering
func TestArchiveClip(t *testing.T) {
	// Setup the application
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	// Archive the video
	_, err = app.ArchiveService.ArchiveClip(context.Background(), archive.ArchiveClipInput{
		ID:          TestTwitchClipId,
		Quality:     utils.R720,
		ArchiveChat: true,
		RenderChat:  true,
	})
	assert.NoError(t, err)

	// Assert video was created
	v, err := app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchClipId)).Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	// Assert queue item was created
	q, err := app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	assert.Equal(t, true, q.ChatProcessing)
	assert.Equal(t, true, q.VideoProcessing)
	assert.Equal(t, true, q.RenderChat)
	assert.Equal(t, true, q.ArchiveChat)
	assert.NotNil(t, q.WorkflowID)
	assert.NotNil(t, q.WorkflowRunID)
	assert.Equal(t, utils.Pending, q.TaskChatDownload)
	assert.Equal(t, utils.Pending, q.TaskChatRender)
	assert.Equal(t, utils.Pending, q.TaskChatMove)
	assert.Equal(t, utils.Pending, q.TaskVideoDownload)
	assert.Equal(t, utils.Pending, q.TaskVideoConvert)
	assert.Equal(t, utils.Pending, q.TaskVideoMove)

	// Wait for the video to be archived
	tests_shared.WaitForArchiveCompletion(t, app, v.ID, TestArchiveTimeout)

	// Requery video
	v, err = app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchClipId)).WithChapters().Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	// Assert queue item was updated
	q, err = app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, false, q.ChatProcessing)
	assert.Equal(t, false, q.VideoProcessing)
	assert.Equal(t, utils.Success, q.TaskChatDownload)
	assert.Equal(t, utils.Success, q.TaskChatRender)
	assert.Equal(t, utils.Success, q.TaskChatMove)
	assert.Equal(t, utils.Success, q.TaskVideoDownload)
	assert.Equal(t, utils.Success, q.TaskVideoConvert)
	assert.Equal(t, utils.Success, q.TaskVideoMove)

	// Assert files exist
	assert.FileExists(t, v.ThumbnailPath)
	assert.FileExists(t, v.WebThumbnailPath)
	assert.FileExists(t, v.VideoPath)
	assert.FileExists(t, v.ChatPath)
	assert.FileExists(t, v.ChatVideoPath)

	assert.NotEqual(t, 0, v.StorageSizeBytes)

	// Assert video is playable
	assert.True(t, tests_shared.IsPlayableVideo(v.VideoPath), "Video file is not playable")
	assert.True(t, tests_shared.IsPlayableVideo(v.ChatVideoPath), "Video file is not playable")
}

// TestArchiveVideoWithSpriteThumbnails tests generate sprite thumbnails after a video is archived.
func TestArchiveVideoWithSpriteThumbnails(t *testing.T) {
	// Setup the application
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	// Archive the video
	_, err = app.ArchiveService.ArchiveVideo(context.Background(), archive.ArchiveVideoInput{
		VideoId:     TestTwitchVideoId,
		Quality:     utils.R720,
		ArchiveChat: false,
		RenderChat:  false,
	})
	assert.NoError(t, err)

	// Assert video was created
	v, err := app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchVideoId)).WithChapters().Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	// Assert queue item was created
	q, err := app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	assert.Equal(t, false, q.ChatProcessing)
	assert.Equal(t, true, q.VideoProcessing)
	assert.Equal(t, false, q.RenderChat)
	assert.Equal(t, false, q.ArchiveChat)
	assert.NotNil(t, q.WorkflowID)
	assert.NotNil(t, q.WorkflowRunID)
	assert.Equal(t, utils.Success, q.TaskChatDownload)
	assert.Equal(t, utils.Success, q.TaskChatRender)
	assert.Equal(t, utils.Success, q.TaskChatMove)
	assert.Equal(t, utils.Pending, q.TaskVideoDownload)
	assert.Equal(t, utils.Pending, q.TaskVideoConvert)
	assert.Equal(t, utils.Pending, q.TaskVideoMove)

	// Wait for the video to be archived
	tests_shared.WaitForArchiveCompletion(t, app, v.ID, TestArchiveTimeout)

	// Requery video
	v, err = app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchVideoId)).WithChapters().Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	// Assert queue item was updated
	q, err = app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)
	assert.Equal(t, false, q.ChatProcessing)
	assert.Equal(t, false, q.VideoProcessing)
	assert.Equal(t, utils.Success, q.TaskChatDownload)
	assert.Equal(t, utils.Success, q.TaskChatRender)
	assert.Equal(t, utils.Success, q.TaskChatMove)
	assert.Equal(t, utils.Success, q.TaskVideoDownload)
	assert.Equal(t, utils.Success, q.TaskVideoConvert)
	assert.Equal(t, utils.Success, q.TaskVideoMove)

	// Assert files exist
	assert.FileExists(t, v.ThumbnailPath)
	assert.FileExists(t, v.WebThumbnailPath)
	assert.FileExists(t, v.VideoPath)
	assert.NoFileExists(t, v.ChatPath)
	assert.NoFileExists(t, v.ChatVideoPath)

	assert.NotEqual(t, 0, v.StorageSizeBytes)

	// Assert at least one chapter exists
	assert.NotEmpty(t, v.Edges.Chapters, "Expected at least one chapter to be present")

	// Assert sprite thumbnail facts
	v, err = app.Database.Client.Vod.Query().Where(vod.ID(v.ID)).Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v.SpriteThumbnailsColumns)
	assert.NotNil(t, v.SpriteThumbnailsRows)
	assert.NotNil(t, v.SpriteThumbnailsHeight)
	assert.NotNil(t, v.SpriteThumbnailsWidth)
	assert.NotNil(t, v.SpriteThumbnailsInterval)
	if len(v.SpriteThumbnailsImages) == 0 {
		t.Errorf("expected more than 0 sprite thumbnails")
	}

	for _, spriteThumbnailPath := range v.SpriteThumbnailsImages {
		assert.FileExists(t, spriteThumbnailPath)
	}
}

// TestArchiveVideoStorageTemplateSettings tests the storage template settings for archiving videos
func TestArchiveVideoStorageTemplateSettings(t *testing.T) {
	// Setup the application
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	// Archive the video
	_, err = app.ArchiveService.ArchiveVideo(context.Background(), archive.ArchiveVideoInput{
		VideoId:     TestTwitchVideoId,
		Quality:     utils.R720,
		ArchiveChat: false,
		RenderChat:  false,
	})
	assert.NoError(t, err)

	// Assert video was created
	v, err := app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchVideoId)).WithChannel().Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	// Assert storage template settings were applied
	expectedFolderName, err := archive.GetFolderName(v.ID, archive.StorageTemplateInput{
		UUID:    v.ID,
		ID:      v.ExtID,
		Channel: v.Edges.Channel.Name,
		Title:   v.Title,
		Type:    string(v.Type),
		Date:    v.StreamedAt.Format("2006-01-02"),
		YYYY:    v.StreamedAt.Format("2006"),
		MM:      v.StreamedAt.Format("01"),
		DD:      v.StreamedAt.Format("02"),
		HH:      v.StreamedAt.Format("15"),
	})
	assert.NoError(t, err)
	expectedFileName, err := archive.GetFileName(v.ID, archive.StorageTemplateInput{
		UUID:    v.ID,
		ID:      v.ExtID,
		Channel: v.Edges.Channel.Name,
		Title:   v.Title,
		Type:    string(v.Type),
		Date:    v.StreamedAt.Format("2006-01-02"),
		YYYY:    v.StreamedAt.Format("2006"),
		MM:      v.StreamedAt.Format("01"),
		DD:      v.StreamedAt.Format("02"),
		HH:      v.StreamedAt.Format("15"),
	})
	assert.NoError(t, err)
	assert.Equal(t, v.FolderName, expectedFolderName, "Folder name should match the expected storage template")
	assert.Equal(t, v.FileName, expectedFileName, "File name should match the expected storage template")
}

// TestArchiveVideoStorageTemplateSettingsCustom tests the custom storage template settings for archiving videos
func TestArchiveVideoStorageTemplateSettingsCustom(t *testing.T) {
	// Setup the application
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	c := config.Get()
	// Set a custom storage template for testing
	c.StorageTemplates.FolderTemplate = "{{YYYY}}{{MM}}-{{DD}}{{HH}} - {{title}}"
	c.StorageTemplates.FileTemplate = "{{title}}_{{id}}_{{uuid}}"
	assert.NoError(t, config.UpdateConfig(c), "failed to update config with custom template")

	// Archive the video
	_, err = app.ArchiveService.ArchiveVideo(context.Background(), archive.ArchiveVideoInput{
		VideoId:     TestTwitchVideoId,
		Quality:     utils.R720,
		ArchiveChat: false,
		RenderChat:  false,
	})
	assert.NoError(t, err)

	// Assert video was created
	v, err := app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchVideoId)).WithChannel().Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	// Assert storage template settings were applied
	expectedFolderName, err := archive.GetFolderName(v.ID, archive.StorageTemplateInput{
		UUID:    v.ID,
		ID:      v.ExtID,
		Channel: v.Edges.Channel.Name,
		Title:   v.Title,
		Type:    string(v.Type),
		Date:    v.StreamedAt.Format("2006-01-02"),
		YYYY:    v.StreamedAt.Format("2006"),
		MM:      v.StreamedAt.Format("01"),
		DD:      v.StreamedAt.Format("02"),
		HH:      v.StreamedAt.Format("15"),
	})
	assert.NoError(t, err)
	expectedFileName, err := archive.GetFileName(v.ID, archive.StorageTemplateInput{
		UUID:    v.ID,
		ID:      v.ExtID,
		Channel: v.Edges.Channel.Name,
		Title:   v.Title,
		Type:    string(v.Type),
		Date:    v.StreamedAt.Format("2006-01-02"),
		YYYY:    v.StreamedAt.Format("2006"),
		MM:      v.StreamedAt.Format("01"),
		DD:      v.StreamedAt.Format("02"),
		HH:      v.StreamedAt.Format("15"),
	})
	assert.NoError(t, err)
	fmt.Printf("Expected folder name: %s, expected file name: %s\n", expectedFolderName, expectedFileName)
	fmt.Printf("Actual folder name: %s, actual file name: %s\n", v.FolderName, v.FileName)
	assert.Equal(t, v.FolderName, expectedFolderName, "Folder name should match the expected storage template")
	assert.Equal(t, v.FileName, expectedFileName, "File name should match the expected storage template")
}
