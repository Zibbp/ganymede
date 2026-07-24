package database_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/blockedvideos"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/server"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/tests"
	tests_shared "github.com/zibbp/ganymede/tests/shared"
)

type DatabaseTest struct {
	App *server.Application
}

type transactionProbeArgs struct {
	ProbeID string `json:"probe_id"`
}

func (transactionProbeArgs) Kind() string { return "database_transaction_probe" }

func TestWithTxCoordinatesEntAndRiver(t *testing.T) {
	app, err := tests.Setup(t)
	require.NoError(t, err)

	ctx := context.Background()
	rollbackID := "rollback-probe"
	rollbackErr := errors.New("force rollback")
	err = app.Database.WithTx(ctx, func(txClient *ent.Client, tx *sql.Tx) error {
		if _, err := txClient.BlockedVideos.Create().SetID(rollbackID).Save(ctx); err != nil {
			return err
		}
		if _, err := app.RiverClient.InsertTx(ctx, tx, transactionProbeArgs{ProbeID: rollbackID}, &river.InsertOpts{ScheduledAt: time.Now().Add(time.Hour)}); err != nil {
			return err
		}
		return rollbackErr
	})
	require.ErrorIs(t, err, rollbackErr)
	assertTransactionProbeState(t, app, rollbackID, false)

	commitID := "commit-probe"
	err = app.Database.WithTx(ctx, func(txClient *ent.Client, tx *sql.Tx) error {
		if _, err := txClient.BlockedVideos.Create().SetID(commitID).Save(ctx); err != nil {
			return err
		}
		_, err := app.RiverClient.InsertTx(ctx, tx, transactionProbeArgs{ProbeID: commitID}, &river.InsertOpts{ScheduledAt: time.Now().Add(time.Hour)})
		return err
	})
	require.NoError(t, err)
	assertTransactionProbeState(t, app, commitID, true)
}

func assertTransactionProbeState(t *testing.T, app *server.Application, probeID string, want bool) {
	t.Helper()
	ctx := context.Background()
	exists, err := app.Database.Client.BlockedVideos.Query().Where(blockedvideos.ID(probeID)).Exist(ctx)
	require.NoError(t, err)
	require.Equal(t, want, exists)

	var count int
	err = app.Database.SQLDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM river_job WHERE kind = $1 AND args ->> 'probe_id' = $2`,
		(transactionProbeArgs{}).Kind(), probeID,
	).Scan(&count)
	require.NoError(t, err)
	if want {
		require.Equal(t, 1, count)
	} else {
		require.Zero(t, count)
	}
}

func TestDatabase(t *testing.T) {
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	databaseTest := DatabaseTest{App: app}

	t.Run("TestVideosDirMigrate", databaseTest.TestVideosDirMigrate)
}

// TestVideosDirMigrate tests the VideosDirMigrate function
func (s *DatabaseTest) TestVideosDirMigrate(t *testing.T) {

	// Archive a video to test the migration
	_, err := s.App.ArchiveService.ArchiveVideo(context.Background(), archive.ArchiveVideoInput{
		VideoId:     tests_shared.TestTwitchVideoId1,
		Quality:     utils.R480,
		ArchiveChat: true,
		RenderChat:  true,
	})
	assert.NoError(t, err)

	// Assert video was created
	v, err := s.App.Database.Client.Vod.Query().Where(vod.ExtID(tests_shared.TestTwitchVideoId1)).WithChapters().Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	tests_shared.WaitForArchiveCompletion(t, s.App, v.ID, tests_shared.TestArchiveTimeout)

	// Migrate the videos directory
	newVideosDir := "/new/videos/dir"
	err = s.App.Database.VideosDirMigrate(context.Background(), newVideosDir)
	assert.NoError(t, err)

	// Fetch the video again
	v, err = s.App.Database.Client.Vod.Query().Where(vod.ExtID(tests_shared.TestTwitchVideoId1)).WithChannel().WithChapters().Only(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	// Assert all *Path fields have been updated to the new directory
	val := reflect.ValueOf(v).Elem()
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		if strings.HasSuffix(field.Name, "Path") && field.Type.Kind() == reflect.String && !strings.Contains(field.Name, "Tmp") {
			pathValue := val.Field(i).String()
			if pathValue == "" {
				continue
			}
			assert.Truef(t, strings.HasPrefix(pathValue, newVideosDir),
				"Field %s was not updated: got %s, want prefix %s", field.Name, pathValue, newVideosDir)
		}
	}

	// Assert sprite thumbnails paths have been updated
	if v.SpriteThumbnailsEnabled && len(v.SpriteThumbnailsImages) > 0 {
		for _, thumb := range v.SpriteThumbnailsImages {
			assert.Truef(t, strings.HasPrefix(thumb, newVideosDir),
				"Sprite thumbnail path was not updated: got %s, want prefix %s", thumb, newVideosDir)
		}
	}
}
