package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type RiverJobRow struct {
	ID    int64
	State string
	Args  RiverJobArgs
}

type RiverJobArgs struct {
	VideoId  string            `json:"video_id"`
	Input    ArchiveVideoInput `json:"input"`
	Continue bool              `json:"continue"`
}

type HeartBeatInput struct {
	TaskId int64
	conn   *pgxpool.Pool
}

func startHeartBeatForTask(ctx context.Context, input HeartBeatInput) {
	logger := log.With().Str("task_id", fmt.Sprintf("%d", input.TaskId)).Logger()
	logger.Debug().Msg("starting heartbeat")

	// perform one-time update before starting the ticker
	if err := updateHeartbeat(ctx, input); err != nil {
		logger.Error().Err(err).Msg("failed to update heartbeat")
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Debug().Msg("heartbeat stopped due to context cancellation")
			return
		case <-ticker.C:
			if err := updateHeartbeat(ctx, input); err != nil {
				logger.Error().Err(err).Msg("failed to update heartbeat")
				return
			}
			logger.Debug().Msg("heartbeat updated")
		}
	}
}

func updateHeartbeat(ctx context.Context, input HeartBeatInput) error {

	if ctx.Err() == context.Canceled {
		return nil
	}

	jobRow, err := getRiverJobById(ctx, input.conn, input.TaskId)
	if err != nil {
		if err == context.Canceled || errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("failed to get river job: %w", err)
	}

	jobRow.Args.Input.HeartBeatTime = time.Now()
	err = updateRiverJobArgs(ctx, input.conn, input.TaskId, jobRow.Args)
	if err != nil {
		if err == context.Canceled || errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("failed to update river job args: %w", err)
	}

	return nil
}

func getRiverJobById(ctx context.Context, conn *pgxpool.Pool, id int64) (*RiverJobRow, error) {
	query := `
		SELECT id, state, args
		FROM river_job
		WHERE id = $1
	`

	var job RiverJobRow
	err := conn.QueryRow(ctx, query, id).Scan(
		&job.ID,
		&job.State,
		&job.Args,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("no river job found with id %d", id)
		}
		return nil, fmt.Errorf("error querying for river job: %w", err)
	}

	return &job, nil
}

func updateRiverJobArgs(ctx context.Context, conn *pgxpool.Pool, id int64, args RiverJobArgs) error {
	jsonBytes, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("error marshalling args: %w", err)
	}

	query := `
		UPDATE river_job
		SET args = $1
		WHERE id = $2
	`

	r, err := conn.Exec(ctx, query, jsonBytes, id)
	if err != nil {
		return fmt.Errorf("error updating river job: %w", err)
	}

	if r.RowsAffected() == 0 {
		return fmt.Errorf("no river job found with id %d", id)
	}

	return nil
}
