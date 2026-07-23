package tests_shared

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/queue"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/server"
	"github.com/zibbp/ganymede/internal/utils"
)

var (
	TestTwitchVideoId1          = "1989753443"
	TestTwitchVideoChannelName1 = "sodapoppin"
	TestTwitchClipId1           = "SarcasticDarkPanCoolCat-rgyYByzzfGqIwbWd"
	TestTwitchClipChannelName1  = "sodapoppin"
	TestTwitchVideoId2          = "2325332129"
	TestTwitchVideoChannelName2 = "datmodz"
	TestTwitchClipId2           = "CleverPolishedSwordPanicBasket-qmNOWICct4rtR_wX"
	TestTwitchClipChannelName2  = "datmodz"
	TestArchiveTimeout          = 500 * time.Second
)

// IsPlayableVideo checks if a video file is playable using ffprobe.
func IsPlayableVideo(path string) bool {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries",
		"stream=codec_name", "-of", "default=noprint_wrappers=1:nokey=1", path)
	err := cmd.Run()
	return err == nil
}

// WaitForProcessExit verifies that a downloader or capture process containing
// match in its command line did not become orphaned after its worker died.
func WaitForProcessExit(t *testing.T, match string, timeout time.Duration) {
	t.Helper()
	if match == "" {
		t.Fatal("process match must not be empty")
	}

	// Bracketing the first character prevents pgrep from matching its own
	// command line while preserving the target match.
	pattern := "[" + match[:1] + "]" + match[1:]
	deadline := time.Now().Add(timeout)
	for {
		err := exec.Command("pgrep", "-f", pattern).Run()
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
				return
			}
			t.Fatalf("check process %q: %v", match, err)
		}
		if time.Now().After(deadline) {
			t.Fatalf("process matching %q remained after worker crash", match)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// WaitForArchiveCompletion waits until the queue item is done processing and no running jobs remain.
func WaitForArchiveCompletion(t *testing.T, app *server.Application, videoId uuid.UUID, timeout time.Duration) {
	startTime := time.Now()
	for {
		if time.Since(startTime) >= timeout {
			t.Fatalf("Timeout reached while waiting for video to be archived")
		}

		q, err := app.Database.Client.Queue.Query().Where(queue.HasVodWith(vod.ID(videoId))).Only(context.Background())
		if err != nil {
			t.Fatalf("Error querying queue item: %v", err)
		}
		runningJobsParams := river.NewJobListParams().States(rivertype.JobStateRunning).First(500)
		runningJobs, err := app.RiverClient.JobList(context.Background(), runningJobsParams)
		if err != nil {
			t.Fatalf("Error listing running jobs: %v", err)
		}

		if !q.Processing && len(runningJobs.Jobs) == 0 {
			break
		}

		time.Sleep(10 * time.Second)
	}
}

// WaitForArchiveCompletionAfterCrash waits for the application queue rather
// than for every historical River row to leave the running state. A worker
// killed with SIGKILL leaves its claimed River row behind until River's
// rescuer performs normal lifecycle cleanup, even though the watchdog has
// already completed the archive through a replacement or downstream job.
func WaitForArchiveCompletionAfterCrash(t *testing.T, app *server.Application, videoID uuid.UUID, timeout time.Duration) *ent.Queue {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		q, err := app.Database.Client.Queue.Query().
			Where(queue.HasVodWith(vod.ID(videoID))).
			Only(t.Context())
		if err != nil {
			t.Fatalf("query crashed archive queue: %v", err)
		}
		if !q.Processing {
			return q
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for crashed archive %s to complete", videoID)
		}
		time.Sleep(time.Second)
	}
}

// WaitForArchiveMetadataFinalization waits for ancillary jobs queued at
// archive completion (notably storage accounting) without waiting for the
// crashed River execution's historical row to be rescued.
func WaitForArchiveMetadataFinalization(t *testing.T, app *server.Application, videoID uuid.UUID, timeout time.Duration) *ent.Vod {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		archivedVod, err := app.Database.Client.Vod.Query().
			Where(vod.ID(videoID)).
			WithChapters().
			Only(t.Context())
		if err != nil {
			t.Fatalf("query finalized archive metadata: %v", err)
		}
		if archivedVod.StorageSizeBytes > 0 && len(archivedVod.Edges.Chapters) > 0 {
			return archivedVod
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for archive %s metadata finalization", videoID)
		}
		time.Sleep(time.Second)
	}
}

// WaitForRunningVideoDownload waits until the queue is running and the media
// file is non-empty, proving that a worker crash occurs during real capture
// rather than before the external downloader starts.
func WaitForRunningVideoDownload(t *testing.T, app *server.Application, queueID uuid.UUID, mediaPath string, minimumBytes int64, timeout time.Duration) int64 {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		q, err := app.Database.Client.Queue.Get(t.Context(), queueID)
		if err != nil {
			t.Fatalf("query archive queue: %v", err)
		}
		if q.TaskVideoDownload == utils.Running {
			if info, err := os.Stat(mediaPath); err == nil && info.Size() >= minimumBytes {
				return info.Size()
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for a running video download with media at %s", mediaPath)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// FindArchiveJob returns the River job of the requested kind and state that
// belongs to queueID. Test databases are isolated, but matching encoded args
// keeps the assertion valid if unrelated periodic work is present.
func FindArchiveJob(t *testing.T, app *server.Application, queueID uuid.UUID, kind string, states ...rivertype.JobState) *rivertype.JobRow {
	t.Helper()

	params := river.NewJobListParams().States(states...).Kinds(kind).First(500)
	result, err := app.RiverClient.JobList(t.Context(), params)
	if err != nil {
		t.Fatalf("list River archive jobs: %v", err)
	}
	for _, job := range result.Jobs {
		var args struct {
			Input struct {
				QueueID uuid.UUID `json:"queue_id"`
			} `json:"input"`
		}
		if err := json.Unmarshal(job.EncodedArgs, &args); err != nil {
			t.Fatalf("decode River job %d args: %v", job.ID, err)
		}
		if args.Input.QueueID == queueID {
			return job
		}
	}
	return nil
}

// WaitForCompletedArchiveRecovery proves that the watchdog inserted and
// completed a replacement generation for a crashed non-live archive job.
func WaitForCompletedArchiveRecovery(t *testing.T, app *server.Application, queueID uuid.UUID, kind string, timeout time.Duration) *rivertype.JobRow {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		params := river.NewJobListParams().
			States(rivertype.JobStateCompleted).
			Kinds(kind).
			First(500)
		result, err := app.RiverClient.JobList(t.Context(), params)
		if err != nil {
			t.Fatalf("list completed River archive jobs: %v", err)
		}
		for _, job := range result.Jobs {
			var metadata struct {
				Ganymede struct {
					QueueID            uuid.UUID `json:"queue_id"`
					RecoveryGeneration int       `json:"recovery_generation"`
				} `json:"ganymede"`
			}
			if err := json.Unmarshal(job.Metadata, &metadata); err != nil {
				t.Fatalf("decode River job %d metadata: %v", job.ID, err)
			}
			if metadata.Ganymede.QueueID == queueID && metadata.Ganymede.RecoveryGeneration > 0 {
				return job
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for completed recovery job for queue %s", queueID)
		}
		time.Sleep(time.Second)
	}
}

func WaitForArchiveJobCancellation(t *testing.T, app *server.Application, jobID int64, timeout time.Duration) time.Time {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		job, err := app.RiverClient.Client.JobGet(t.Context(), jobID)
		if err != nil {
			t.Fatalf("get River archive job %d: %v", jobID, err)
		}
		var metadata struct {
			CancelAttemptedAt time.Time `json:"cancel_attempted_at"`
		}
		if err := json.Unmarshal(job.Metadata, &metadata); err != nil {
			t.Fatalf("decode River job %d cancellation metadata: %v", jobID, err)
		}
		if !metadata.CancelAttemptedAt.IsZero() {
			return metadata.CancelAttemptedAt
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for River job %d cancellation", jobID)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
