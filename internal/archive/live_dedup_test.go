package archive

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/queue"
	tasks_client "github.com/zibbp/ganymede/internal/tasks/client"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
	"github.com/zibbp/ganymede/tests/pgtest"
)

func TestConcurrentLiveArchiveCreation(t *testing.T) {
	ctx := t.Context()
	store := database.NewDatabase(ctx, database.DatabaseConnectionInput{DBString: pgtest.ConnectionString(t), IsWorker: true})
	t.Cleanup(func() { store.Close() })
	rc, err := tasks_client.NewRiverClient(tasks_client.RiverClientInput{Database: store})
	require.NoError(t, err)
	services := []*Service{
		{Store: store, VodService: &vod.Service{}, QueueService: &queue.Service{}, RiverClient: rc},
		{Store: store, VodService: &vod.Service{}, QueueService: &queue.Service{}, RiverClient: rc},
	}
	channel := store.Client.Channel.Create().SetName("test").SetDisplayName("Test").SetImagePath("").SaveX(ctx)
	dto := func() vod.Vod {
		return vod.Vod{ID: uuid.New(), ExtID: "stream", ExtStreamID: "stream", Platform: utils.PlatformTwitch,
			Type: utils.Live, Title: "Test", Status: utils.ArchiveQueued, WebThumbnailPath: "", VideoPath: ""}
	}
	options := queue.Queue{LiveArchive: true}

	// Hold the channel lock until both service instances have competing transactions.
	lock, err := store.SQLDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer lock.Rollback()
	_, err = lock.ExecContext(ctx, `SELECT id FROM channels WHERE id=$1 FOR UPDATE`, channel.ID)
	require.NoError(t, err)
	var wg sync.WaitGroup
	outcomes := make(chan error, 8)
	for i := range 8 {
		wg.Go(func() {
			_, err := services[i%len(services)].createArchiveRecordsAndEnqueue(ctx, dto(), channel.ID, options)
			outcomes <- err
		})
	}
	require.Eventually(t, func() bool {
		var waiting int
		err := store.SQLDB.QueryRowContext(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock'`).Scan(&waiting)
		return err == nil && waiting >= 2
	}, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, lock.Commit())
	wg.Wait()
	close(outcomes)
	created, duplicates := 0, 0
	for err := range outcomes {
		switch {
		case err == nil:
			created++
		case errors.Is(err, ErrActiveLiveArchive):
			duplicates++
		default:
			require.NoError(t, err)
		}
	}
	require.Equal(t, 1, created)
	require.Equal(t, 7, duplicates)
	require.Equal(t, 1, store.Client.Vod.Query().CountX(ctx))
	require.Equal(t, 1, store.Client.Queue.Query().CountX(ctx))
	var jobs int
	require.NoError(t, store.SQLDB.QueryRowContext(ctx, `SELECT count(*) FROM river_job`).Scan(&jobs))
	require.Equal(t, 1, jobs)

	video := store.Client.Vod.Query().OnlyX(ctx)
	// ExtID can change to the published VOD ID; ExtStreamID remains the capture identity.
	store.Client.Vod.UpdateOneID(video.ID).SetExtID("published-vod").ExecX(ctx)
	for _, status := range utils.ActiveArchiveStatuses() {
		store.Client.Vod.UpdateOneID(video.ID).SetStatus(status).ExecX(ctx)
		_, err := services[0].createArchiveRecordsAndEnqueue(ctx, dto(), channel.ID, options)
		require.ErrorIs(t, err, ErrActiveLiveArchive, string(status))
	}
	for _, status := range []utils.ArchiveStatus{utils.ArchiveCompleted, utils.ArchiveFailed} {
		store.Client.Vod.UpdateOneID(video.ID).SetStatus(status).ExecX(ctx)
		result, err := services[0].createArchiveRecordsAndEnqueue(ctx, dto(), channel.ID, options)
		require.NoError(t, err, string(status))
		video = result.Video
	}
	otherStream := dto()
	otherStream.ExtID, otherStream.ExtStreamID = "other-stream", "other-stream"
	_, err = services[0].createArchiveRecordsAndEnqueue(ctx, otherStream, channel.ID, options)
	require.NoError(t, err, "different streams may be archived")
	otherPlatform := dto()
	otherPlatform.Platform = utils.PlatformYoutube
	_, err = services[0].createArchiveRecordsAndEnqueue(ctx, otherPlatform, channel.ID, options)
	require.NoError(t, err, "stream identity includes the platform")
}
