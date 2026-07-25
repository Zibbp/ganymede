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
	"github.com/riverqueue/river/rivertype"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/notification"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/tasks"
	tasks_periodic "github.com/zibbp/ganymede/internal/tasks/periodic"
	"github.com/zibbp/ganymede/internal/tasks/registry"
	tasks_shared "github.com/zibbp/ganymede/internal/tasks/shared"
)

type RiverWorkerInput struct {
	Context                 context.Context
	DB_URL                  string
	DB                      *database.Database
	LiveService             *live.Service
	PlatformTwitch          platform.Platform
	NotificationService     *notification.Service
	Enqueuer                tasks_shared.Enqueuer
	VideoDownloadWorkers    int
	VideoPostProcessWorkers int
	ChatDownloadWorkers     int
	ChatRenderWorkers       int
	SpriteThumbnailWorkers  int
}

type RiverWorkerClient struct {
	Ctx            context.Context
	Database       *database.Database
	PgxPool        *pgxpool.Pool
	RiverPgxDriver *riverpgxv5.Driver
	Client         *river.Client[pgx.Tx]
}

func NewRiverWorker(input RiverWorkerInput) (*RiverWorkerClient, error) {
	rc := &RiverWorkerClient{}
	rc.Database = input.DB

	workers, err := registry.New()
	if err != nil {
		return rc, err
	}

	if input.Context == nil {
		return rc, fmt.Errorf("worker context is required")
	}
	rc.Ctx = input.Context
	rc.Ctx = context.WithValue(rc.Ctx, tasks_shared.StoreKey, input.DB)
	rc.Ctx = context.WithValue(rc.Ctx, tasks_shared.PlatformTwitchKey, input.PlatformTwitch)
	rc.Ctx = context.WithValue(rc.Ctx, tasks_shared.NotificationServiceKey, input.NotificationService)
	rc.Ctx = context.WithValue(rc.Ctx, tasks_shared.LiveServiceKey, input.LiveService)
	rc.Ctx = context.WithValue(rc.Ctx, tasks_shared.EnqueuerKey, input.Enqueuer)

	periodicJobs, err := getPeriodicTasks()
	if err != nil {
		return rc, err
	}

	// create postgres pool connection
	pool, err := pgxpool.New(rc.Ctx, input.DB_URL)
	if err != nil {
		return rc, fmt.Errorf("error connecting to postgres: %v", err)
	}
	rc.PgxPool = pool

	// create river pgx driver
	rc.RiverPgxDriver = riverpgxv5.New(rc.PgxPool)

	// create river client
	archiveMiddleware := tasks.NewArchiveMiddleware()
	riverClient, err := river.NewClient(rc.RiverPgxDriver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault:                  {MaxWorkers: 100}, // non-resource intensive tasks or time sensitive tasks (live videos and chat)
			tasks.QueueVideoDownload:            {MaxWorkers: input.VideoDownloadWorkers},
			tasks.QueueVideoPostProcess:         {MaxWorkers: input.VideoPostProcessWorkers},
			tasks.QueueChatDownload:             {MaxWorkers: input.ChatDownloadWorkers},
			tasks.QueueChatRender:               {MaxWorkers: input.ChatRenderWorkers},
			tasks.QueueGenerateThumbnailSprites: {MaxWorkers: input.SpriteThumbnailWorkers},
		},
		Workers:         workers,
		Middleware:      []rivertype.Middleware{archiveMiddleware},
		PeriodicJobs:    periodicJobs,
		SoftStopTimeout: 30 * time.Second,
		ErrorHandler:    &tasks.CustomErrorHandler{},
		JobStuckHandler: func(ctx context.Context, params river.JobStuckHandlerParams) river.JobStuckHandlerResult {
			log.Error().Int64("job_id", params.ID).Str("kind", params.Kind).Str("queue", params.Queue).Int("total_stuck_jobs", params.TotalStuckJobs).Msg("River job did not stop after its timeout")
			return river.JobStuckHandlerResult{AddWorkerSlot: false}
		},
	})
	if err != nil {
		return rc, fmt.Errorf("error creating river client: %v", err)
	}

	log.Info().Str("default_workers", "100").Str("download_workers", strconv.Itoa(input.VideoDownloadWorkers)).Str("post_process_workers", strconv.Itoa(input.VideoPostProcessWorkers)).Str("chat_download_workers", strconv.Itoa(input.ChatDownloadWorkers)).Str("chat_render_workers", strconv.Itoa(input.ChatRenderWorkers)).Str("sprite_thumbnail_workers", strconv.Itoa(input.SpriteThumbnailWorkers)).Msg("created river client")

	rc.Client = riverClient
	archiveMiddleware.SetWorkerClient(riverClient)

	return rc, nil
}

func (rc *RiverWorkerClient) Start() error {
	log.Info().Str("name", rc.Client.ID()).Msg("starting worker")
	if err := rc.Client.Start(rc.Ctx); err != nil {
		return err
	}
	return nil
}

func (rc *RiverWorkerClient) Stop(ctx context.Context) error {
	if err := rc.Client.Stop(ctx); err != nil {
		return err
	}
	return nil
}

func (rc *RiverWorkerClient) Close() error {
	if rc.PgxPool != nil {
		rc.PgxPool.Close()
	}
	if rc.Database != nil {
		return rc.Database.Close()
	}
	return nil
}

func getPeriodicTasks() ([]*river.PeriodicJob, error) {
	env := config.GetEnvConfig()
	midnightCron, err := cron.ParseStandard("0 0 * * *")
	if err != nil {
		return nil, err
	}

	// get interval configs
	configCheckLiveInterval := config.Get().LiveCheckInterval
	configCheckVideoInterval := config.Get().VideoCheckInterval
	if configCheckLiveInterval < 15 {
		log.Warn().Msg("Live check interval should not be less than 15 seconds.")
	}

	periodicJobs := []*river.PeriodicJob{
		// Archive jobs heartbeat once per minute and are considered stale after
		// 90 seconds. Run the watchdog every minute so a cancellation that is
		// inside its finalization grace window is revisited promptly instead of
		// waiting another five minutes.
		river.NewPeriodicJob(
			river.PeriodicInterval(time.Minute),
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks.WatchdogArgs{}, periodicInsertOpts(time.Minute)
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		),

		// check watched channels for live streams
		// run at specified interval
		river.NewPeriodicJob(
			river.PeriodicInterval(time.Duration(configCheckLiveInterval)*time.Second),
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.CheckChannelsForLivestreamsArgs{}, periodicInsertOpts(time.Duration(configCheckLiveInterval) * time.Second)
			},
			&river.PeriodicJobOpts{RunOnStart: false},
		),

		// check watched channels for new videos
		// run at specified interval
		river.NewPeriodicJob(
			river.PeriodicInterval(time.Duration(configCheckVideoInterval)*time.Minute),
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.CheckChannelsForNewVideosArgs{}, periodicInsertOpts(time.Duration(configCheckVideoInterval) * time.Minute)
			},
			&river.PeriodicJobOpts{RunOnStart: false},
		),

		// check watched channels for new clips
		// runs once a day at midnight
		river.NewPeriodicJob(
			midnightCron,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.TaskCheckChannelForNewClipsArgs{}, periodicInsertOpts(24 * time.Hour)
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		),

		// prune videos
		// runs once a day at midnight
		river.NewPeriodicJob(
			midnightCron,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.PruneVideosArgs{}, periodicInsertOpts(24 * time.Hour)
			},
			&river.PeriodicJobOpts{RunOnStart: false},
		),

		// import categories
		// runs once a day at midnight
		river.NewPeriodicJob(
			midnightCron,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.ImportCategoriesArgs{}, periodicInsertOpts(24 * time.Hour)
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		),

		// authenticate to platform
		// runs every hour
		river.NewPeriodicJob(
			river.PeriodicInterval(1*time.Hour),
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.AuthenticatePlatformArgs{}, periodicInsertOpts(time.Hour)
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		),

		// update video storage usage
		// runs once a day at midnight
		river.NewPeriodicJob(
			midnightCron,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks.UpdateVideoStorageUsage{}, periodicInsertOpts(24 * time.Hour)
			},
			&river.PeriodicJobOpts{RunOnStart: false},
		),

		// update channel storage usage
		// runs every hour
		river.NewPeriodicJob(
			river.PeriodicInterval(1*time.Hour),
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks.UpdateChannelStorageUsage{}, periodicInsertOpts(time.Hour)
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		),

		// process playlist video rules
		// runs every hour
		river.NewPeriodicJob(
			river.PeriodicInterval(1*time.Hour),
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.ProcessPlaylistVideoRulesArgs{}, periodicInsertOpts(time.Hour)
			},
			&river.PeriodicJobOpts{RunOnStart: false},
		),

		// update twitch channels
		// runs every 12 hour
		river.NewPeriodicJob(
			river.PeriodicInterval(12*time.Hour),
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.UpdateTwitchChannelsArgs{}, periodicInsertOpts(12 * time.Hour)
			},
			&river.PeriodicJobOpts{RunOnStart: false},
		),

		// prune log files
		// runs once a day at midnight
		river.NewPeriodicJob(
			midnightCron,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.PruneLogFilesArgs{}, periodicInsertOpts(24 * time.Hour)
			},
			&river.PeriodicJobOpts{RunOnStart: false},
		),
	}

	// check jwks
	if env.OAuthEnabled {
		// runs once a day at midnight
		periodicJobs = append(periodicJobs, river.NewPeriodicJob(
			midnightCron,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.FetchJWKSArgs{}, periodicInsertOpts(24 * time.Hour)
			},
			&river.PeriodicJobOpts{RunOnStart: true},
		))
	}

	return periodicJobs, nil
}

func periodicInsertOpts(period time.Duration) *river.InsertOpts {
	return &river.InsertOpts{UniqueOpts: river.UniqueOpts{ByPeriod: period}}
}
