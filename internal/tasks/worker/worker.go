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

type periodicTask struct {
	Job        *river.PeriodicJob
	Kind       string
	Schedule   string
	RunOnStart bool
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

	periodicTasks, err := getPeriodicTasks()
	if err != nil {
		return rc, err
	}

	// Print out the periodic tasks that are registered and enabled
	periodicJobs := make([]*river.PeriodicJob, 0, len(periodicTasks))
	log.Info().Int("count", len(periodicTasks)).Msg("enabled periodic jobs registered")
	for _, task := range periodicTasks {
		periodicJobs = append(periodicJobs, task.Job)
		log.Info().Str("kind", task.Kind).Str("schedule", task.Schedule).Bool("run_on_start", task.RunOnStart).Msg("enabled periodic job registered")
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

func getPeriodicTasks() ([]periodicTask, error) {
	env := config.GetEnvConfig()
	midnightCron, err := cron.ParseStandard("0 0 * * *")
	if err != nil {
		return nil, err
	}

	// get interval configs
	configCheckLiveInterval := config.Get().LiveCheckInterval
	configCheckVideoInterval := config.Get().VideoCheckInterval
	configGenerateNFOFiles := config.Get().Archive.GenerateNFOFiles
	if configCheckLiveInterval < 15 {
		log.Warn().Msg("Live check interval should not be less than 15 seconds.")
	}
	configPeriodicUpdateChannels := config.Get().Tasks.PeriodicUpdateChannels

	periodicTasks := []periodicTask{
		// Archive jobs heartbeat once per minute and are considered stale after
		// 90 seconds. Run the watchdog every minute so a cancellation that is
		// inside its finalization grace window is revisited promptly instead of
		// waiting another five minutes.
		newPeriodicTask(tasks.TaskArchiveWatchdog, river.PeriodicInterval(time.Minute), "1m", true,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks.WatchdogArgs{}, periodicInsertOpts(time.Minute)
			},
		),

		// check watched channels for live streams
		// run at specified interval
		newPeriodicTask(tasks.TaskCheckChannelsForLivestreams, river.PeriodicInterval(time.Duration(configCheckLiveInterval)*time.Second), fmt.Sprintf("%ds", configCheckLiveInterval), false,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.CheckChannelsForLivestreamsArgs{}, periodicInsertOpts(time.Duration(configCheckLiveInterval) * time.Second)
			},
		),

		// check watched channels for new videos
		// run at specified interval
		newPeriodicTask(tasks.TaskCheckChannelsForNewVideos, river.PeriodicInterval(time.Duration(configCheckVideoInterval)*time.Minute), fmt.Sprintf("%dm", configCheckVideoInterval), false,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.CheckChannelsForNewVideosArgs{}, periodicInsertOpts(time.Duration(configCheckVideoInterval) * time.Minute)
			},
		),

		// check watched channels for new clips
		// runs once a day at midnight
		newPeriodicTask(tasks.TaskCheckChannelsForNewClips, midnightCron, "daily at midnight", true,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.TaskCheckChannelForNewClipsArgs{}, periodicInsertOpts(24 * time.Hour)
			},
		),

		// prune videos
		// runs once a day at midnight
		newPeriodicTask(tasks.TaskPruneVideos, midnightCron, "daily at midnight", false,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.PruneVideosArgs{}, periodicInsertOpts(24 * time.Hour)
			},
		),

		// import categories
		// runs once a day at midnight
		newPeriodicTask(tasks.TaskImportVideos, midnightCron, "daily at midnight", true,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.ImportCategoriesArgs{}, periodicInsertOpts(24 * time.Hour)
			},
		),

		// authenticate to platform
		// runs every hour
		newPeriodicTask(tasks.TaskAuthenticatePlatform, river.PeriodicInterval(time.Hour), "1h", true,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.AuthenticatePlatformArgs{}, periodicInsertOpts(time.Hour)
			},
		),

		// update video storage usage
		// runs once a day at midnight
		newPeriodicTask(tasks.TaskUpdateVideoStorageUsage, midnightCron, "daily at midnight", false,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks.UpdateVideoStorageUsage{}, periodicInsertOpts(24 * time.Hour)
			},
		),

		// update channel storage usage
		// runs every hour
		newPeriodicTask(tasks.TaskUpdateChannelStorageUsage, river.PeriodicInterval(time.Hour), "1h", true,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks.UpdateChannelStorageUsage{}, periodicInsertOpts(time.Hour)
			},
		),

		// process playlist video rules
		// runs every hour
		newPeriodicTask(tasks.TaskProcessPlaylistVideoRules, river.PeriodicInterval(time.Hour), "1h", false,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.ProcessPlaylistVideoRulesArgs{}, periodicInsertOpts(time.Hour)
			},
		),

		// prune log files
		// runs once a day at midnight
		newPeriodicTask(tasks.TaskPruneLogFiles, midnightCron, "daily at midnight", false,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.PruneLogFilesArgs{}, periodicInsertOpts(24 * time.Hour)
			},
		),
	}

	if configGenerateNFOFiles {
		periodicTasks = append(periodicTasks, newPeriodicTask(tasks.TaskGenerateNFOFiles, midnightCron, "daily at midnight", true,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks.GenerateNFOFilesArgs{}, periodicInsertOpts(24 * time.Hour)
			},
		))
	}

	if configPeriodicUpdateChannels {
		periodicTasks = append(periodicTasks, newPeriodicTask(tasks.TaskUpdateTwitchChannels, river.PeriodicInterval(12*time.Hour), "12h", false,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.UpdateTwitchChannelsArgs{}, periodicInsertOpts(12 * time.Hour)
			},
		))
	}

	// check jwks
	if env.OAuthEnabled {
		// runs once a day at midnight
		periodicTasks = append(periodicTasks, newPeriodicTask(tasks.TaskFetchJWKS, midnightCron, "daily at midnight", true,
			func() (river.JobArgs, *river.InsertOpts) {
				return tasks_periodic.FetchJWKSArgs{}, periodicInsertOpts(24 * time.Hour)
			},
		))
	}

	return periodicTasks, nil
}

func newPeriodicTask(kind string, schedule river.PeriodicSchedule, scheduleDescription string, runOnStart bool, constructor river.PeriodicJobConstructor) periodicTask {
	return periodicTask{
		Job:        river.NewPeriodicJob(schedule, constructor, &river.PeriodicJobOpts{RunOnStart: runOnStart}),
		Kind:       kind,
		Schedule:   scheduleDescription,
		RunOnStart: runOnStart,
	}
}

func periodicInsertOpts(period time.Duration) *river.InsertOpts {
	return &river.InsertOpts{UniqueOpts: river.UniqueOpts{ByPeriod: period}}
}
