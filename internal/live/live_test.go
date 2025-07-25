package live_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/ent"
	entChannel "github.com/zibbp/ganymede/ent/channel"
	entLive "github.com/zibbp/ganymede/ent/live"
	"github.com/zibbp/ganymede/ent/queue"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/server"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/tests"
	tests_shared "github.com/zibbp/ganymede/tests/shared"
)

var (
	LiveArchiveCheckTimeout = 15 * time.Second // Maximum wait time for live archive to start
	TestArchiveTimeout      = 300 * time.Second
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
func createWatchedChannel(t *testing.T, app *server.Application, liveInput live.Live, channelID uuid.UUID) *ent.Live {
	watchedChannel, err := app.LiveService.AddLiveWatchedChannel(t.Context(), liveInput)
	assert.NoError(t, err)
	assert.NotNil(t, watchedChannel, "Expected a valid watched channel to be created")

	watchedChannel, err = app.Database.Client.Live.Query().Where(entLive.HasChannelWith(entChannel.ID(channelID))).WithChannel().Only(t.Context())
	assert.NoError(t, err, "Failed to query watched channel")
	assert.NotNil(t, watchedChannel, "Watched channel should not be nil")
	assert.Equal(t, channelID, watchedChannel.Edges.Channel.ID, "Watched channel should be linked to the archived platform channel")
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

// Helper to assert VOD and queue item, stop archive, and check files
func assertVodAndQueue(t *testing.T, app *server.Application, liveChannel platform.LiveStreamInfo) {
	vod, err := app.Database.Client.Vod.Query().Where(entVod.ExtStreamID(liveChannel.ID)).First(t.Context())
	assert.NoError(t, err, "Failed to query VOD for live stream")
	assert.NotNil(t, vod, "VOD should not be nil")

	q, err := app.Database.Client.Queue.Query().Where(queue.HasVodWith(entVod.ID(vod.ID))).Only(t.Context())
	assert.NoError(t, err, "Failed to query queue item for VOD")
	assert.NotNil(t, q, "Queue item for VOD should not be nil")

	t.Logf("Waiting for live stream to archive")
	time.Sleep(1 * time.Minute)

	assert.NoError(t, app.QueueService.StopQueueItem(t.Context(), q.ID), "Failed to stop live archive")
	tests_shared.WaitForArchiveCompletion(t, app, vod.ID, TestArchiveTimeout)

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
	watchedChannel := createWatchedChannel(t, app, liveInput, channel.ID)
	startAndWaitForArchiving(t, app, watchedChannel.ID, false)
	assertVodAndQueue(t, app, liveChannel)
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
	watchedChannel := createWatchedChannel(t, app, liveInput, channel.ID)
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
	watchedChannel := createWatchedChannel(t, app, liveInput, channel.ID)
	startAndWaitForArchiving(t, app, watchedChannel.ID, false)
	assertVodAndQueue(t, app, liveChannel)
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
	watchedChannel := createWatchedChannel(t, app, liveInput, channel.ID)
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
	watchedChannel := createWatchedChannel(t, app, liveInput, channel.ID)
	startAndWaitForArchiving(t, app, watchedChannel.ID, false)
	assertVodAndQueue(t, app, liveChannel)
}
