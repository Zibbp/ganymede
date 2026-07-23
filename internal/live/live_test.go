package live_test

import (
	"context"
	"fmt"
	"os"
	osExec "os/exec"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/ent"
	entChannel "github.com/zibbp/ganymede/ent/channel"
	entLive "github.com/zibbp/ganymede/ent/live"
	"github.com/zibbp/ganymede/ent/queue"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/config"
	internalExec "github.com/zibbp/ganymede/internal/exec"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/server"
	"github.com/zibbp/ganymede/internal/tasks"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/tests"
	tests_shared "github.com/zibbp/ganymede/tests/shared"
)

var (
	LiveArchiveCheckTimeout = 15 * time.Second // Maximum wait time for live archive to start
	TestArchiveTimeout      = 120 * time.Second
)

// waitForWatchedChannelToStartArchiving waits for the watched channel to start archiving.
func waitForWatchedChannelToStartArchiving(t *testing.T, app *server.Application, watchedChannelId uuid.UUID) error {
	startTime := time.Now()
	for {
		if time.Since(startTime) >= LiveArchiveCheckTimeout {
			return fmt.Errorf("Timeout reached while waiting for watched channel to start archiving")
		}

		watchedChannel, err := app.Database.Client.Live.Query().Where(entLive.ID(watchedChannelId)).WithChannel().Only(t.Context())
		assert.NoError(t, err, "Failed to query watched channel")
		assert.NotNil(t, watchedChannel, "Watched channel should not be nil")

		if watchedChannel.IsLive && watchedChannel.WatchLive {
			t.Logf("Watched channel %s is now live and archiving", watchedChannel.Edges.Channel.Name)
			return nil
		}

		time.Sleep(5 * time.Second) // Wait before checking again
	}
}

func setupAppAndLiveChannel(t *testing.T) (*server.Application, platform.LiveStreamInfo, *ent.Channel) {
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	liveChannels, err := app.PlatformTwitch.GetStreams(t.Context(), 1)
	assert.NoError(t, err)
	assert.NotEmpty(t, liveChannels, "Expected at least one live channel")
	liveChannel := liveChannels[0]

	_, err = app.ArchiveService.ArchiveChannel(context.Background(), liveChannel.UserLogin)
	assert.NoError(t, err)

	channel, err := app.Database.Client.Channel.Query().Where(entChannel.ExtID(liveChannel.UserID)).Only(t.Context())
	assert.NoError(t, err, "Failed to query channel")
	assert.NotNil(t, channel, "Channel should not be nil")

	return app, liveChannel, channel
}

// Helper to create a watched channel
func createWatchedChannel(t *testing.T, app *server.Application, liveInput live.Live, channelID uuid.UUID, storageTemplateFolder *string, storageTemplateFile *string) *ent.Live {
	watchedChannel, err := app.LiveService.AddLiveWatchedChannel(t.Context(), liveInput)
	assert.NoError(t, err)
	assert.NotNil(t, watchedChannel, "Expected a valid watched channel to be created")

	watchedChannel, err = app.Database.Client.Live.Query().Where(entLive.HasChannelWith(entChannel.ID(channelID))).WithChannel().Only(t.Context())
	assert.NoError(t, err, "Failed to query watched channel")
	assert.NotNil(t, watchedChannel, "Watched channel should not be nil")
	assert.Equal(t, channelID, watchedChannel.Edges.Channel.ID, "Watched channel should be linked to the archived platform channel")

	// Update storage template settings if provided
	if storageTemplateFolder != nil || storageTemplateFile != nil {
		updatedConfig := config.Get()
		if storageTemplateFolder != nil {
			updatedConfig.StorageTemplates.FolderTemplate = *storageTemplateFolder
		}
		if storageTemplateFile != nil {
			updatedConfig.StorageTemplates.FileTemplate = *storageTemplateFile
		}
		assert.NoError(t, config.UpdateConfig(updatedConfig), "Failed to update config with storage template settings")
	}

	return watchedChannel
}

// Helper to start live check and wait for archiving
func startAndWaitForArchiving(t *testing.T, app *server.Application, watchedChannelID uuid.UUID, expectError bool) {
	assert.NoError(t, app.TaskService.StartTask(t.Context(), "check_live"), "Failed to start task to check live watched channels")
	if expectError {
		assert.Error(t, waitForWatchedChannelToStartArchiving(t, app, watchedChannelID), "Expected error waiting for watched channel to start archiving")
	} else {
		assert.NoError(t, waitForWatchedChannelToStartArchiving(t, app, watchedChannelID), "Failed to wait for watched channel to start archiving")
	}
}

func waitForQueueTaskStatus(t *testing.T, app *server.Application, queueID uuid.UUID, status utils.TaskStatus, timeout time.Duration) (*ent.Queue, error) {
	start := time.Now()
	for {
		if time.Since(start) >= timeout {
			return nil, fmt.Errorf("timeout reached waiting for queue task_video_download status %s", status)
		}

		q, err := app.Database.Client.Queue.Get(t.Context(), queueID)
		if err != nil {
			return nil, err
		}

		if q.TaskVideoDownload == status {
			return q, nil
		}

		time.Sleep(2 * time.Second)
	}
}

// Helper to assert VOD and queue item, stop archive, and check files
func assertVodAndQueue(t *testing.T, app *server.Application, liveChannel platform.LiveStreamInfo, stopArchive bool) {
	vod, err := app.Database.Client.Vod.Query().Where(entVod.ExtStreamID(liveChannel.ID)).WithChannel().WithChapters().First(t.Context())
	assert.NoError(t, err, "Failed to query VOD for live stream")
	assert.NotNil(t, vod, "VOD should not be nil")

	q, err := app.Database.Client.Queue.Query().Where(queue.HasVodWith(entVod.ID(vod.ID))).Only(t.Context())
	assert.NoError(t, err, "Failed to query queue item for VOD")
	assert.NotNil(t, q, "Queue item for VOD should not be nil")

	// Assert storage template settings were applied
	expectedFolderName, err := archive.GetFolderName(vod.ID, archive.StorageTemplateInput{
		UUID:    vod.ID,
		ID:      vod.ExtID,
		Channel: vod.Edges.Channel.Name,
		Title:   vod.Title,
		Type:    string(vod.Type),
		Date:    vod.StreamedAt.Format("2006-01-02"),
		YYYY:    vod.StreamedAt.Format("2006"),
		MM:      vod.StreamedAt.Format("01"),
		DD:      vod.StreamedAt.Format("02"),
		HH:      vod.StreamedAt.Format("15"),
	})
	assert.NoError(t, err)
	expectedFileName, err := archive.GetFileName(vod.ID, archive.StorageTemplateInput{
		UUID:    vod.ID,
		ID:      vod.ExtID,
		Channel: vod.Edges.Channel.Name,
		Title:   vod.Title,
		Type:    string(vod.Type),
		Date:    vod.StreamedAt.Format("2006-01-02"),
		YYYY:    vod.StreamedAt.Format("2006"),
		MM:      vod.StreamedAt.Format("01"),
		DD:      vod.StreamedAt.Format("02"),
		HH:      vod.StreamedAt.Format("15"),
	})
	assert.NoError(t, err)
	assert.Equal(t, vod.FolderName, expectedFolderName, "Folder name should match the expected storage template")
	assert.Equal(t, vod.FileName, expectedFileName, "File name should match the expected storage template")

	t.Logf("Waiting for live stream to archive")
	time.Sleep(60 * time.Second)

	// If watch while archiving is enabled, check that the hls playlist exists
	if config.Get().Livestream.WatchWhileArchiving {
		hlsPlaylistPath := fmt.Sprintf("%s/%s-video.m3u8", vod.TmpVideoHlsPath, vod.ExtID)
		assert.FileExists(t, hlsPlaylistPath, "HLS playlist file should exist for watch while archiving")
	}

	if stopArchive {
		assert.NoError(t, app.QueueService.StopQueueItem(t.Context(), q.ID), "Failed to stop live archive")
		tests_shared.WaitForArchiveCompletion(t, app, vod.ID, TestArchiveTimeout)
	}

	q, err = app.Database.Client.Queue.Get(t.Context(), q.ID)
	assert.NoError(t, err)
	assert.NotNil(t, q)
	assert.Equal(t, true, q.LiveArchive)
	assert.Equal(t, false, q.ChatProcessing)
	assert.Equal(t, false, q.VideoProcessing)
	assert.Equal(t, utils.Success, q.TaskChatDownload)
	assert.Equal(t, utils.Success, q.TaskChatRender)
	assert.Equal(t, utils.Success, q.TaskChatMove)
	assert.Equal(t, utils.Success, q.TaskVideoDownload)
	assert.Equal(t, utils.Success, q.TaskVideoConvert)
	assert.Equal(t, utils.Success, q.TaskVideoMove)

	assert.FileExists(t, vod.ThumbnailPath)
	assert.FileExists(t, vod.WebThumbnailPath)
	assert.FileExists(t, vod.VideoPath)
	assert.FileExists(t, vod.ChatPath)
	assert.FileExists(t, vod.ChatVideoPath)
	assert.NotEqual(t, 0, vod.StorageSizeBytes)
	assert.True(t, tests_shared.IsPlayableVideo(vod.VideoPath), "Video file is not playable")
	assert.True(t, tests_shared.IsPlayableVideo(vod.ChatVideoPath), "Video file is not playable")

	// Assert at least one chapter exists
	assert.NotEmpty(t, vod.Edges.Chapters, "Expected at least one chapter to be present")

	// Assert video is playable
	assert.True(t, tests_shared.IsPlayableVideo(vod.VideoPath), "Video file is not playable")
	assert.True(t, tests_shared.IsPlayableVideo(vod.ChatVideoPath), "Video file is not playable")

	// Assert chat files is greater than 0 bytes
	chatFileInfo, err := os.Stat(vod.ChatPath)
	assert.NoError(t, err)
	assert.Greater(t, chatFileInfo.Size(), int64(0), "Chat file should not be empty")

	// Assert info file is greater than 0 bytes
	infoFileInfo, err := os.Stat(vod.InfoPath)
	assert.NoError(t, err)
	assert.Greater(t, infoFileInfo.Size(), int64(0), "Info file should not be empty")

	// Assert thumbnail is greater than 0 bytes
	thumbnailFileInfo, err := os.Stat(vod.ThumbnailPath)
	assert.NoError(t, err)
	assert.Greater(t, thumbnailFileInfo.Size(), int64(0), "Thumbnail file should not be empty")

	// Assert web thumbnail is greater than 0 bytes
	webThumbnailFileInfo, err := os.Stat(vod.WebThumbnailPath)
	assert.NoError(t, err)
	assert.Greater(t, webThumbnailFileInfo.Size(), int64(0), "Web thumbnail file should not be empty")

	// Assert sprite thumbnail facts
	vod, err = app.Database.Client.Vod.Query().Where(entVod.ExtStreamID(liveChannel.ID)).WithChannel().WithChapters().First(t.Context())
	assert.NoError(t, err, "Failed to query VOD for live stream")
	assert.NoError(t, err)
	assert.NotNil(t, vod)
	assert.Greater(t, len(vod.SpriteThumbnailsImages), 0, "Sprite thumbnails should be generated for videos")

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

func assertAudioOnlyVodAndQueue(t *testing.T, app *server.Application, liveChannel platform.LiveStreamInfo) {
	vod, err := app.Database.Client.Vod.Query().Where(entVod.ExtStreamID(liveChannel.ID)).WithChannel().WithChapters().First(t.Context())
	assert.NoError(t, err, "Failed to query VOD for live stream")
	assert.NotNil(t, vod, "VOD should not be nil")

	q, err := app.Database.Client.Queue.Query().Where(queue.HasVodWith(entVod.ID(vod.ID))).Only(t.Context())
	assert.NoError(t, err, "Failed to query queue item for VOD")
	assert.NotNil(t, q, "Queue item for VOD should not be nil")

	t.Logf("Waiting for audio-only live stream to archive")
	time.Sleep(60 * time.Second)

	assert.NoError(t, app.QueueService.StopQueueItem(t.Context(), q.ID), "Failed to stop live archive")
	tests_shared.WaitForArchiveCompletion(t, app, vod.ID, TestArchiveTimeout)

	vod, err = app.Database.Client.Vod.Query().Where(entVod.ExtStreamID(liveChannel.ID)).WithChannel().WithChapters().First(t.Context())
	assert.NoError(t, err, "Failed to query VOD for live stream")
	assert.NotNil(t, vod, "VOD should not be nil")

	q, err = app.Database.Client.Queue.Get(t.Context(), q.ID)
	assert.NoError(t, err)
	assert.NotNil(t, q)
	assert.Equal(t, true, q.LiveArchive)
	assert.Equal(t, false, q.ArchiveChat)
	assert.Equal(t, false, q.RenderChat)
	assert.Equal(t, false, q.ChatProcessing)
	assert.Equal(t, false, q.VideoProcessing)
	assert.Equal(t, utils.Success, q.TaskChatDownload)
	assert.Equal(t, utils.Success, q.TaskChatConvert)
	assert.Equal(t, utils.Success, q.TaskChatRender)
	assert.Equal(t, utils.Success, q.TaskChatMove)
	assert.Equal(t, utils.Success, q.TaskVideoDownload)
	assert.Equal(t, utils.Success, q.TaskVideoConvert)
	assert.Equal(t, utils.Success, q.TaskVideoMove)

	assert.FileExists(t, vod.ThumbnailPath)
	assert.FileExists(t, vod.WebThumbnailPath)
	assert.FileExists(t, vod.VideoPath)
	assert.Empty(t, vod.ChatPath)
	assert.Empty(t, vod.ChatVideoPath)
	assert.NotEqual(t, 0, vod.StorageSizeBytes)
	assertAudioOnlyFile(t, vod.VideoPath)

	assert.NotEmpty(t, vod.Edges.Chapters, "Expected at least one chapter to be present")

	infoFileInfo, err := os.Stat(vod.InfoPath)
	assert.NoError(t, err)
	assert.Greater(t, infoFileInfo.Size(), int64(0), "Info file should not be empty")

	thumbnailFileInfo, err := os.Stat(vod.ThumbnailPath)
	assert.NoError(t, err)
	assert.Greater(t, thumbnailFileInfo.Size(), int64(0), "Thumbnail file should not be empty")

	webThumbnailFileInfo, err := os.Stat(vod.WebThumbnailPath)
	assert.NoError(t, err)
	assert.Greater(t, webThumbnailFileInfo.Size(), int64(0), "Web thumbnail file should not be empty")
}

// Helper to assert no VOD and queue item exist
func assertNoVodAndQueue(t *testing.T, app *server.Application, liveChannel platform.LiveStreamInfo) {
	vod, err := app.Database.Client.Vod.Query().Where(entVod.ExtStreamID(liveChannel.ID)).Only(t.Context())
	assert.Error(t, err, "Expected error querying VOD for live stream")
	assert.Nil(t, vod, "VOD should be nil")

	q, err := app.Database.Client.Queue.Query().Where(queue.HasVodWith(entVod.ExtStreamID(liveChannel.ID))).Only(t.Context())
	assert.Error(t, err, "Expected error querying queue item for VOD")
	assert.Nil(t, q, "Queue item for VOD should be nil")
}

// TestTwitchWatchedChannelLive tests the basic live archiving of a Twitch channel
func TestTwitchWatchedChannelLive(t *testing.T) {
	app, liveChannel, channel := setupAppAndLiveChannel(t)
	liveInput := live.Live{
		ID:                    channel.ID,
		WatchLive:             true,
		WatchVod:              false,
		DownloadArchives:      false,
		DownloadHighlights:    false,
		DownloadUploads:       false,
		ArchiveChat:           true,
		Resolution:            "best",
		RenderChat:            true,
		DownloadSubOnly:       false,
		UpdateMetadataMinutes: 1,
	}
	watchedChannel := createWatchedChannel(t, app, liveInput, channel.ID, nil, nil)
	startAndWaitForArchiving(t, app, watchedChannel.ID, false)
	assertVodAndQueue(t, app, liveChannel, true)
}

// TestTwitchWatchedChannelLiveAudioOnlyNoChat tests live archiving audio only without chat.
func TestTwitchWatchedChannelLiveAudioOnlyNoChat(t *testing.T) {
	app, liveChannel, channel := setupAppAndLiveChannel(t)
	liveInput := live.Live{
		ID:                    channel.ID,
		WatchLive:             true,
		WatchVod:              false,
		DownloadArchives:      false,
		DownloadHighlights:    false,
		DownloadUploads:       false,
		ArchiveChat:           false,
		Resolution:            "audio",
		RenderChat:            false,
		DownloadSubOnly:       false,
		UpdateMetadataMinutes: 1,
	}
	watchedChannel := createWatchedChannel(t, app, liveInput, channel.ID, nil, nil)
	startAndWaitForArchiving(t, app, watchedChannel.ID, false)
	assertAudioOnlyVodAndQueue(t, app, liveChannel)
}

// TestTwitchWatchedChannelLiveFFmpegKilledStillFinalizes verifies that if ffmpeg dies during live recording,
// the workflow still finalizes: live chat is cancelled and post-process/move tasks are allowed to continue.
func TestTwitchWatchedChannelLiveFFmpegKilledStillFinalizes(t *testing.T) {
	app, liveChannel, channel := setupAppAndLiveChannel(t)
	liveInput := live.Live{
		ID:                    channel.ID,
		WatchLive:             true,
		WatchVod:              false,
		DownloadArchives:      false,
		DownloadHighlights:    false,
		DownloadUploads:       false,
		ArchiveChat:           true,
		Resolution:            "best",
		RenderChat:            true,
		DownloadSubOnly:       false,
		UpdateMetadataMinutes: 1,
	}
	watchedChannel := createWatchedChannel(t, app, liveInput, channel.ID, nil, nil)
	startAndWaitForArchiving(t, app, watchedChannel.ID, false)

	vod, err := app.Database.Client.Vod.Query().Where(entVod.ExtStreamID(liveChannel.ID)).First(t.Context())
	assert.NoError(t, err, "Failed to query VOD for live stream")
	assert.NotNil(t, vod, "VOD should not be nil")

	q, err := app.Database.Client.Queue.Query().Where(queue.HasVodWith(entVod.ID(vod.ID))).Only(t.Context())
	assert.NoError(t, err, "Failed to query queue item for VOD")
	assert.NotNil(t, q, "Queue item should not be nil")

	_, err = waitForQueueTaskStatus(t, app, q.ID, utils.Running, 60*time.Second)
	assert.NoError(t, err, "Expected live video download task to reach running state")

	// Kill ffmpeg process by matching VOD UUID present in ffmpeg command args
	killMatched := false
	for i := 0; i < 5; i++ {
		cmd := osExec.Command("pkill", "-f", vod.ID.String())
		if err := cmd.Run(); err == nil {
			killMatched = true
			break
		}
		time.Sleep(2 * time.Second)
	}
	assert.True(t, killMatched, "Expected to kill at least one ffmpeg process for the live VOD")

	q, err = waitForQueueTaskStatus(t, app, q.ID, utils.Success, 90*time.Second)
	assert.NoError(t, err, "Expected queue task_video_download to be finalized after ffmpeg termination")
	assert.Equal(t, utils.Success, q.TaskVideoDownload)

	// Downstream tasks should be able to progress after ffmpeg is killed.
	deadline := time.Now().Add(90 * time.Second)
	for {
		q, err = app.Database.Client.Queue.Get(t.Context(), q.ID)
		assert.NoError(t, err)
		if q.TaskVideoConvert != utils.Pending || q.TaskVideoMove != utils.Pending {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected post-process or move task to start after ffmpeg termination")
		}
		time.Sleep(2 * time.Second)
	}
}

// TestTwitchLiveArchiveRecoversAfterWorkerCrash verifies the complete
// watchdog path after SIGKILL: the live capture child exits with its worker,
// the partial transport stream is finalized, and the resulting archive is
// playable.
func TestTwitchLiveArchiveRecoversAfterWorkerCrash(t *testing.T) {
	app, err := tests.SetupWithoutWorker(t)
	require.NoError(t, err)

	liveChannels, err := app.PlatformTwitch.GetStreams(t.Context(), 1)
	require.NoError(t, err)
	require.NotEmpty(t, liveChannels, "expected at least one live channel")
	liveChannel := liveChannels[0]

	_, err = app.ArchiveService.ArchiveChannel(t.Context(), liveChannel.UserLogin)
	require.NoError(t, err)
	channel, err := app.Database.Client.Channel.Query().
		Where(entChannel.ExtID(liveChannel.UserID)).
		Only(t.Context())
	require.NoError(t, err)

	workerProcess := tests.StartCrashableWorker(t, "TestWorkerCrashHelper")
	watchedChannel := createWatchedChannel(t, app, live.Live{
		ID:                    channel.ID,
		WatchLive:             true,
		WatchVod:              false,
		DownloadArchives:      false,
		DownloadHighlights:    false,
		DownloadUploads:       false,
		ArchiveChat:           false,
		Resolution:            utils.R160.String(),
		RenderChat:            false,
		DownloadSubOnly:       false,
		UpdateMetadataMinutes: 1,
	}, channel.ID, nil, nil)
	startAndWaitForArchiving(t, app, watchedChannel.ID, false)

	v, err := app.Database.Client.Vod.Query().
		Where(entVod.ExtStreamID(liveChannel.ID)).
		Only(t.Context())
	require.NoError(t, err)
	q, err := app.Database.Client.Queue.Query().
		Where(queue.HasVodWith(entVod.ID(v.ID))).
		Only(t.Context())
	require.NoError(t, err)

	capturedBytes := tests_shared.WaitForRunningVideoDownload(
		t,
		app,
		q.ID,
		v.TmpVideoDownloadPath,
		256*1024,
		90*time.Second,
	)
	originalJob := tests_shared.FindArchiveJob(
		t,
		app,
		q.ID,
		string(utils.TaskDownloadLiveVideo),
		rivertype.JobStateRunning,
	)
	require.NotNil(t, originalJob, "expected the original live download River job")

	require.NoError(t, workerProcess.Crash())
	tests_shared.WaitForProcessExit(t, v.ID.String(), 5*time.Second)
	info, err := os.Stat(v.TmpVideoDownloadPath)
	require.NoError(t, err)
	require.GreaterOrEqual(t, info.Size(), capturedBytes)

	_ = tests.StartCrashableWorker(t, "TestWorkerCrashHelper")

	// Advance the isolated job heartbeat so the explicitly queued watchdog
	// observes the worker claim as stale immediately.
	_, err = app.RiverClient.Client.JobUpdate(t.Context(), originalJob.ID, &river.JobUpdateParams{
		Output: tasks.ArchiveProgressOutput{HeartbeatAt: time.Now().Add(-5 * time.Minute)},
	})
	require.NoError(t, err)
	_, err = app.RiverClient.Insert(t.Context(), tasks.WatchdogArgs{}, nil)
	require.NoError(t, err)
	tests_shared.WaitForArchiveJobCancellation(t, app, originalJob.ID, 30*time.Second)

	// The production grace and media-quiet windows are time based. Backdating
	// only this test container's cancellation marker and partial file models
	// their expiry without adding several minutes to every integration run.
	result, err := app.Database.SQLDB.ExecContext(t.Context(), `
		UPDATE river_job
		SET metadata = jsonb_set(
			metadata,
			'{cancel_attempted_at}',
			to_jsonb($1::text),
			true
		)
		WHERE id = $2
	`, time.Now().Add(-5*time.Minute).Format(time.RFC3339Nano), originalJob.ID)
	require.NoError(t, err)
	updated, err := result.RowsAffected()
	require.NoError(t, err)
	require.EqualValues(t, 1, updated)

	quietTime := time.Now().Add(-time.Minute)
	require.NoError(t, os.Chtimes(v.TmpVideoDownloadPath, quietTime, quietTime))
	_, err = app.RiverClient.Insert(t.Context(), tasks.WatchdogArgs{}, nil)
	require.NoError(t, err)

	q = tests_shared.WaitForArchiveCompletionAfterCrash(t, app, v.ID, TestArchiveTimeout)
	v = tests_shared.WaitForArchiveMetadataFinalization(t, app, v.ID, 30*time.Second)

	require.True(t, q.LiveArchive)
	require.False(t, q.Processing)
	require.False(t, q.VideoProcessing)
	require.False(t, q.ChatProcessing)
	require.Equal(t, utils.Success, q.TaskVideoDownload)
	require.Equal(t, utils.Success, q.TaskVideoConvert)
	require.Equal(t, utils.Success, q.TaskVideoMove)
	require.Equal(t, utils.Success, q.TaskChatDownload)
	require.Equal(t, utils.Success, q.TaskChatConvert)
	require.Equal(t, utils.Success, q.TaskChatRender)
	require.Equal(t, utils.Success, q.TaskChatMove)
	require.FileExists(t, v.VideoPath)
	require.True(t, tests_shared.IsPlayableVideo(v.VideoPath), "recovered live archive is not playable")
	require.NotZero(t, v.StorageSizeBytes)
	require.NotEmpty(t, v.Edges.Chapters)
}

// TestWorkerCrashHelper is executed in a child copy of this test binary by
// StartCrashableWorker. During an ordinary package test it returns immediately.
func TestWorkerCrashHelper(t *testing.T) {
	tests.RunWorkerCrashHelper(t)
}

// TestTwitchWatchedChannelLiveWithWatchLive tests the basic live archiving of a Twitch channel with the watch live feature
func TestTwitchWatchedChannelLiveWithWatchLive(t *testing.T) {
	app, liveChannel, channel := setupAppAndLiveChannel(t)

	updatedConfig := config.Get()
	updatedConfig.Livestream.WatchWhileArchiving = true
	assert.NoError(t, config.UpdateConfig(updatedConfig))

	liveInput := live.Live{
		ID:                    channel.ID,
		WatchLive:             true,
		WatchVod:              false,
		DownloadArchives:      false,
		DownloadHighlights:    false,
		DownloadUploads:       false,
		ArchiveChat:           true,
		Resolution:            "best",
		RenderChat:            true,
		DownloadSubOnly:       false,
		UpdateMetadataMinutes: 1,
	}
	watchedChannel := createWatchedChannel(t, app, liveInput, channel.ID, nil, nil)
	startAndWaitForArchiving(t, app, watchedChannel.ID, false)
	assertVodAndQueue(t, app, liveChannel, true)
}

// TestTwitchWatchedChannelLiveCategoryRestrictionFail tests live archiving with category restrictions that prevent archiving
func TestTwitchWatchedChannelLiveCategoryRestrictionFail(t *testing.T) {
	app, _, channel := setupAppAndLiveChannel(t)
	liveInput := live.Live{
		ID:                    channel.ID,
		WatchLive:             true,
		WatchVod:              false,
		DownloadArchives:      false,
		DownloadHighlights:    false,
		DownloadUploads:       false,
		ArchiveChat:           true,
		Resolution:            "best",
		RenderChat:            true,
		DownloadSubOnly:       false,
		UpdateMetadataMinutes: 1,
		Categories:            []string{"Factorio"},
		ApplyCategoriesToLive: true,
	}
	watchedChannel := createWatchedChannel(t, app, liveInput, channel.ID, nil, nil)
	startAndWaitForArchiving(t, app, watchedChannel.ID, true)
}

// TestTwitchWatchedChannelLiveCategoryRestriction tests live archiving with matching category restrictions
func TestTwitchWatchedChannelLiveCategoryRestriction(t *testing.T) {
	app, liveChannel, channel := setupAppAndLiveChannel(t)
	liveInput := live.Live{
		ID:                    channel.ID,
		WatchLive:             true,
		WatchVod:              false,
		DownloadArchives:      false,
		DownloadHighlights:    false,
		DownloadUploads:       false,
		ArchiveChat:           true,
		Resolution:            "best",
		RenderChat:            true,
		DownloadSubOnly:       false,
		UpdateMetadataMinutes: 1,
		Categories:            []string{liveChannel.GameName},
	}
	watchedChannel := createWatchedChannel(t, app, liveInput, channel.ID, nil, nil)
	startAndWaitForArchiving(t, app, watchedChannel.ID, false)
	assertVodAndQueue(t, app, liveChannel, true)
}

// TestTwitchWatchedChannelLiveCategoryRestrictionStrict tests live archiving with matching category restrictions with strict category restriction enabled. The category is changed after the live stream archive starts causing it to stop archiving once the category no longer matches.
func TestTwitchWatchedChannelLiveCategoryRestrictionStrict(t *testing.T) {
	app, liveChannel, channel := setupAppAndLiveChannel(t)
	liveInput := live.Live{
		ID:                    channel.ID,
		WatchLive:             true,
		WatchVod:              false,
		DownloadArchives:      false,
		DownloadHighlights:    false,
		DownloadUploads:       false,
		ArchiveChat:           true,
		Resolution:            "best",
		RenderChat:            true,
		DownloadSubOnly:       false,
		UpdateMetadataMinutes: 1,
		ApplyCategoriesToLive: true,
		StrictCategoriesLive:  true,
		Categories:            []string{liveChannel.GameName},
	}
	watchedChannel := createWatchedChannel(t, app, liveInput, channel.ID, nil, nil)

	startAndWaitForArchiving(t, app, watchedChannel.ID, false)

	// Change category to "TestCategory" causing the live archive to stop
	liveInput.Categories = []string{"TestCategory"}
	liveInput.ID = watchedChannel.ID
	_, err := app.LiveService.UpdateLiveWatchedChannel(context.Background(), liveInput)
	assert.NoError(t, err, "Failed to update watched channel with new category")

	time.Sleep(15 * time.Second) // Let stream archive for a bit

	// Run check live again
	assert.NoError(t, app.TaskService.StartTask(t.Context(), "check_live"), "Failed to run check_live task")

	assertVodAndQueue(t, app, liveChannel, false)
}

// TestTwitchWatchedChannelBlacklistCategoryRestriction tests live archiving with a blacklisted category that prevents archiving
func TestTwitchWatchedChannelBlacklistCategoryRestriction(t *testing.T) {
	app, liveChannel, channel := setupAppAndLiveChannel(t)
	liveInput := live.Live{
		ID:                    channel.ID,
		WatchLive:             true,
		WatchVod:              false,
		DownloadArchives:      false,
		DownloadHighlights:    false,
		DownloadUploads:       false,
		ArchiveChat:           true,
		Resolution:            "720p",
		RenderChat:            true,
		DownloadSubOnly:       false,
		ApplyCategoriesToLive: true,
		BlacklistCategories:   true,
		Categories:            []string{liveChannel.GameName},
	}
	watchedChannel := createWatchedChannel(t, app, liveInput, channel.ID, nil, nil)
	startAndWaitForArchiving(t, app, watchedChannel.ID, true)
	assertNoVodAndQueue(t, app, liveChannel)
}

// TestTwitchWatchedChannelBlacklistCategoryRestrictionNoCategorySelected tests live archiving with a blacklisted category but no category selected, allowing archiving to proceed
func TestTwitchWatchedChannelBlacklistCategoryRestrictionNoCategorySelected(t *testing.T) {
	app, liveChannel, channel := setupAppAndLiveChannel(t)
	liveInput := live.Live{
		ID:                    channel.ID,
		WatchLive:             true,
		WatchVod:              false,
		DownloadArchives:      false,
		DownloadHighlights:    false,
		DownloadUploads:       false,
		ArchiveChat:           true,
		Resolution:            "720p",
		RenderChat:            true,
		DownloadSubOnly:       false,
		ApplyCategoriesToLive: true,
		BlacklistCategories:   true,
	}
	watchedChannel := createWatchedChannel(t, app, liveInput, channel.ID, nil, nil)
	startAndWaitForArchiving(t, app, watchedChannel.ID, false)
	assertVodAndQueue(t, app, liveChannel, true)
}

// TestTwitchWatchedChannelTitleRegexFail tests live archiving with a title regex that does not match
func TestTwitchWatchedChannelTitleRegexFail(t *testing.T) {
	app, _, channel := setupAppAndLiveChannel(t)
	liveInput := live.Live{
		ID:                    channel.ID,
		WatchLive:             true,
		WatchVod:              false,
		DownloadArchives:      false,
		DownloadHighlights:    false,
		DownloadUploads:       false,
		ArchiveChat:           true,
		Resolution:            "best",
		RenderChat:            true,
		DownloadSubOnly:       false,
		UpdateMetadataMinutes: 1,
		TitleRegex: []ent.LiveTitleRegex{
			{
				Negative:      false,
				Regex:         "(?i:GanymedeDevelopment)",
				ApplyToVideos: false,
			},
		},
	}
	watchedChannel := createWatchedChannel(t, app, liveInput, channel.ID, nil, nil)
	startAndWaitForArchiving(t, app, watchedChannel.ID, true)
}

// TestTwitchWatchedChannelTitleRegex tests live archiving with a title regex that matches
func TestTwitchWatchedChannelTitleRegex(t *testing.T) {
	app, liveChannel, channel := setupAppAndLiveChannel(t)
	liveInput := live.Live{
		ID:                    channel.ID,
		WatchLive:             true,
		WatchVod:              false,
		DownloadArchives:      false,
		DownloadHighlights:    false,
		DownloadUploads:       false,
		ArchiveChat:           true,
		Resolution:            "best",
		RenderChat:            true,
		DownloadSubOnly:       false,
		UpdateMetadataMinutes: 1,
		TitleRegex: []ent.LiveTitleRegex{
			{
				Negative:      false,
				Regex:         "(.*)",
				ApplyToVideos: false,
			},
		},
	}

	// Set custom storage template settings for this test
	customFolder := "{{YYYY}}{{MM}}-{{DD}}{{HH}} - {{title}}"
	customFile := "{{title}}_{{id}}_{{uuid}}"

	watchedChannel := createWatchedChannel(t, app, liveInput, channel.ID, &customFolder, &customFile)
	startAndWaitForArchiving(t, app, watchedChannel.ID, false)
	assertVodAndQueue(t, app, liveChannel, true)
}
