package queue

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/internal/database"
	tasks_client "github.com/zibbp/ganymede/internal/tasks/client"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/tests/pgtest"
)

func TestRestartArchiveStatus(t *testing.T) {
	ctx := t.Context()
	store := database.NewDatabase(ctx, database.DatabaseConnectionInput{DBString: pgtest.ConnectionString(t), IsWorker: true})
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	rc, err := tasks_client.NewRiverClient(tasks_client.RiverClientInput{Database: store})
	require.NoError(t, err)
	service := &Service{Store: store, RiverClient: rc}
	channel := store.Client.Channel.Create().SetName("test").SetDisplayName("Test").SetImagePath("").SaveX(ctx)

	for _, tc := range []struct {
		name          string
		chat          bool
		before, after utils.ArchiveStatus
	}{
		{name: "failed video", before: utils.ArchiveFailed, after: utils.ArchiveQueued},
		{name: "completed chat", chat: true, before: utils.ArchiveCompleted, after: utils.ArchiveFinalizing},
	} {
		t.Run(tc.name, func(t *testing.T) {
			video := store.Client.Vod.Create().SetChannel(channel).SetExtID(tc.name).SetTitle(tc.name).
				SetWebThumbnailPath("").SetVideoPath("").SetStatus(tc.before).SaveX(ctx)
			q := store.Client.Queue.Create().SetVod(video).SetArchiveChat(tc.chat).SetOnHold(true).
				SetTaskVideoDownload(utils.Success).SetTaskVideoConvert(utils.Success).SetTaskVideoMove(utils.Success).
				SetTaskChatDownload(utils.Success).SetTaskChatMove(utils.Success).SaveX(ctx)
			task := utils.TaskDownloadVideo
			if tc.chat {
				task = utils.TaskDownloadChat
			} else {
				store.Client.Queue.UpdateOne(q).SetTaskVideoDownload(utils.Failed).ExecX(ctx)
			}
			input := StartQueueTaskInput{QueueId: q.ID, TaskName: string(task), Continue: true}

			// Reject insertion after the reset to exercise the shared transaction.
			_, err := store.SQLDB.ExecContext(ctx, `ALTER TABLE river_job ADD CONSTRAINT reject_restart CHECK (kind <> '`+string(task)+`')`)
			require.NoError(t, err)
			_, err = service.RestartQueueTask(ctx, input)
			require.Error(t, err)
			require.Equal(t, tc.before, store.Client.Vod.GetX(ctx, video.ID).Status)
			require.True(t, store.Client.Queue.GetX(ctx, q.ID).OnHold)
			_, err = store.SQLDB.ExecContext(ctx, `ALTER TABLE river_job DROP CONSTRAINT reject_restart`)
			require.NoError(t, err)

			result, err := service.RestartQueueTask(ctx, input)
			require.NoError(t, err)
			require.Positive(t, result.JobID)
			require.Equal(t, tc.after, store.Client.Vod.GetX(ctx, video.ID).Status)
			updated := store.Client.Queue.GetX(ctx, q.ID)
			require.False(t, updated.OnHold)
			require.Equal(t, utils.Pending, restartStageStatus(updated, task))
			_, err = service.RestartQueueTask(ctx, input)
			require.ErrorIs(t, err, ErrRestartConflict)
		})
	}
}
