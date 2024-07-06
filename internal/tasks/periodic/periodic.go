package tasks_periodic

import (
	"context"
	"time"

	"github.com/riverqueue/river"
	"github.com/zibbp/ganymede/internal/errors"
	"github.com/zibbp/ganymede/internal/live"
)

func liveServiceFromContext(ctx context.Context) (*live.Service, error) {
	liveService, exists := ctx.Value("live_service").(*live.Service)
	if !exists || liveService == nil {
		return nil, errors.New("live service not found in context")
	}

	return liveService, nil
}

// Check watched channels for new videos
type CheckChannelsForNewVideosArgs struct{}

func (CheckChannelsForNewVideosArgs) Kind() string { return "check_channels_for_new_videos" }

func (w CheckChannelsForNewVideosArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
	}
}

func (w CheckChannelsForNewVideosArgs) Timeout(job *river.Job[CheckChannelsForNewVideosArgs]) time.Duration {
	return 1 * time.Minute
}

type CheckChannelsForNewVideosWorker struct {
	river.WorkerDefaults[CheckChannelsForNewVideosArgs]
}

func (w CheckChannelsForNewVideosWorker) Work(ctx context.Context, job *river.Job[CheckChannelsForNewVideosArgs]) error {

	liveService, err := liveServiceFromContext(ctx)
	if err != nil {
		return err
	}

	err = liveService.CheckVodWatchedChannels(ctx)
	if err != nil {
		return err
	}

	return nil
}
