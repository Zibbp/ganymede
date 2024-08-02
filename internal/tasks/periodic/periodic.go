package tasks_periodic

import (
	"context"
	"fmt"
	"time"

	"github.com/riverqueue/river"
	"github.com/rs/zerolog/log"
	entTwitchCategory "github.com/zibbp/ganymede/ent/twitchcategory"
	"github.com/zibbp/ganymede/internal/auth"
	"github.com/zibbp/ganymede/internal/errors"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/tasks"
	tasks_shared "github.com/zibbp/ganymede/internal/tasks/shared"
	"github.com/zibbp/ganymede/internal/vod"
)

func liveServiceFromContext(ctx context.Context) (*live.Service, error) {
	liveService, exists := ctx.Value(tasks_shared.LiveServiceKey).(*live.Service)
	if !exists || liveService == nil {
		return nil, errors.New("live service not found in context")
	}

	return liveService, nil
}

// Check watched channels for new videos
type CheckChannelsForNewVideosArgs struct{}

func (CheckChannelsForNewVideosArgs) Kind() string { return tasks.TaskCheckChannelForNewVideos }

func (w CheckChannelsForNewVideosArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
	}
}

func (w CheckChannelsForNewVideosArgs) Timeout(job *river.Job[CheckChannelsForNewVideosArgs]) time.Duration {
	return 10 * time.Minute
}

type CheckChannelsForNewVideosWorker struct {
	river.WorkerDefaults[CheckChannelsForNewVideosArgs]
}

func (w CheckChannelsForNewVideosWorker) Work(ctx context.Context, job *river.Job[CheckChannelsForNewVideosArgs]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()
	logger.Info().Msg("starting task")

	liveService, err := liveServiceFromContext(ctx)
	if err != nil {
		return err
	}

	err = liveService.CheckVodWatchedChannels(ctx, logger)
	if err != nil {
		return err
	}

	logger.Info().Msg("task completed")

	return nil
}

// Prune videos
type PruneVideosArgs struct{}

func (PruneVideosArgs) Kind() string { return tasks.TaskPruneVideos }

func (w PruneVideosArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
	}
}

func (w PruneVideosArgs) Timeout(job *river.Job[PruneVideosArgs]) time.Duration {
	return 1 * time.Minute
}

type PruneVideosWorker struct {
	river.WorkerDefaults[PruneVideosArgs]
}

func (w PruneVideosWorker) Work(ctx context.Context, job *river.Job[PruneVideosArgs]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()
	logger.Info().Msg("starting task")

	store, err := tasks.StoreFromContext(ctx)
	if err != nil {
		return err
	}

	err = vod.PruneVideos(ctx, store)
	if err != nil {
		return err
	}

	logger.Info().Msg("task completed")

	return nil
}

// Import Twitch categories
type ImportCategoriesArgs struct{}

func (ImportCategoriesArgs) Kind() string { return tasks.TaskImportVideos }

func (w ImportCategoriesArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
	}
}

func (w ImportCategoriesArgs) Timeout(job *river.Job[ImportCategoriesArgs]) time.Duration {
	return 1 * time.Minute
}

type ImportCategoriesWorker struct {
	river.WorkerDefaults[ImportCategoriesArgs]
}

func (w ImportCategoriesWorker) Work(ctx context.Context, job *river.Job[ImportCategoriesArgs]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()
	logger.Info().Msg("starting task")

	store, err := tasks.StoreFromContext(ctx)
	if err != nil {
		return err
	}

	platform, err := tasks.PlatformFromContext(ctx)
	if err != nil {
		return err
	}

	categories, err := platform.GetCategories(ctx)
	if err != nil {
		return err
	}

	logger.Info().Msgf("importing %d categories", len(categories))

	// upsert categories
	for _, category := range categories {
		err = store.Client.TwitchCategory.Create().SetID(category.ID).SetName(category.Name).OnConflictColumns(entTwitchCategory.FieldID).UpdateNewValues().Exec(context.Background())
		if err != nil {
			return fmt.Errorf("failed to upsert twitch category: %v", err)
		}
	}

	logger.Info().Msg("task completed")

	return nil
}

// Authenticate with Platform
type AuthenticatePlatformArgs struct{}

func (AuthenticatePlatformArgs) Kind() string { return tasks.TaskAuthenticatePlatform }

func (w AuthenticatePlatformArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
	}
}

func (w AuthenticatePlatformArgs) Timeout(job *river.Job[AuthenticatePlatformArgs]) time.Duration {
	return 1 * time.Minute
}

type AuthenticatePlatformWorker struct {
	river.WorkerDefaults[AuthenticatePlatformArgs]
}

func (w AuthenticatePlatformWorker) Work(ctx context.Context, job *river.Job[AuthenticatePlatformArgs]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()
	logger.Info().Msg("starting task")

	platform, err := tasks.PlatformFromContext(ctx)
	if err != nil {
		return err
	}

	_, err = platform.Authenticate(ctx)
	if err != nil {
		return err
	}

	logger.Info().Msg("task completed")

	return nil
}

// Fetch Json Web Keys if using OIDC
type FetchJWKSArgs struct{}

func (FetchJWKSArgs) Kind() string { return tasks.TaskFetchJWKS }

func (w FetchJWKSArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
	}
}

func (w FetchJWKSArgs) Timeout(job *river.Job[FetchJWKSArgs]) time.Duration {
	return 1 * time.Minute
}

type FetchJWKSWorker struct {
	river.WorkerDefaults[FetchJWKSArgs]
}

func (w FetchJWKSWorker) Work(ctx context.Context, job *river.Job[FetchJWKSArgs]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()
	logger.Info().Msg("starting task")

	err := auth.FetchJWKS(ctx)
	if err != nil {
		return err
	}

	logger.Info().Msg("task completed")

	return nil
}
