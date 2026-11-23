package archive_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/storage"
	"github.com/zibbp/ganymede/tests"
)

// staticStorageChecker returns a fixed status.
type staticStorageChecker struct{ status storage.Status }

func (c staticStorageChecker) Check(context.Context) (storage.Status, error) { return c.status, nil }

// TestArchiveLowStorageGate verifies new archives are rejected with
// ErrTempDirLowSpace and no records are created.
func TestArchiveLowStorageGate(t *testing.T) {
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	ctx := context.Background()

	vodCountBefore, err := app.Database.Client.Vod.Query().Count(ctx)
	require.NoError(t, err)
	queueCountBefore, err := app.Database.Client.Queue.Query().Count(ctx)
	require.NoError(t, err)

	lowSpace := staticStorageChecker{storage.Status{State: storage.StateWarning, FreeBytes: 1, TotalBytes: 100}}
	app.ArchiveService.StorageChecker = lowSpace

	t.Run("video", func(t *testing.T) {
		_, err := app.ArchiveService.ArchiveVideo(ctx, archive.ArchiveVideoInput{
			VideoId: "123456789",
			Quality: "best",
		})
		require.ErrorIs(t, err, archive.ErrTempDirLowSpace)
	})

	t.Run("livestream", func(t *testing.T) {
		// The gate rejects before the channel is fetched.
		_, err := app.ArchiveService.ArchiveLivestream(ctx, archive.ArchiveVideoInput{
			ChannelId: uuid.New(),
			Quality:   "best",
		})
		require.ErrorIs(t, err, archive.ErrTempDirLowSpace)
	})

	t.Run("clip", func(t *testing.T) {
		_, err := app.ArchiveService.ArchiveClip(ctx, archive.ArchiveClipInput{
			ID:      "SomeClipSlug-abc123",
			Quality: "best",
		})
		require.ErrorIs(t, err, archive.ErrTempDirLowSpace)
	})

	// No VOD or queue rows may be created while archiving is paused.
	vodCountAfter, err := app.Database.Client.Vod.Query().Count(ctx)
	require.NoError(t, err)
	queueCountAfter, err := app.Database.Client.Queue.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, vodCountBefore, vodCountAfter)
	assert.Equal(t, queueCountBefore, queueCountAfter)

	// A healthy temp directory unblocks archiving. The video lookup can fail
	// on its own without platform credentials, so only check the gate.
	app.ArchiveService.StorageChecker = staticStorageChecker{storage.Status{State: storage.StateOK}}
	if app.PlatformTwitch != nil {
		_, err = app.ArchiveService.ArchiveVideo(ctx, archive.ArchiveVideoInput{
			VideoId: "123456789",
			Quality: "best",
		})
		assert.NotErrorIs(t, err, archive.ErrTempDirLowSpace)
	}
}
