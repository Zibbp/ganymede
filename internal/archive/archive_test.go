package archive_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/queue"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/config"
	internalExec "github.com/zibbp/ganymede/internal/exec"
	"github.com/zibbp/ganymede/internal/server"
	"github.com/zibbp/ganymede/internal/tasks"
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
	TestTwitchClipId             = "PatientSeductiveInternBIRB-55ziJ1I-ayol2RHH"
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

func assertAudioOnlyFile(t *testing.T, path string) {
	t.Helper()

	probeData, err := internalExec.GetFfprobeVideoData(t.Context(), path)
	assert.NoError(t, err, "Failed to probe archived media")
	assert.NotNil(t, probeData, "Expected ffprobe data for archived media")

	audioStreams := 0
	videoStreams := 0
	for _, stream := range probeData.Streams {
		switch stream.CodecType {
		case "audio":
			audioStreams++
		case "video":
			videoStreams++
		}
	}

	assert.Greater(t, audioStreams, 0, "Archived media should contain at least one audio stream")
	assert.Zero(t, videoStreams, "Archived media should not contain video streams")
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

// TestArchiveVideoRecoversAfterWorkerCrash verifies that a hard worker crash
// during a real yt-dlp transfer is recovered by the archive watchdog. The
// worker runs in a subprocess so SIGKILL cannot trigger River's graceful stop
// path.
func TestArchiveVideoRecoversAfterWorkerCrash(t *testing.T) {
	app, err := tests.SetupWithoutWorker(t)
	require.NoError(t, err)

	workerProcess := tests.StartCrashableWorker(t, "TestWorkerCrashHelper")

	_, err = app.ArchiveService.ArchiveVideo(t.Context(), archive.ArchiveVideoInput{
		VideoId:     TestTwitchVideoId,
		Quality:     utils.R160,
		ArchiveChat: false,
		RenderChat:  false,
	})
	require.NoError(t, err)

	v, err := app.Database.Client.Vod.Query().
		Where(vod.ExtID(TestTwitchVideoId)).
		Only(t.Context())
	require.NoError(t, err)

	q, err := app.Database.Client.Queue.Query().
		Where(queue.HasVodWith(vod.ID(v.ID))).
		Only(t.Context())
	require.NoError(t, err)

	capturedBytes := tests_shared.WaitForRunningVideoDownload(t, app, q.ID, v.TmpVideoDownloadPath, 1, 90*time.Second)
	originalJob := tests_shared.FindArchiveJob(
		t,
		app,
		q.ID,
		string(utils.TaskDownloadVideo),
		rivertype.JobStateRunning,
	)
	require.NotNil(t, originalJob, "expected the original VOD download River job")

	require.NoError(t, workerProcess.Crash())
	tests_shared.WaitForProcessExit(t, v.ID.String(), 5*time.Second)
	info, err := os.Stat(v.TmpVideoDownloadPath)
	require.NoError(t, err)
	require.GreaterOrEqual(t, info.Size(), capturedBytes)

	_ = tests.StartCrashableWorker(t, "TestWorkerCrashHelper")

	// Advance only the persisted heartbeat used by this isolated fixture. This
	// avoids sleeping through the production 90-second timeout while still
	// exercising the real watchdog cancellation and replacement paths.
	_, err = app.RiverClient.Client.JobUpdate(t.Context(), originalJob.ID, &river.JobUpdateParams{
		Output: tasks.ArchiveProgressOutput{HeartbeatAt: time.Now().Add(-5 * time.Minute)},
	})
	require.NoError(t, err)
	_, err = app.RiverClient.Insert(t.Context(), tasks.WatchdogArgs{}, nil)
	require.NoError(t, err)

	tests_shared.WaitForArchiveJobCancellation(t, app, originalJob.ID, 30*time.Second)
	q = tests_shared.WaitForArchiveCompletionAfterCrash(t, app, v.ID, TestArchiveTimeout)
	recoveredJob := tests_shared.WaitForCompletedArchiveRecovery(
		t,
		app,
		q.ID,
		string(utils.TaskDownloadVideo),
		30*time.Second,
	)
	require.NotEqual(t, originalJob.ID, recoveredJob.ID)

	v = tests_shared.WaitForArchiveMetadataFinalization(t, app, v.ID, 30*time.Second)

	require.False(t, q.Processing)
	require.False(t, q.VideoProcessing)
	require.Equal(t, utils.Success, q.TaskVideoDownload)
	require.Equal(t, utils.Success, q.TaskVideoConvert)
	require.Equal(t, utils.Success, q.TaskVideoMove)
	require.FileExists(t, v.VideoPath)
	require.True(t, tests_shared.IsPlayableVideo(v.VideoPath), "recovered VOD is not playable")
	require.NotZero(t, v.StorageSizeBytes)
	require.NotEmpty(t, v.Edges.Chapters)
}

// TestWorkerCrashHelper is executed in a child copy of this test binary by
// StartCrashableWorker. During an ordinary package test it returns immediately.
func TestWorkerCrashHelper(t *testing.T) {
	tests.RunWorkerCrashHelper(t)
}

// ArchiveVideo tests the full archive process for an audio-only video without chat downloading, processing, and rendering.
func TestArchiveVideoAudioOnlyNoChat(t *testing.T) {
	// Setup the application
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	// Archive the video
	_, err = app.ArchiveService.ArchiveVideo(context.Background(), archive.ArchiveVideoInput{
		VideoId:     TestTwitchVideoId,
		Quality:     utils.Audio,
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

	// Assert archived media is audio-only
	assertAudioOnlyFile(t, v.VideoPath)
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

	// Clip chat is only available when the clip has a backing VOD id.
	// ArchiveClip may force-disable chat if clip.VideoID is empty.
	assert.Equal(t, true, q.VideoProcessing)
	assert.Equal(t, true, q.RenderChat)
	assert.Equal(t, v.ClipExtVodID != "", q.ArchiveChat)
	assert.Equal(t, q.ArchiveChat, q.ChatProcessing)
	assert.NotNil(t, q.WorkflowID)
	assert.NotNil(t, q.WorkflowRunID)

	if q.ArchiveChat {
		// Async workers may already have started by the time we assert,
		// so allow task statuses to be pending/running/success.
		assert.NotEqual(t, utils.Failed, q.TaskChatDownload)
		assert.NotEqual(t, utils.Failed, q.TaskChatRender)
		assert.NotEqual(t, utils.Failed, q.TaskChatMove)
	} else {
		assert.Equal(t, utils.Success, q.TaskChatDownload)
		assert.Equal(t, utils.Success, q.TaskChatRender)
		assert.Equal(t, utils.Success, q.TaskChatMove)
	}
	assert.NotEqual(t, utils.Failed, q.TaskVideoDownload)
	assert.NotEqual(t, utils.Failed, q.TaskVideoConvert)
	assert.NotEqual(t, utils.Failed, q.TaskVideoMove)

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
	if q.ArchiveChat {
		if v.ChatPath != "" {
			assert.FileExists(t, v.ChatPath)
		}
		if v.ChatVideoPath != "" {
			assert.FileExists(t, v.ChatVideoPath)
		}
	} else {
		assert.Equal(t, "", v.ChatPath)
		assert.Equal(t, "", v.ChatVideoPath)
	}

	assert.NotEqual(t, 0, v.StorageSizeBytes)

	// Assert video is playable
	assert.True(t, tests_shared.IsPlayableVideo(v.VideoPath), "Video file is not playable")
	if v.ChatVideoPath != "" {
		assert.True(t, tests_shared.IsPlayableVideo(v.ChatVideoPath), "Video file is not playable")
	}
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
