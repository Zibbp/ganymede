package vod_test

import (
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/database"
	queues "github.com/zibbp/ganymede/internal/queue"
	"github.com/zibbp/ganymede/internal/utils"
	vods "github.com/zibbp/ganymede/internal/vod"
	"github.com/zibbp/ganymede/tests/pgtest"
	"net/http/httptest"
	"testing"
)

func TestArchiveStatusFilters(t *testing.T) {
	ctx := t.Context()
	db := pgtest.Open(t)
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	require.NoError(t, client.Schema.Create(ctx))
	store := &database.Database{SQLDB: db, Client: client}
	channel := client.Channel.Create().SetName("test").SetDisplayName("Test").SetImagePath("").SaveX(ctx)
	for _, status := range []utils.ArchiveStatus{utils.ArchiveCompleted, utils.ArchiveFailed, utils.ArchiveRunning} {
		v := client.Vod.Create().SetChannel(channel).SetExtID(string(status)).SetTitle("Test").SetWebThumbnailPath("").SetVideoPath("").SetStatus(status).SaveX(ctx)
		client.Queue.Create().SetVod(v).SaveX(ctx)
	}
	c := echo.New().NewContext(httptest.NewRequest("GET", "/", nil), httptest.NewRecorder())
	service := vods.NewService(store, nil, nil)
	for _, filter := range [][]utils.ArchiveStatus{{utils.ArchiveCompleted}, {utils.ArchiveFailed, utils.ArchiveRunning}, nil} {
		result, err := service.GetVodsPagination(c, 1, 0, channel.ID, nil, uuid.Nil, filter, "", "")
		require.NoError(t, err)
		count := len(filter)
		if filter == nil {
			count = 3
		}
		require.EqualValues(t, count, result.TotalCount)
		require.EqualValues(t, count, result.Pages)
		require.Len(t, result.Data, 1)
	}
	queueService := &queues.Service{Store: store}
	result, err := queueService.GetQueueItemsFilter(c, []utils.ArchiveStatus{utils.ArchiveFailed})
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, utils.ArchiveFailed, result[0].Edges.Vod.Status)
}
