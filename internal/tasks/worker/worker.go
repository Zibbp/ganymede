package tasks_worker

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/tasks"
	tasks_periodic "github.com/zibbp/ganymede/internal/tasks/periodic"
	tasks_shared "github.com/zibbp/ganymede/internal/tasks/shared"
)

type RiverWorkerInput struct {
	DB_URL                  string
	DB                      *database.Database
	PlatformTwitch          platform.Platform
	VideoDownloadWorkers    int
	VideoPostProcessWorkers int
	ChatDownloadWorkers     int
	ChatRenderWorkers       int
	SpriteThumbnailWorkers  int
}

type RiverWorkerClient struct {
	Ctx            context.Context
	PgxPool        *pgxpool.Pool
	RiverPgxDriver *riverpgxv5.Driver
	Client         *river.Client[pgx.Tx]
}

func NewRiverWorker(input RiverWorkerInput) (*RiverWorkerClient, error) {
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
	if err := river.AddWorkerSafely(workers, &tasks_periodic.PruneVideosWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks_periodic.ImportCategoriesWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks_periodic.AuthenticatePlatformWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks_periodic.FetchJWKSWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks_periodic.SaveVideoChaptersWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks.UpdateStreamVideoIdWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks.GenerateStaticThubmnailWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks.GenerateSpriteThumbnailWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks.UpdateLiveStreamMetadataWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks_periodic.TaskCheckChannelForNewClipsWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks_periodic.CheckChannelsForLivestreamsWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks.UpdateVideoStorageUsageWorker{}); err != nil {
		return rc, err
	}
	if err := river.AddWorkerSafely(workers, &tasks.UpdateChannelStorageUsageWorker{}); err != nil {
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

	// create river client
	riverClient, err := river.NewClient(rc.RiverPgxDriver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault:                  {MaxWorkers: 100}, // non-resource intensive tasks or time sensitive tasks (live videos and chat)
			tasks.QueueVideoDownload:            {MaxWorkers: input.VideoDownloadWorkers},
			tasks.QueueVideoPostProcess:         {MaxWorkers: input.VideoPostProcessWorkers},
			tasks.QueueChatDownload:             {MaxWorkers: input.ChatDownloadWorkers},
			tasks.QueueChatRender:               {MaxWorkers: input.ChatRenderWorkers},
			tasks.QueueGenerateThumbnailSprites: {MaxWorkers: input.SpriteThumbnailWorkers},
		},
		Workers:              workers,
		JobTimeout:           -1,
		RescueStuckJobsAfter: 49 * time.Hour,
		ErrorHandler:         &tasks.CustomErrorHandler{},
	})
	if err != nil {
		return rc, fmt.Errorf("error creating river client: %v", err)
	}

	log.Info().Str("default_workers", "100").Str("download_workers", strconv.Itoa(input.VideoDownloadWorkers)).Str("post_process_workers", strconv.Itoa(input.VideoPostProcessWorkers)).Str("chat_download_workers", strconv.Itoa(input.ChatDownloadWorkers)).Str("chat_render_workers", strconv.Itoa(input.ChatRenderWorkers)).Str("sprite_thumbnail_workers", strconv.Itoa(input.SpriteThumbnailWorkers)).Msg("created river client")

	rc.Client = riverClient

	// put store in context for workers
	rc.Ctx = context.WithValue(rc.Ctx, tasks_shared.StoreKey, input.DB)

	// put platform in context for workers
	rc.Ctx = context.WithValue(rc.Ctx, tasks_shared.PlatformTwitchKey, input.PlatformTwitch)

	return rc, nil
}

func (rc *RiverWorkerClient) Start() error {
	log.Info().Str("name", rc.Client.ID()).Msg("starting worker")
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

func (rc *RiverWorkerClient) GetPeriodicTasks(liveService *live.Service) ([]*river.PeriodicJob, error) {
	env := config.GetEnvConfig()
	midnightCron, err := cron.ParseStandard("0 0 * * *")
	if err != nil {
		return nil, err
	}

	// put services in ctx for workers
	rc.Ctx = context.WithValue(rc.Ctx, tasks_shared.LiveServiceKey, liveService)

	// get interval configs
	configCheckLiveInterval := config.Get().LiveCheckInterval
	configCheckVideoInterval := config.Get().VideoCheckInterval
	if configCheckLiveInterval < 15 {
		log.Warn().Msg("Live check interval should not be less than 15 seconds.")
	}

	periodicJobs := []*river.PeriodicJob{
		// archive watchdog
		// runs every 5 minutes
		river.NewPeriodicJob(
			river.PeriodicInterval(5*time.Minute),
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks.WatchdogArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		),

		// check watched channels for live streams
		// run at specified interval
		river.NewPeriodicJob(
			river.PeriodicInterval(time.Duration(configCheckLiveInterval)*time.Second),
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.CheckChannelsForLivestreamsArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: false},
		),

		// check watched channels for new videos
		// run at specified interval
		river.NewPeriodicJob(
			river.PeriodicInterval(time.Duration(configCheckVideoInterval)*time.Minute),
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.CheckChannelsForNewVideosArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: false},
		),

		// check watched channels for new clips
		// runs once a day at midnight
		river.NewPeriodicJob(
			midnightCron,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.TaskCheckChannelForNewClipsArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		),

		// prune videos
		// runs once a day at midnight
		river.NewPeriodicJob(
			midnightCron,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.PruneVideosArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: false},
		),

		// import categories
		// runs once a day at midnight
		river.NewPeriodicJob(
			midnightCron,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.ImportCategoriesArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		),

		// authenticate to platform
		// runs once a day at midnight
		river.NewPeriodicJob(
			midnightCron,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.AuthenticatePlatformArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		),

		// update video storage usage
		// runs once a day at midnight
		river.NewPeriodicJob(
			midnightCron,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks.UpdateVideoStorageUsage{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: false},
		),

		// update channel storage usage
		// runs every hour
		river.NewPeriodicJob(
			river.PeriodicInterval(1*time.Hour),
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks.UpdateChannelStorageUsage{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		),
	}

	// check jwks
	if env.OAuthEnabled {
		// runs once a day at midnight
		periodicJobs = append(periodicJobs, river.NewPeriodicJob(
			midnightCron,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.FetchJWKSArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		))
	}

	return periodicJobs, nil
}
