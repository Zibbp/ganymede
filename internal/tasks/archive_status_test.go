package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
	tasks_shared "github.com/zibbp/ganymede/internal/tasks/shared"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/tests/pgtest"
)

// The probe writes through the supplied transaction, so rollback assertions test
// the same Ent/SQL boundary used by the River insertion client.
type statusProbeEnqueuer struct{ fail bool }

func (*statusProbeEnqueuer) Insert(context.Context, river.JobArgs, *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return nil, errors.New("nontransactional insertion")
}
func (e *statusProbeEnqueuer) InsertTx(ctx context.Context, tx *sql.Tx, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	if e.fail {
		return nil, errors.New("insertion failed")
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO status_probe(kind) VALUES($1)`, args.Kind())
	return &rivertype.JobInsertResult{}, err
}

func TestArchiveStatusTransactions(t *testing.T) {
	t.Setenv("CONFIG_DIR", t.TempDir())
	t.Setenv("DEBUG", "false")
	t.Setenv("TWITCH_CLIENT_ID", "test")
	t.Setenv("TWITCH_CLIENT_SECRET", "test")
	_, err := config.Init()
	require.NoError(t, err)
	db := pgtest.Open(t)
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	store := &database.Database{SQLDB: db, Client: client}
	ctx := context.WithValue(t.Context(), tasks_shared.StoreKey, store)
	probe := &statusProbeEnqueuer{}
	ctx = context.WithValue(ctx, tasks_shared.EnqueuerKey, probe)
	require.NoError(t, client.Schema.Create(ctx))
	_, err = db.ExecContext(ctx, `CREATE TABLE status_probe (kind text NOT NULL)`)
	require.NoError(t, err)
	ch := client.Channel.Create().SetName("test").SetDisplayName("Test").SetImagePath("").SaveX(ctx)
	v := client.Vod.Create().SetChannel(ch).SetExtID("test").SetTitle("Test").SetWebThumbnailPath("").SetVideoPath("").SetStatus(utils.ArchiveQueued).SaveX(ctx)
	q := client.Queue.Create().SetVod(v).SetArchiveChat(true).SetRenderChat(false).SaveX(ctx)
	input := ArchiveVideoInput{QueueId: q.ID}
	set := func(task utils.TaskName, status utils.TaskStatus) {
		t.Helper()
		require.NoError(t, setQueueStatus(ctx, client, QueueStatusInput{QueueId: q.ID, Task: task, Status: status}))
	}
	assertStatus := func(want utils.ArchiveStatus) { t.Helper(); require.Equal(t, want, client.Vod.GetX(ctx, v.ID).Status) }
	set(utils.TaskDownloadVideo, utils.Running)
	assertStatus(utils.ArchiveRunning)
	needsRecovery, err := archiveQueueStageNeedsRecovery(ctx, store, q.ID, string(utils.TaskDownloadVideo))
	require.NoError(t, err)
	require.True(t, needsRecovery)
	probe.fail = true
	err = setQueueStatusAndEnqueue(ctx, store, QueueStatusInput{QueueId: q.ID, Task: utils.TaskDownloadVideo, Status: utils.Success}, transactionalJob{Args: PostProcessVideoArgs{Input: input}})
	require.Error(t, err)
	require.Equal(t, utils.Running, client.Queue.GetX(ctx, q.ID).TaskVideoDownload)
	assertStatus(utils.ArchiveRunning)
	probe.fail = false
	require.NoError(t, setQueueStatusAndEnqueue(ctx, store, QueueStatusInput{QueueId: q.ID, Task: utils.TaskDownloadVideo, Status: utils.Success}, transactionalJob{Args: PostProcessVideoArgs{Input: input}}))
	assertStatus(utils.ArchiveFinalizing)
	set(utils.TaskPostProcessVideo, utils.Failed)
	assertStatus(utils.ArchiveFailed)
	set(utils.TaskPostProcessVideo, utils.Running)
	assertStatus(utils.ArchiveFinalizing)
	set(utils.TaskPostProcessVideo, utils.Success)
	set(utils.TaskMoveVideo, utils.Success)
	assertStatus(utils.ArchiveFinalizing)
	set(utils.TaskDownloadChat, utils.Failed)
	probe.fail = true
	require.Error(t, checkIfTasksAreDone(ctx, client, input))
	assertStatus(utils.ArchiveFinalizing)
	probe.fail = false
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() { results <- checkIfTasksAreDone(ctx, client, input) })
	}
	wg.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	assertStatus(utils.ArchiveCompleted)
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM status_probe WHERE kind=$1`, (UpdateVideoStorageUsage{}).Kind()).Scan(&count))
	require.Equal(t, 1, count, "concurrent completion enqueues follow-up exactly once")
	require.NoError(t, checkIfTasksAreDone(ctx, client, input))
	liveArgs, _ := json.Marshal(RiverJobArgs{Input: input})
	(&CustomErrorHandler{}).HandleError(ctx, &rivertype.JobRow{
		Kind: string(utils.TaskDownloadLiveVideo), Tags: []string{archive_tag},
		Attempt: 1, MaxAttempts: 2, EncodedArgs: liveArgs,
	}, rivertype.ErrJobCancelledRemotely)
	assertStatus(utils.ArchiveCompleted)

	// Retryable attempts must not leave a failed terminal status; remote terminal
	// cancellation must persist even with an already cancelled job context.
	set(utils.TaskDownloadVideo, utils.Running)
	args, _ := json.Marshal(RiverJobArgs{Input: input})
	job := &rivertype.JobRow{Kind: string(utils.TaskDownloadVideo), Tags: []string{archive_tag}, Attempt: 1, MaxAttempts: 2, EncodedArgs: args}
	handler := &CustomErrorHandler{}
	handler.HandleError(ctx, job, errors.New("temporary download error"))
	assertStatus(utils.ArchiveQueued)
	job.Attempt = 2
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	handler.HandleError(cancelled, job, rivertype.ErrJobCancelledRemotely)
	assertStatus(utils.ArchiveFailed)
	require.Equal(t, utils.Failed, client.Queue.GetX(ctx, q.ID).TaskVideoDownload)

	set(utils.TaskDownloadVideo, utils.Running)
	job.Attempt = 1
	handler.HandlePanic(ctx, job, "temporary panic", "test trace")
	assertStatus(utils.ArchiveQueued)
	job.Attempt = 2
	handler.HandlePanic(ctx, job, "terminal panic", "test trace")
	assertStatus(utils.ArchiveFailed)

	client.Queue.UpdateOneID(q.ID).SetTaskVideoDownload(utils.Running).SaveX(ctx)
	client.Vod.UpdateOneID(v.ID).SetStatus(utils.ArchiveCompleted).SaveX(ctx)
	needsRecovery, err = archiveQueueStageNeedsRecovery(ctx, store, q.ID, string(utils.TaskDownloadVideo))
	require.NoError(t, err)
	require.False(t, needsRecovery, "terminal archives are not restarted by the existing watchdog")
}
