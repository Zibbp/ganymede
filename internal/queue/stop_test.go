package queue

import (
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/tasks"
	tasks_client "github.com/zibbp/ganymede/internal/tasks/client"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/tests/pgtest"
)

func TestStopLiveRecordingPreservesFinalization(t *testing.T) {
	ctx := t.Context()
	store := database.NewDatabase(ctx, database.DatabaseConnectionInput{DBString: pgtest.ConnectionString(t), IsWorker: true})
	t.Cleanup(func() { require.NoError(t, store.Client.Close()) })
	rc, err := tasks_client.NewRiverClient(tasks_client.RiverClientInput{Database: store})
	require.NoError(t, err)
	service := &Service{Store: store, RiverClient: rc}
	channel := store.Client.Channel.Create().SetName("test").SetDisplayName("Test").SetImagePath("").SaveX(ctx)
	video := store.Client.Vod.Create().SetChannel(channel).SetExtID("test").SetTitle("Test").
		SetWebThumbnailPath("").SetVideoPath("kept.mp4").SetStatus(utils.ArchiveRunning).SaveX(ctx)
	q := store.Client.Queue.Create().SetVod(video).SetLiveArchive(true).SetTaskVideoDownload(utils.Running).SaveX(ctx)
	input := tasks.ArchiveVideoInput{QueueId: q.ID}
	insert := func(args river.JobArgs, state rivertype.JobState) int64 {
		t.Helper()
		result, err := rc.Insert(ctx, args, nil)
		require.NoError(t, err)
		_, err = store.SQLDB.ExecContext(ctx, `UPDATE river_job SET state=$1 WHERE id=$2`, state, result.Job.ID)
		require.NoError(t, err)
		return result.Job.ID
	}
	capture := insert(tasks.DownloadLiveVideoArgs{Continue: true, Input: input}, rivertype.JobStateRunning)
	chat := insert(tasks.DownloadLiveChatArgs{Continue: true, Input: input}, rivertype.JobStateRunning)
	postProcess := insert(tasks.PostProcessVideoArgs{Continue: true, Input: input}, rivertype.JobStateAvailable)
	otherCapture := insert(tasks.DownloadLiveVideoArgs{Continue: true, Input: tasks.ArchiveVideoInput{QueueId: uuid.New()}}, rivertype.JobStateRunning)
	assertCancelled := func(id int64, want bool) {
		t.Helper()
		var attempted bool
		require.NoError(t, store.SQLDB.QueryRowContext(ctx, `SELECT metadata ? 'cancel_attempted_at' FROM river_job WHERE id=$1`, id).Scan(&attempted))
		require.Equal(t, want, attempted)
	}

	require.NoError(t, service.StopQueueItem(ctx, q.ID))
	assertCancelled(capture, true)
	assertCancelled(chat, false) // The capture worker stops chat before handing off.
	assertCancelled(postProcess, false)
	assertCancelled(otherCapture, false)
	require.Equal(t, "kept.mp4", store.Client.Vod.GetX(ctx, video.ID).VideoPath)
	require.Equal(t, utils.ArchiveRunning, store.Client.Vod.GetX(ctx, video.ID).Status, "the worker owns lifecycle transitions")

	// Simulate the capture worker finishing before a delayed second stop arrives.
	_, err = store.SQLDB.ExecContext(ctx, `UPDATE river_job SET state='completed', finalized_at=now() WHERE id=$1`, capture)
	require.NoError(t, err)
	store.Client.Queue.UpdateOneID(q.ID).SetTaskVideoDownload(utils.Success).ExecX(ctx)
	require.NoError(t, service.StopQueueItem(ctx, q.ID))
	assertCancelled(chat, false)
	assertCancelled(postProcess, false)
	assertCancelled(otherCapture, false)

	// Non-live queue cancellation still stops all of its tasks.
	store.Client.Queue.UpdateOneID(q.ID).SetLiveArchive(false).ExecX(ctx)
	require.NoError(t, service.StopQueueItem(ctx, q.ID))
	assertCancelled(chat, true)
	assertCancelled(postProcess, true)
	assertCancelled(otherCapture, false)
}
