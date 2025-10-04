package tests_shared

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/zibbp/ganymede/ent/queue"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/server"
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
	TestArchiveTimeout          = 300 * time.Second
)

// IsPlayableVideo checks if a video file is playable using ffprobe.
func IsPlayableVideo(path string) bool {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries",
		"stream=codec_name", "-of", "default=noprint_wrappers=1:nokey=1", path)
	err := cmd.Run()
	return err == nil
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
		runningJobsParams := river.NewJobListParams().States(rivertype.JobStateRunning).First(10000)
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
