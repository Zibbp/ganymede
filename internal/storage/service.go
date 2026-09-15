package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/storagefinding"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/tasks"
	tasks_client "github.com/zibbp/ganymede/internal/tasks/client"
)

type Service struct {
	Store       *database.Database
	RiverClient *tasks_client.RiverClient
	Platform    platform.Platform
}

func NewService(store *database.Database, riverClient *tasks_client.RiverClient, platform platform.Platform) *Service {
	return &Service{Store: store, RiverClient: riverClient, Platform: platform}
}

type ScanState string

const (
	ScanStateIdle    ScanState = "idle"
	ScanStateQueued  ScanState = "queued"
	ScanStateRunning ScanState = "running"
)

// ScanStatus describes the most recent reconciliation job.
type ScanStatus struct {
	State           ScanState  `json:"state"`
	StartedAt       *time.Time `json:"started_at"`
	LastCompletedAt *time.Time `json:"last_completed_at"`
	LastFailedAt    *time.Time `json:"last_failed_at"`
}

type StorageFinding struct {
	ID           uuid.UUID `json:"id"`
	Kind         string    `json:"kind"`
	Path         string    `json:"path"`
	RelativePath string    `json:"relative_path"`
	SizeBytes    int64     `json:"size_bytes"`
	DetectedAt   time.Time `json:"detected_at"`
}

type ListFindingsResponse struct {
	VideosDirectory string           `json:"videos_directory"`
	Scan            ScanStatus       `json:"scan"`
	Findings        []StorageFinding `json:"findings"`
}

type ActionFailure struct {
	ID    uuid.UUID `json:"id"`
	Path  string    `json:"path"`
	Error string    `json:"error"`
}

// ActionResult reports what happened to each finding of a batch.
type ActionResult struct {
	Done   []uuid.UUID     `json:"done"`
	Failed []ActionFailure `json:"failed"`
}

// ListFindings returns the stored findings and the state of the reconciliation job.
func (s *Service) ListFindings(ctx context.Context) (ListFindingsResponse, error) {
	videosDir := VideosDirectory()
	resp := ListFindingsResponse{
		VideosDirectory: videosDir,
		Findings:        []StorageFinding{},
	}

	rows, err := s.Store.Client.StorageFinding.Query().Order(ent.Asc(storagefinding.FieldPath)).All(ctx)
	if err != nil {
		return resp, fmt.Errorf("error getting storage findings: %w", err)
	}

	for _, row := range rows {
		resp.Findings = append(resp.Findings, toStorageFinding(row, videosDir))
	}

	status, err := s.scanStatus(ctx)
	if err != nil {
		return resp, err
	}
	resp.Scan = status

	return resp, nil
}

func toStorageFinding(row *ent.StorageFinding, videosDir string) StorageFinding {
	relativePath, err := filepath.Rel(videosDir, row.Path)
	if err != nil {
		relativePath = row.Path
	}
	return StorageFinding{
		ID:           row.ID,
		Kind:         string(row.Kind),
		Path:         row.Path,
		RelativePath: relativePath,
		SizeBytes:    row.SizeBytes,
		DetectedAt:   row.DetectedAt,
	}
}

// scanStatus derives the state of the reconciliation from its most recent job.
func (s *Service) scanStatus(ctx context.Context) (ScanStatus, error) {
	status := ScanStatus{State: ScanStateIdle}

	params := river.NewJobListParams().Kinds(tasks.TaskReconcileStorage).OrderBy(river.JobListOrderByTime, river.SortOrderDesc).First(10)
	result, err := s.RiverClient.JobList(ctx, params)
	if err != nil {
		return status, fmt.Errorf("error listing reconciliation jobs: %w", err)
	}

	for _, job := range result.Jobs {
		switch job.State {
		case rivertype.JobStateRunning:
			if status.State == ScanStateIdle {
				status.State = ScanStateRunning
				status.StartedAt = job.AttemptedAt
			}
		case rivertype.JobStateAvailable, rivertype.JobStateScheduled, rivertype.JobStatePending, rivertype.JobStateRetryable:
			if status.State == ScanStateIdle {
				status.State = ScanStateQueued
			}
		case rivertype.JobStateCompleted:
			if status.LastCompletedAt == nil {
				status.LastCompletedAt = job.FinalizedAt
			}
		case rivertype.JobStateDiscarded, rivertype.JobStateCancelled:
			if status.LastFailedAt == nil {
				status.LastFailedAt = job.FinalizedAt
			}
		}
	}

	return status, nil
}

// DeleteFindings deletes the directories of the supplied findings. Each one is verified again
// right before it is removed.
func (s *Service) DeleteFindings(ctx context.Context, ids []uuid.UUID, username string) (ActionResult, error) {
	result := ActionResult{Done: []uuid.UUID{}, Failed: []ActionFailure{}}
	videosDir := VideosDirectory()

	// Everything is removed through a handle on the videos directory, so a path that leaves it,
	// even through a symlink swapped in after the check, is refused by the operating system.
	root, err := os.OpenRoot(videosDir)
	if err != nil {
		return result, fmt.Errorf("error opening the videos directory: %w", err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			log.Warn().Err(err).Msg("error closing the videos directory")
		}
	}()

	for _, id := range ids {
		finding, err := s.claimFinding(ctx, id)
		if err != nil {
			result.Failed = append(result.Failed, ActionFailure{ID: id, Error: findingLookupError(err)})
			continue
		}

		if err := VerifyFinding(ctx, s.Store, videosDir, finding.Path); err != nil {
			result.Failed = append(result.Failed, ActionFailure{ID: id, Path: finding.Path, Error: err.Error()})
			continue
		}

		log.Info().Str("user", username).Msgf("deleting orphaned directory %s", finding.Path)

		if err := removeInsideVideosDir(root, videosDir, finding.Path); err != nil {
			result.Failed = append(result.Failed, ActionFailure{ID: id, Path: finding.Path, Error: fmt.Sprintf("error deleting directory: %v", err)})
			continue
		}

		result.Done = append(result.Done, id)
	}

	return result, nil
}

// ImportFindings turns the directories of the supplied findings back into videos. Each one is
// verified again right before it is imported.
func (s *Service) ImportFindings(ctx context.Context, ids []uuid.UUID, username string) (ActionResult, error) {
	result := ActionResult{Done: []uuid.UUID{}, Failed: []ActionFailure{}}
	videosDir := VideosDirectory()

	for _, id := range ids {
		finding, err := s.claimFinding(ctx, id)
		if err != nil {
			result.Failed = append(result.Failed, ActionFailure{ID: id, Error: findingLookupError(err)})
			continue
		}

		if err := VerifyFinding(ctx, s.Store, videosDir, finding.Path); err != nil {
			result.Failed = append(result.Failed, ActionFailure{ID: id, Path: finding.Path, Error: err.Error()})
			continue
		}

		log.Info().Str("user", username).Msgf("importing orphaned directory %s", finding.Path)

		if err := s.importDirectory(ctx, videosDir, finding.Path); err != nil {
			result.Failed = append(result.Failed, ActionFailure{ID: id, Path: finding.Path, Error: err.Error()})
			continue
		}

		result.Done = append(result.Done, id)
	}

	return result, nil
}

// claimFinding removes the finding and returns it, so that two administrators acting on the
// same directory at the same time cannot both get past this point. A finding whose action then
// fails is reported back and found again by the next scan.
func (s *Service) claimFinding(ctx context.Context, id uuid.UUID) (*ent.StorageFinding, error) {
	finding, err := s.Store.Client.StorageFinding.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// The delete is what claims it: only one caller can remove the row.
	claimed, err := s.Store.Client.StorageFinding.Delete().Where(storagefinding.ID(id)).Exec(ctx)
	if err != nil {
		return nil, err
	}
	if claimed == 0 {
		return nil, &ent.NotFoundError{}
	}

	return finding, nil
}

// removeInsideVideosDir deletes a directory through the handle on the videos directory.
func removeInsideVideosDir(root *os.Root, videosDir string, path string) error {
	relativePath, err := filepath.Rel(videosDir, path)
	if err != nil {
		return err
	}
	return root.RemoveAll(relativePath)
}

// findingLookupError keeps a database failure from being reported as a missing finding.
func findingLookupError(err error) string {
	if ent.IsNotFound(err) {
		return "finding not found"
	}
	log.Error().Err(err).Msg("error getting storage finding")
	return "error getting the finding"
}

// ReconcileAndStore runs the reconciliation and replaces the stored findings with the result.
// Findings that are still there keep their id and detection time, findings that are gone are
// removed.
func ReconcileAndStore(ctx context.Context, store *database.Database) (int, error) {
	references, err := ReferencesFromDatabase(ctx, store)
	if err != nil {
		return 0, err
	}

	findings, err := Reconcile(references)
	if err != nil {
		return 0, err
	}

	tx, err := store.Client.Tx(ctx)
	if err != nil {
		return 0, fmt.Errorf("error starting transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !isTxDone(err) {
			log.Error().Err(err).Msg("error rolling back storage findings")
		}
	}()

	seen := make([]string, 0, len(findings))
	for _, finding := range findings {
		seen = append(seen, finding.Path)
		err := tx.StorageFinding.Create().
			SetKind(storagefinding.KindOrphanedDirectory).
			SetPath(finding.Path).
			SetSizeBytes(finding.SizeBytes).
			OnConflictColumns(storagefinding.FieldPath).
			UpdateSizeBytes().
			Exec(ctx)
		if err != nil {
			return 0, fmt.Errorf("error storing finding %s: %w", finding.Path, err)
		}
	}

	if _, err := tx.StorageFinding.Delete().Where(storagefinding.PathNotIn(seen...)).Exec(ctx); err != nil {
		return 0, fmt.Errorf("error removing stale findings: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("error committing storage findings: %w", err)
	}

	return len(findings), nil
}

func isTxDone(err error) bool {
	return errors.Is(err, sql.ErrTxDone)
}
