package tasks_client

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/tasks"
	"github.com/zibbp/ganymede/internal/utils"
)

type RiverClientInput struct {
	Database *database.Database
}

// RiverClient is an insertion-only River client. It intentionally uses the
// same database/sql pool as Ent, which allows InsertTx to participate in the
// same transaction as application state changes. It is never started as a
// worker client.
type RiverClient struct {
	Client *river.Client[*sql.Tx]
}

func NewRiverClient(input RiverClientInput) (*RiverClient, error) {
	if input.Database == nil || input.Database.SQLDB == nil {
		return nil, fmt.Errorf("database is required")
	}

	archiveMiddleware := tasks.NewArchiveMiddleware()
	riverClient, err := river.NewClient(riverdatabasesql.New(input.Database.SQLDB), &river.Config{
		// This client only inserts and administrates jobs. Workers and queues
		// belong exclusively to the started pgx worker client.
		Middleware: []rivertype.Middleware{archiveMiddleware},
	})
	if err != nil {
		return nil, err
	}

	return &RiverClient{Client: riverClient}, nil
}

func (rc *RiverClient) Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return rc.Client.Insert(ctx, args, opts)
}

func (rc *RiverClient) InsertTx(ctx context.Context, tx *sql.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return rc.Client.InsertTx(ctx, tx, args, opts)
}

func (rc *RiverClient) JobList(ctx context.Context, params *river.JobListParams) (*river.JobListResult, error) {
	return rc.Client.JobList(ctx, params)
}

// CancelJobsForQueueId cancels every active archive job for a queue. New jobs
// are found through indexed River metadata; the paginated args scan preserves
// compatibility with jobs inserted by older Ganymede releases.
func (rc *RiverClient) CancelJobsForQueueId(ctx context.Context, queueID uuid.UUID) error {
	states := []rivertype.JobState{
		rivertype.JobStateAvailable,
		rivertype.JobStatePending,
		rivertype.JobStateRetryable,
		rivertype.JobStateRunning,
		rivertype.JobStateScheduled,
	}
	seen := make(map[int64]struct{})

	metadata, err := json.Marshal(map[string]any{
		"ganymede": map[string]string{"queue_id": queueID.String()},
	})
	if err != nil {
		return err
	}
	modern := river.NewJobListParams().States(states...).Metadata(string(metadata)).First(500)
	if err := rc.cancelPages(ctx, modern, queueID, seen, false); err != nil {
		return err
	}

	legacy := river.NewJobListParams().States(states...).First(500)
	if err := rc.cancelPages(ctx, legacy, queueID, seen, true); err != nil {
		return err
	}

	if len(seen) == 0 {
		log.Info().Str("queue_id", queueID.String()).Msg("no active River jobs found for queue")
	}
	return nil
}

func (rc *RiverClient) cancelPages(ctx context.Context, params *river.JobListParams, queueID uuid.UUID, seen map[int64]struct{}, inspectArgs bool) error {
	for {
		result, err := rc.Client.JobList(ctx, params)
		if err != nil {
			return err
		}

		for _, job := range result.Jobs {
			if _, ok := seen[job.ID]; ok || !utils.Contains(job.Tags, "archive") {
				continue
			}
			if inspectArgs {
				var args struct {
					Input struct {
						QueueID uuid.UUID `json:"queue_id"`
					} `json:"input"`
				}
				if err := json.Unmarshal(job.EncodedArgs, &args); err != nil || args.Input.QueueID != queueID {
					continue
				}
			}

			if _, err := rc.Client.JobCancel(ctx, job.ID); err != nil {
				return fmt.Errorf("cancel River job %d: %w", job.ID, err)
			}
			seen[job.ID] = struct{}{}
		}

		if len(result.Jobs) < 500 || result.LastCursor == nil {
			return nil
		}
		params = params.After(result.LastCursor)
	}
}
