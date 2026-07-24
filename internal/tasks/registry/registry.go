// Package registry owns the complete River worker registry used by the
// execution client.
package registry

import (
	"fmt"

	"github.com/riverqueue/river"
	"github.com/zibbp/ganymede/internal/tasks"
	tasks_periodic "github.com/zibbp/ganymede/internal/tasks/periodic"
)

// New returns all job workers supported by Ganymede.
func New() (*river.Workers, error) {
	workers := river.NewWorkers()
	registrations := []func() error{
		func() error { return river.AddWorkerSafely(workers, &tasks.WatchdogWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks.CreateDirectoryWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks.SaveVideoInfoWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks.DownloadTumbnailsWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks.DownloadVideoWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks.PostProcessVideoWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks.MoveVideoWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks.DownloadChatWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks.RenderChatWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks.MoveChatWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks.DownloadLiveVideoWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks.DownloadLiveChatWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks.ConvertLiveChatWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks_periodic.CheckChannelsForNewVideosWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks_periodic.PruneVideosWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks_periodic.ImportCategoriesWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks_periodic.AuthenticatePlatformWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks_periodic.FetchJWKSWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks_periodic.SaveVideoChaptersWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks.UpdateStreamVideoIdWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks.GenerateStaticThubmnailWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks.GenerateSpriteThumbnailWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks.UpdateLiveStreamMetadataWorker{}) },
		func() error {
			return river.AddWorkerSafely(workers, &tasks_periodic.TaskCheckChannelForNewClipsWorker{})
		},
		func() error {
			return river.AddWorkerSafely(workers, &tasks_periodic.CheckChannelsForLivestreamsWorker{})
		},
		func() error { return river.AddWorkerSafely(workers, &tasks.UpdateVideoStorageUsageWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks.UpdateChannelStorageUsageWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks_periodic.ProcessPlaylistVideoRulesWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks_periodic.UpdateTwitchChannelsWorker{}) },
		func() error { return river.AddWorkerSafely(workers, &tasks_periodic.PruneLogFilesWorker{}) },
	}

	for _, register := range registrations {
		if err := register(); err != nil {
			return nil, fmt.Errorf("register River worker: %w", err)
		}
	}
	return workers, nil
}
