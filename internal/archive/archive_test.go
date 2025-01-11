package archive_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/ent/queue"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/server"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/tests"
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

// TestAdmin tests the admin service. This function runs all the tests to avoid spinning up multiple containers.
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
func TestArchiveVideo(t *testing.T) {
	// Setup the application
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	// Archive the video
	err = app.ArchiveService.ArchiveVideo(context.Background(), archive.ArchiveVideoInput{
		VideoId:     TestTwitchVideoId,
		Quality:     utils.R720P60,
		ArchiveChat: true,
		RenderChat:  true,
	})
	assert.NoError(t, err)

	// Assert video was created
	v, err := app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchVideoId)).Only(context.Background())
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
	startTime := time.Now()
	for {
		if time.Since(startTime) >= TestArchiveTimeout {
			t.Errorf("Timeout reached while waiting for video to be archived")
		}

		q, err := app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
		if err != nil {
			t.Errorf("Error querying queue item: %v", err)
		}

		if !q.Processing {
			break
		}

		time.Sleep(5 * time.Second)
	}

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
}

// ArchiveVideo tests the full archive process for a video without chat downloading, processing, and rendering
func TestArchiveVideoNoChat(t *testing.T) {
	// Setup the application
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	// Archive the video
	err = app.ArchiveService.ArchiveVideo(context.Background(), archive.ArchiveVideoInput{
		VideoId:     TestTwitchVideoId,
		Quality:     utils.R720P60,
		ArchiveChat: false,
		RenderChat:  false,
	})
	assert.NoError(t, err)

	// Assert video was created
	v, err := app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchVideoId)).Only(context.Background())
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
	startTime := time.Now()
	for {
		if time.Since(startTime) >= TestArchiveTimeout {
			t.Errorf("Timeout reached while waiting for video to be archived")
		}

		q, err := app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
		if err != nil {
			t.Errorf("Error querying queue item: %v", err)
		}

		if !q.Processing {
			break
		}

		time.Sleep(5 * time.Second)
	}

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
}

// ArchiveVideo tests the full archive process for a video without chat rendering
func TestArchiveVideoNoChatRender(t *testing.T) {
	// Setup the application
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	// Archive the video
	err = app.ArchiveService.ArchiveVideo(context.Background(), archive.ArchiveVideoInput{
		VideoId:     TestTwitchVideoId,
		Quality:     utils.R720P60,
		ArchiveChat: true,
		RenderChat:  false,
	})
	assert.NoError(t, err)

	// Assert video was created
	v, err := app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchVideoId)).Only(context.Background())
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
	startTime := time.Now()
	for {
		if time.Since(startTime) >= TestArchiveTimeout {
			t.Errorf("Timeout reached while waiting for video to be archived")
		}

		q, err := app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
		if err != nil {
			t.Errorf("Error querying queue item: %v", err)
		}

		if !q.Processing {
			break
		}

		time.Sleep(5 * time.Second)
	}

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
	err = app.ArchiveService.ArchiveVideo(context.Background(), archive.ArchiveVideoInput{
		VideoId:     TestTwitchVideoId,
		Quality:     utils.R720P60,
		ArchiveChat: false,
		RenderChat:  false,
	})
	assert.NoError(t, err)

	// Assert video was created
	v, err := app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchVideoId)).Only(context.Background())
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
	startTime := time.Now()
	for {
		if time.Since(startTime) >= TestArchiveTimeout {
			t.Errorf("Timeout reached while waiting for video to be archived")
		}

		q, err := app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
		if err != nil {
			t.Errorf("Error querying queue item: %v", err)
		}

		if !q.Processing {
			break
		}

		time.Sleep(5 * time.Second)
	}

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

	assert.DirExists(t, v.VideoHlsPath)

	// Assert number of files in HLS directory is greater than 0
	files, err := os.ReadDir(v.VideoHlsPath)
	assert.NoError(t, err)
	assert.Greater(t, len(files), 0)
}

// ArchiveVideo tests the full archive process for a video with chat downloading, processing, and rendering
func TestArchiveClip(t *testing.T) {
	// Setup the application
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	// Archive the video
	err = app.ArchiveService.ArchiveClip(context.Background(), archive.ArchiveClipInput{
		ID:          TestTwitchClipId,
		Quality:     utils.R720P60,
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
	startTime := time.Now()
	for {
		if time.Since(startTime) >= TestArchiveTimeout {
			t.Errorf("Timeout reached while waiting for video to be archived")
		}

		q, err := app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
		if err != nil {
			t.Errorf("Error querying queue item: %v", err)
		}

		if !q.Processing {
			break
		}

		time.Sleep(5 * time.Second)
	}

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
}

// TestArchiveVideoWithSpriteThumbnails tests generate sprite thumbnails after a video is archived.
func TestArchiveVideoWithSpriteThumbnails(t *testing.T) {
	// Setup the application
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	// Archive the video
	err = app.ArchiveService.ArchiveVideo(context.Background(), archive.ArchiveVideoInput{
		VideoId:     TestTwitchVideoId,
		Quality:     utils.R720P60,
		ArchiveChat: false,
		RenderChat:  false,
	})
	assert.NoError(t, err)

	// Assert video was created
	v, err := app.Database.Client.Vod.Query().Where(vod.ExtID(TestTwitchVideoId)).Only(context.Background())
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
	startTime := time.Now()
	for {
		if time.Since(startTime) >= TestArchiveTimeout {
			t.Errorf("Timeout reached while waiting for video to be archived")
		}

		q, err := app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(v.ID))).Only(context.Background())
		if err != nil {
			t.Errorf("Error querying queue item: %v", err)
		}

		if !q.Processing {
			break
		}

		time.Sleep(5 * time.Second)
	}

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

	// Wait 5 seconds for sprite thumbnails
	time.Sleep(5 * time.Second)

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
