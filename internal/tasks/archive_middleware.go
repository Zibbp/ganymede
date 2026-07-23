package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/utils"
)

const archiveHeartbeatInterval = time.Minute

type archiveJobMetadata struct {
	QueueID            uuid.UUID `json:"queue_id"`
	RecoveryGeneration int       `json:"recovery_generation"`
}

// ArchiveProgressOutput is stored through River's public JobUpdate API. It
// replaces the legacy behavior that rewrote encoded job arguments directly.
type ArchiveProgressOutput struct {
	HeartbeatAt         time.Time `json:"heartbeat_at"`
	ShutdownInterrupted bool      `json:"shutdown_interrupted,omitempty"`
}

// ArchiveMiddleware attaches searchable queue metadata during inserts and
// records progress for running archive jobs. The worker client is optional so
// the same middleware can be installed on the insertion-only producer client.
type ArchiveMiddleware struct {
	river.MiddlewareDefaults
	client *river.Client[pgx.Tx]
}

func NewArchiveMiddleware() *ArchiveMiddleware { return &ArchiveMiddleware{} }

func (m *ArchiveMiddleware) SetWorkerClient(client *river.Client[pgx.Tx]) {
	m.client = client
}

func (m *ArchiveMiddleware) InsertMany(ctx context.Context, manyParams []*rivertype.JobInsertParams, doInner func(context.Context) ([]*rivertype.JobInsertResult, error)) ([]*rivertype.JobInsertResult, error) {
	for _, params := range manyParams {
		if !utils.Contains(params.Tags, archive_tag) {
			continue
		}

		var args RiverJobArgs
		if err := json.Unmarshal(params.EncodedArgs, &args); err != nil || args.Input.QueueId == uuid.Nil {
			continue
		}

		metadata := make(map[string]json.RawMessage)
		if len(params.Metadata) > 0 {
			if err := json.Unmarshal(params.Metadata, &metadata); err != nil {
				return nil, err
			}
		}
		if metadata == nil {
			metadata = make(map[string]json.RawMessage)
		}
		ganymedeMetadata, err := json.Marshal(archiveJobMetadata{
			QueueID:            args.Input.QueueId,
			RecoveryGeneration: args.Input.RecoveryGeneration,
		})
		if err != nil {
			return nil, err
		}
		metadata["ganymede"] = ganymedeMetadata
		encoded, err := json.Marshal(metadata)
		if err != nil {
			return nil, err
		}
		params.Metadata = encoded
	}
	return doInner(ctx)
}

func (m *ArchiveMiddleware) Work(ctx context.Context, job *rivertype.JobRow, doInner func(context.Context) error) error {
	if m.client == nil || !utils.Contains(job.Tags, archive_tag) {
		return doInner(ctx)
	}

	heartbeatCtx, stopHeartbeat := context.WithCancel(ctx)
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		m.runHeartbeat(heartbeatCtx, job.ID)
	}()

	err := doInner(ctx)
	stopHeartbeat()
	<-heartbeatDone

	// Remote cancellation is handled by the worker-specific cancellation path;
	// other cancellation causes can be a process shutdown and are checkpointed
	// for watchdog recovery.
	if ctx.Err() != nil && !errors.Is(context.Cause(ctx), rivertype.ErrJobCancelledRemotely) {
		m.updateProgress(job.ID, true)
	}
	return err
}

func (m *ArchiveMiddleware) runHeartbeat(ctx context.Context, jobID int64) {
	m.updateProgress(jobID, false)
	ticker := time.NewTicker(archiveHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.updateProgress(jobID, false)
		}
	}
}

func (m *ArchiveMiddleware) updateProgress(jobID int64, shutdownInterrupted bool) {
	for attempt := 1; attempt <= 3; attempt++ {
		// Keep the worst-case retry window below the worker's shutdown grace
		// period so a checkpoint attempt cannot itself prevent a clean stop.
		updateCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err := m.client.JobUpdate(updateCtx, jobID, &river.JobUpdateParams{Output: ArchiveProgressOutput{
			HeartbeatAt:         time.Now().UTC(),
			ShutdownInterrupted: shutdownInterrupted,
		}})
		cancel()
		if err == nil {
			return
		}
		log.Warn().Err(err).Int64("job_id", jobID).Int("attempt", attempt).Msg("failed to update archive job progress")
		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
}

var (
	_ rivertype.JobInsertMiddleware = (*ArchiveMiddleware)(nil)
	_ rivertype.WorkerMiddleware    = (*ArchiveMiddleware)(nil)
)
