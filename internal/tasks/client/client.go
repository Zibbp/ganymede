package tasks_client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/tasks"
	"github.com/zibbp/ganymede/internal/utils"
)

type RiverClientInput struct {
	DB_URL string
}

type RiverClient struct {
	Ctx            context.Context
	PgxPool        *pgxpool.Pool
	RiverPgxDriver *riverpgxv5.Driver
	Client         *river.Client[pgx.Tx]
}

func NewRiverClient(input RiverClientInput) (*RiverClient, error) {
	rc := &RiverClient{}
	rc.Ctx = context.Background()

	// create postgres pool connection
	pool, err := pgxpool.New(rc.Ctx, input.DB_URL)
	if err != nil {
		return rc, err
	}
	rc.PgxPool = pool

	// create river pgx driver
	rc.RiverPgxDriver = riverpgxv5.New(rc.PgxPool)

	// periodicJobs := setupPeriodicJobs()

	// create river client
	riverClient, err := river.NewClient(rc.RiverPgxDriver, &river.Config{
		JobTimeout:           -1,
		RescueStuckJobsAfter: 49 * time.Hour,
		// PeriodicJobs:         periodicJobs,
	})
	if err != nil {
		return rc, err
	}

	rc.Client = riverClient

	return rc, nil
}

func (rc *RiverClient) Stop() error {
	if err := rc.Client.Stop(rc.Ctx); err != nil {
		return err
	}
	return nil
}

// Run river database migrations
func (rc *RiverClient) RunMigrations() error {
	migrator, err := rivermigrate.New(rc.RiverPgxDriver, nil)
	if err != nil {
		return fmt.Errorf("error creating river migrations: %v", err)
	}

	_, err = migrator.Migrate(rc.Ctx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{})
	if err != nil {
		return fmt.Errorf("error running river migrations: %v", err)
	}

	log.Info().Msg("successfully applied river migrations")

	return nil
}

// params := river.NewJobListParams().States(rivertype.JobStateRunning).First(10000)
func (rc *RiverClient) JobList(ctx context.Context, params *river.JobListParams) (*river.JobListResult, error) {
	// fetch jobs
	jobs, err := rc.Client.JobList(ctx, params)
	if err != nil {
		return nil, err
	}

	return jobs, nil
}

// CancelJobsForQueueId cancels all jobs for a queue. This fetches all jobs and check if the queue id of the job matches by unmarshalling the job args
func (rc *RiverClient) CancelJobsForQueueId(ctx context.Context, queueId uuid.UUID) error {
	params := river.NewJobListParams().States(rivertype.JobStateRunning, rivertype.JobStatePending, rivertype.JobStateScheduled, rivertype.JobStateRetryable).First(10000)
	jobs, err := rc.Client.JobList(ctx, params)
	if err != nil {
		return err
	}

	if len(jobs.Jobs) == 0 {
		log.Info().Msg("no jobs found for queue id, nothing to cancel")
		return nil
	}

	// check jobs
	for _, job := range jobs.Jobs {
		// only check archive jobs
		if utils.Contains(job.Tags, "archive") {
			// unmarshal args
			var args tasks.RiverJobArgs

			if err := json.Unmarshal(job.EncodedArgs, &args); err != nil {
				return err
			}

			if args.Input.QueueId == queueId {
				_, err := rc.Client.JobCancel(ctx, job.ID)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
