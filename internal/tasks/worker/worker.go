package tasks_worker

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/platform"
	platform_twitch "github.com/zibbp/ganymede/internal/platform/twitch"
	"github.com/zibbp/ganymede/internal/tasks"
	tasks_periodic "github.com/zibbp/ganymede/internal/tasks/periodic"
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
	if err := river.AddWorkerSafely(workers, &tasks.WatchdogWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks.CreateDirectoryWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks.SaveVideoInfoWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks.DownloadTumbnailsWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks.DownloadVideoWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks.PostProcessVideoWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks.MoveVideoWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks.DownloadChatWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks.RenderChatWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks.MoveChatWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks.DownloadLiveVideoWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks.DownloadLiveChatWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks.ConvertLiveChatWorker{}); err != nil {
		return rc, err
	}
	// periodic tasks
	if err := river.AddWorkerSafely(workers, &tasks_periodic.CheckChannelsForNewVideosWorker{}); err != nil {
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
		ErrorHandler: &tasks.CustomErrorHandler{},
	})
	if err != nil {
		return rc, fmt.Errorf("error creating river client: %v", err)
	}
	rc.Client = riverClient

	// put store in context for workers
	rc.Ctx = context.WithValue(rc.Ctx, "store", db)

	// put platform in context for workers
	rc.Ctx = context.WithValue(rc.Ctx, "platform", platformService)

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

func (rc *RiverWorkerClient) GetPeriodicTasks(liveService *live.Service) []*river.PeriodicJob {

	// put services in ctx for workers
	rc.Ctx = context.WithValue(rc.Ctx, "live_service", liveService)

	periodicJobs := []*river.PeriodicJob{
		// run watchdog job every minute
		river.NewPeriodicJob(
			river.PeriodicInterval(1*time.Minute),
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks.WatchdogArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		),

		// check watched channels for new videos
		river.NewPeriodicJob(
			river.PeriodicInterval(1*time.Minute),
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.CheckChannelsForNewVideosArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		),
	}

	return periodicJobs
}
