package tasks_periodic

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/rs/zerolog/log"
	entPlaylist "github.com/zibbp/ganymede/ent/playlist"
	entPlaylistGroup "github.com/zibbp/ganymede/ent/playlistrulegroup"
	entTwitchCategory "github.com/zibbp/ganymede/ent/twitchcategory"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/auth"
	"github.com/zibbp/ganymede/internal/errors"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/playlist"
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

// Process playlist video rules
type ProcessPlaylistVideoRulesArgs struct{}

func (ProcessPlaylistVideoRulesArgs) Kind() string { return tasks.TaskProcessPlaylistVideoRules }

func (w ProcessPlaylistVideoRulesArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
	}
}

func (w ProcessPlaylistVideoRulesArgs) Timeout(job *river.Job[ProcessPlaylistVideoRulesArgs]) time.Duration {
	return 5 * time.Minute
}

type ProcessPlaylistVideoRulesWorker struct {
	river.WorkerDefaults[ProcessPlaylistVideoRulesArgs]
}

// ProcessPlaylistVideoRulesWorker processes playlist video rules by checking if videos should be added or removed from playlists based on defined rules.
func (w ProcessPlaylistVideoRulesWorker) Work(ctx context.Context, job *river.Job[ProcessPlaylistVideoRulesArgs]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()
	logger.Info().Msg("starting task")

	store, err := tasks.StoreFromContext(ctx)
	if err != nil {
		return err
	}

	playlistService := playlist.NewService(store)

	// Get all playlists
	var playlistIds []uuid.UUID
	err = store.Client.Playlist.
		Query().
		Select(entPlaylist.FieldID).
		Scan(ctx, &playlistIds)
	if err != nil {
		return fmt.Errorf("failed to get playlists: %w", err)
	}

	// Get all videos
	var videoIds []uuid.UUID
	err = store.Client.Vod.
		Query().
		Select(entPlaylist.FieldID).
		Scan(ctx, &videoIds)
	if err != nil {
		return fmt.Errorf("failed to get videos: %w", err)
	}

	log.Info().Msgf("found %d playlists and %d videos, processing playlist rules", len(playlistIds), len(videoIds))
	for _, playlistID := range playlistIds {
		logger := logger.With().Str("playlist_id", playlistID.String()).Logger()
		logger.Info().Msg("processing playlist video rules")

		// Query rule groups for the playlist
		groups, err := store.Client.PlaylistRuleGroup.
			Query().
			Where(entPlaylistGroup.HasPlaylistWith(entPlaylist.IDEQ(playlistID))).
			WithRules().
			All(ctx)
		if err != nil {
			logger.Error().Err(err).Msg("failed to get playlist rule groups")
			continue
		}
		if len(groups) == 0 {
			logger.Info().Msg("no rule groups found for playlist, skipping")
			continue
		}

		// Evaluate each video against the playlist rules
		for _, videoID := range videoIds {
			shouldBeIn, err := playlistService.ShouldVideoBeInPlaylist(ctx, videoID, playlistID)
			if err != nil {
				logger.Error().Err(err).Msg("failed to evaluate video against playlist rules")
				continue
			}

			if shouldBeIn {
				logger.Debug().Msgf("video %s should be in playlist %s", videoID, playlistID)
				err = playlistService.AddVodToPlaylist(ctx, playlistID, videoID)
				if err != nil {
					logger.Error().Err(err).Msg("failed to add video to playlist")
					continue
				}
				logger.Info().Msgf("video %s added to playlist %s", videoID, playlistID)

			} else {
				logger.Debug().Msgf("video %s should NOT be in playlist %s", videoID, playlistID)
				// Check if video is actually in the playlist before attempting removal
				inPlaylist, err := store.Client.Playlist.Query().
					Where(entPlaylist.HasVodsWith(entVod.IDEQ(videoID))).
					Where(entPlaylist.IDEQ(playlistID)).
					Exist(ctx)
				if err != nil {
					logger.Error().Err(err).Msg("failed to check if video is in playlist")
					continue
				}
				if inPlaylist {
					err = playlistService.DeleteVodFromPlaylist(ctx, playlistID, videoID)
					if err != nil {
						logger.Error().Err(err).Msg("failed to remove video from playlist")
						continue
					}
					logger.Info().Msgf("video %s removed from playlist %s", videoID, playlistID)
				}
			}
		}
	}

	logger.Info().Msg("task completed")

	return nil
}
