package tasks_periodic

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/riverqueue/river"
	"github.com/rs/zerolog/log"
	entTwitchCategory "github.com/zibbp/ganymede/ent/twitchcategory"
	"github.com/zibbp/ganymede/internal/auth"
	"github.com/zibbp/ganymede/internal/errors"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/tasks"
	tasks_shared "github.com/zibbp/ganymede/internal/tasks/shared"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
)

func liveServiceFromContext(ctx context.Context) (*live.Service, error) {
	liveService, exists := ctx.Value(tasks_shared.LiveServiceKey).(*live.Service)
	if !exists || liveService == nil {
		return nil, errors.New("live service not found in context")
	}

	return liveService, nil
}

// Check watched channels for live streams
type CheckChannelsForLivestreamsArgs struct{}

func (CheckChannelsForLivestreamsArgs) Kind() string { return tasks.TaskCheckChannelsForLivestreams }

func (w CheckChannelsForLivestreamsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
	}
}

func (w CheckChannelsForLivestreamsArgs) Timeout(job *river.Job[CheckChannelsForLivestreamsArgs]) time.Duration {
	return 10 * time.Minute
}

type CheckChannelsForLivestreamsWorker struct {
	river.WorkerDefaults[CheckChannelsForLivestreamsArgs]
}

func (w CheckChannelsForLivestreamsWorker) Work(ctx context.Context, job *river.Job[CheckChannelsForLivestreamsArgs]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()
	logger.Info().Msg("starting task")

	liveService, err := liveServiceFromContext(ctx)
	if err != nil {
		return err
	}

	err = liveService.Check(ctx)
	if err != nil {
		return err
	}

	logger.Info().Msg("task completed")

	return nil
}

// Check watched channels for new videos
type CheckChannelsForNewVideosArgs struct{}

func (CheckChannelsForNewVideosArgs) Kind() string { return tasks.TaskCheckChannelsForNewVideos }

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

// Check watched channels for new clips
type TaskCheckChannelForNewClipsArgs struct{}

func (TaskCheckChannelForNewClipsArgs) Kind() string { return tasks.TaskCheckChannelsForNewClips }

func (w TaskCheckChannelForNewClipsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
	}
}

func (w TaskCheckChannelForNewClipsArgs) Timeout(job *river.Job[TaskCheckChannelForNewClipsArgs]) time.Duration {
	return 10 * time.Minute
}

type TaskCheckChannelForNewClipsWorker struct {
	river.WorkerDefaults[TaskCheckChannelForNewClipsArgs]
}

func (w TaskCheckChannelForNewClipsWorker) Work(ctx context.Context, job *river.Job[TaskCheckChannelForNewClipsArgs]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()
	logger.Info().Msg("starting task")

	liveService, err := liveServiceFromContext(ctx)
	if err != nil {
		return err
	}

	err = liveService.CheckWatchedChannelClips(ctx, logger)
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

// Update video storage usage
type UpdateVideoStorageUsage struct{}

func (UpdateVideoStorageUsage) Kind() string { return tasks.TaskUpdateVideoStorageUsage }

func (w UpdateVideoStorageUsage) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
	}
}

func (w UpdateVideoStorageUsage) Timeout(job *river.Job[UpdateVideoStorageUsage]) time.Duration {
	return 5 * time.Minute
}

type UpdateVideoStorageUsageWorker struct {
	river.WorkerDefaults[UpdateVideoStorageUsage]
}

func (w UpdateVideoStorageUsageWorker) Work(ctx context.Context, job *river.Job[UpdateVideoStorageUsage]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()
	logger.Info().Msg("starting task")

	store, err := tasks.StoreFromContext(ctx)
	if err != nil {
		return err
	}

	// Fetch all videos and update their storage size
	// This updates all videos in the database to ensure their storage size is accurate if their files have changed (e.g. after an external re-encode)
	videos, err := store.Client.Vod.Query().All(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch videos: %v", err)
	}

	for _, video := range videos {
		if video.VideoPath == "" {
			logger.Warn().Msgf("video %s has no video path, skipping storage size update", video.ID)
		}
		directory := filepath.Dir(video.VideoPath)
		// If hls video need to go up one more directory as the hls files are in a subdirectory
		if video.VideoHlsPath != "" {
			directory = filepath.Dir(directory)
		}

		size, err := utils.GetSizeOfDirectory(directory)
		if err != nil {
			logger.Error().Err(err).Msgf("failed to get size of directory %s for video %s", directory, video.ID)
			continue // Skip this video if we can't get the size
		}

		// Check if size needs to be updated
		if video.StorageSizeBytes != size {
			_, err = store.Client.Vod.UpdateOneID(video.ID).SetStorageSizeBytes(size).Save(ctx)
			if err != nil {
				return fmt.Errorf("failed to update video %s storage size: %v", video.ID, err)
			}

			logger.Info().Msgf("updated video %s storage size to %d bytes", video.ID, size)
		} else {
			logger.Debug().Msgf("video %s storage size is already %d bytes, skipping update", video.ID, size)
		}
	}

	logger.Info().Msg("task completed")

	return nil
}
