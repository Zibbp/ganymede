package registry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/internal/tasks"
	tasks_periodic "github.com/zibbp/ganymede/internal/tasks/periodic"
)

func TestNewRegistersEveryWorker(t *testing.T) {
	t.Parallel()
	workers, err := New()
	require.NoError(t, err)
	require.NotNil(t, workers)
}

func TestEveryWorkerHasItsConfiguredTimeout(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"watchdog", (&tasks.WatchdogWorker{}).Timeout(nil), time.Minute},
		{"create directory", (&tasks.CreateDirectoryWorker{}).Timeout(nil), time.Minute},
		{"save video info", (&tasks.SaveVideoInfoWorker{}).Timeout(nil), time.Minute},
		{"download thumbnail", (&tasks.DownloadTumbnailsWorker{}).Timeout(nil), time.Minute},
		{"download video", (&tasks.DownloadVideoWorker{}).Timeout(nil), 49 * time.Hour},
		{"post-process video", (&tasks.PostProcessVideoWorker{}).Timeout(nil), 24 * time.Hour},
		{"move video", (&tasks.MoveVideoWorker{}).Timeout(nil), 24 * time.Hour},
		{"download chat", (&tasks.DownloadChatWorker{}).Timeout(nil), 49 * time.Hour},
		{"render chat", (&tasks.RenderChatWorker{}).Timeout(nil), 49 * time.Hour},
		{"move chat", (&tasks.MoveChatWorker{}).Timeout(nil), 49 * time.Hour},
		{"download live video", (&tasks.DownloadLiveVideoWorker{}).Timeout(nil), 49 * time.Hour},
		{"download live chat", (&tasks.DownloadLiveChatWorker{}).Timeout(nil), 49 * time.Hour},
		{"convert live chat", (&tasks.ConvertLiveChatWorker{}).Timeout(nil), 49 * time.Hour},
		{"check new videos", (&tasks_periodic.CheckChannelsForNewVideosWorker{}).Timeout(nil), 10 * time.Minute},
		{"prune videos", (&tasks_periodic.PruneVideosWorker{}).Timeout(nil), time.Minute},
		{"import categories", (&tasks_periodic.ImportCategoriesWorker{}).Timeout(nil), time.Minute},
		{"authenticate platform", (&tasks_periodic.AuthenticatePlatformWorker{}).Timeout(nil), time.Minute},
		{"fetch JWKS", (&tasks_periodic.FetchJWKSWorker{}).Timeout(nil), time.Minute},
		{"save chapters", (&tasks_periodic.SaveVideoChaptersWorker{}).Timeout(nil), 10 * time.Minute},
		{"update stream video ID", (&tasks.UpdateStreamVideoIdWorker{}).Timeout(nil), 10 * time.Minute},
		{"static thumbnail", (&tasks.GenerateStaticThubmnailWorker{}).Timeout(nil), time.Minute},
		{"sprite thumbnail", (&tasks.GenerateSpriteThumbnailWorker{}).Timeout(nil), time.Hour},
		{"update live metadata", (&tasks.UpdateLiveStreamMetadataWorker{}).Timeout(nil), time.Minute},
		{"check new clips", (&tasks_periodic.TaskCheckChannelForNewClipsWorker{}).Timeout(nil), 10 * time.Minute},
		{"check livestreams", (&tasks_periodic.CheckChannelsForLivestreamsWorker{}).Timeout(nil), 10 * time.Minute},
		{"video storage", (&tasks.UpdateVideoStorageUsageWorker{}).Timeout(nil), 5 * time.Minute},
		{"channel storage", (&tasks.UpdateChannelStorageUsageWorker{}).Timeout(nil), 5 * time.Minute},
		{"playlist rules", (&tasks_periodic.ProcessPlaylistVideoRulesWorker{}).Timeout(nil), 5 * time.Minute},
		{"update channels", (&tasks_periodic.UpdateTwitchChannelsWorker{}).Timeout(nil), time.Minute},
		{"prune logs", (&tasks_periodic.PruneLogFilesWorker{}).Timeout(nil), 10 * time.Minute},
	}

	require.Len(t, tests, 30)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, test.got)
			require.NotZero(t, test.got)
		})
	}
}
