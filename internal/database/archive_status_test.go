package database

import (
	"database/sql"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/tests/pgtest"
)

func TestArchiveStatusMigration(t *testing.T) {
	for _, workerFirst := range []bool{false, true} {
		name := "api-first"
		if workerFirst {
			name = "worker-first"
		}
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			dsn := pgtest.ConnectionString(t)
			db, err := sql.Open("pgx", dsn)
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
			require.NoError(t, client.Schema.Create(ctx))
			channel := client.Channel.Create().SetName("test").SetDisplayName("Test").SetImagePath("").SaveX(ctx)
			tests := []struct {
				name  string
				setup func(*ent.QueueCreate)
				want  utils.ArchiveStatus
			}{
				{"queued", func(q *ent.QueueCreate) {}, utils.ArchiveQueued},
				{"running", func(q *ent.QueueCreate) { q.SetTaskVideoDownload(utils.Running) }, utils.ArchiveRunning},
				{"finalizing", func(q *ent.QueueCreate) { q.SetTaskVideoDownload(utils.Success) }, utils.ArchiveFinalizing},
				{"failed", func(q *ent.QueueCreate) { q.SetTaskVideoConvert(utils.Failed) }, utils.ArchiveFailed},
				{"video-only", func(q *ent.QueueCreate) {
					q.SetArchiveChat(false).SetTaskVideoDownload(utils.Success).SetTaskVideoConvert(utils.Success).SetTaskVideoMove(utils.Success)
				}, utils.ArchiveCompleted},
				{"chat-failed", func(q *ent.QueueCreate) {
					q.SetTaskVideoDownload(utils.Success).SetTaskVideoConvert(utils.Success).SetTaskVideoMove(utils.Success).SetTaskChatDownload(utils.Failed)
				}, utils.ArchiveCompleted},
				{"chat-waiting", func(q *ent.QueueCreate) {
					q.SetTaskVideoDownload(utils.Success).SetTaskVideoConvert(utils.Success).SetTaskVideoMove(utils.Success)
				}, utils.ArchiveFinalizing},
				{"orphan", nil, utils.ArchiveFailed},
				{"already-completed", nil, utils.ArchiveCompleted},
			}
			var videos []*ent.Vod
			for _, tc := range tests {
				v := client.Vod.Create().SetChannel(channel).SetExtID(tc.name).SetTitle(tc.name).SetWebThumbnailPath("").SetVideoPath("").SaveX(ctx)
				videos = append(videos, v)
				if tc.setup != nil {
					q := client.Queue.Create().SetVod(v)
					tc.setup(q)
					q.SaveX(ctx)
				}
			}
			_, err = db.ExecContext(ctx, `ALTER TABLE vods DROP COLUMN status;
    ALTER TABLE vods ADD COLUMN processing boolean NOT NULL DEFAULT true;
    ALTER TABLE queues ADD COLUMN processing boolean NOT NULL DEFAULT true,
      ADD COLUMN video_processing boolean NOT NULL DEFAULT true, ADD COLUMN chat_processing boolean NOT NULL DEFAULT true;
    UPDATE vods SET processing = false WHERE ext_id = 'already-completed'`)
			require.NoError(t, err)
			for _, role := range []bool{workerFirst, !workerFirst} {
				store := NewDatabase(ctx, DatabaseConnectionInput{DBString: dsn, IsWorker: role})
				for i, tc := range tests {
					require.Equal(t, tc.want, store.Client.Vod.GetX(ctx, videos[i].ID).Status, tc.name)
				}
				var oldColumns int
				require.NoError(t, store.SQLDB.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name IN ('vods','queues') AND column_name IN ('processing','video_processing','chat_processing')`).Scan(&oldColumns))
				require.Zero(t, oldColumns)
				require.NoError(t, store.Client.Close())
				if store.ConnPool != nil {
					store.ConnPool.Close()
				}
			}
		})
	}
}

func TestArchiveStatusMigrationRollbackAndFreshDatabase(t *testing.T) {
	db := pgtest.Open(t)
	ctx := t.Context()
	require.NoError(t, migrateArchiveStatus(ctx, db), "fresh database")
	_, err := db.ExecContext(ctx, `CREATE TABLE vods (id uuid PRIMARY KEY, processing boolean NOT NULL)`)
	require.NoError(t, err)
	require.Error(t, migrateArchiveStatus(ctx, db), "missing queue schema must abort")
	var statusExists bool
	require.NoError(t, db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name='vods' AND column_name='status')`).Scan(&statusExists))
	require.False(t, statusExists, "failed backfill must roll back new column")
}
