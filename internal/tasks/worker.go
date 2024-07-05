package tasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/platform"
	platform_twitch "github.com/zibbp/ganymede/internal/platform/twitch"
)

type contextKey string

const storeKey contextKey = "store"
const platformKey contextKey = "platform"

type RiverWorkerInput struct {
	DB_URL string
}

type RiverWorkerClient struct {
	Ctx            context.Context
	PgxPool        *pgxpool.Pool
	RiverPgxDriver *riverpgxv5.Driver
	Client         *river.Client[pgx.Tx]
}

func NewRiverWorker(input RiverWorkerInput, db *database.Database, platformService platform.PlatformService[platform_twitch.TwitchVideoInfo, platform_twitch.TwitchLivestreamInfo, platform_twitch.TwitchChannel]) (*RiverWorkerClient, error) {
	rc := &RiverWorkerClient{}

	workers := river.NewWorkers()
	if err := river.AddWorkerSafely(workers, &WatchdogWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &CreateDirectoryWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &SaveVideoInfoWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &DownloadTumbnailsWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &DownloadVideoWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &PostProcessVideoWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &MoveVideoWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &DownloadChatWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &RenderChatWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &MoveChatWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &DownloadLiveVideoWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &DownloadLiveChatWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &ConvertLiveChatWorker{}); err != nil {
		return rc, err
	}

	rc.Ctx = context.Background()

	// create postgres pool connection
	pool, err := pgxpool.New(rc.Ctx, input.DB_URL)
	if err != nil {
		return rc, fmt.Errorf("error connecting to postgres: %v", err)
	}
	rc.PgxPool = pool

	// create river pgx driver
	rc.RiverPgxDriver = riverpgxv5.New(rc.PgxPool)

	// periodicJobs := setupPeriodicJobs()

	// create river client
	riverClient, err := river.NewClient(rc.RiverPgxDriver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault:  {MaxWorkers: 5},
			"video-download":    {MaxWorkers: 5},
			"video-postprocess": {MaxWorkers: 5},
			"chat-render":       {MaxWorkers: 5},
		},
		Workers:              workers,
		JobTimeout:           -1,
		RescueStuckJobsAfter: 49 * time.Hour,
		// PeriodicJobs:         periodicJobs,
		ErrorHandler: &CustomErrorHandler{},
	})
	if err != nil {
		return rc, fmt.Errorf("error creating river client: %v", err)
	}
	rc.Client = riverClient

	// put store in context for workers
	rc.Ctx = context.WithValue(rc.Ctx, storeKey, db)

	// put platform in context for workers
	rc.Ctx = context.WithValue(rc.Ctx, platformKey, platformService)

	return rc, nil
}

func (rc *RiverWorkerClient) Start() error {
	log.Info().Str("name", rc.Client.ID()).Msg("starting wortker")
	if err := rc.Client.Start(rc.Ctx); err != nil {
		return err
	}
	return nil
}

func (rc *RiverWorkerClient) Stop() error {
	if err := rc.Client.Stop(rc.Ctx); err != nil {
		return err
	}
	return nil
}

// func (rc *RiverWorkerClient) GetPeriodicJobs() []river.PeriodicJob {
// 	srv := archive.NewService()
// 	return nil
// }

func StoreFromContext(ctx context.Context) (*database.Database, error) {
	store, exists := ctx.Value(storeKey).(*database.Database)
	if !exists || store == nil {
		return nil, errors.New("store not found in context")
	}

	return store, nil
}

func PlatformFromContext(ctx context.Context) (platform.PlatformService[platform_twitch.TwitchVideoInfo, platform_twitch.TwitchLivestreamInfo, platform_twitch.TwitchChannel], error) {
	platform, exists := ctx.Value(platformKey).(platform.PlatformService[platform_twitch.TwitchVideoInfo, platform_twitch.TwitchLivestreamInfo, platform_twitch.TwitchChannel])
	if !exists || platform == nil {
		return nil, errors.New("platform not found in context")
	}

	return platform, nil
}
