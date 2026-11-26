package live

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/tests/pgtest"
)

// A finalizing archive must block the automatic check for the same stream.
// A different stream ID is still allowed.
func TestActiveStreamArchiveExists(t *testing.T) {
	ctx := t.Context()
	store := database.NewDatabase(ctx, database.DatabaseConnectionInput{DBString: pgtest.ConnectionString(t), IsWorker: true})
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	service := &Service{Store: store}
	channel := store.Client.Channel.Create().SetName("eerie").SetDisplayName("eerie").SetImagePath("").SaveX(ctx)

	createVod := func(streamID string, videoPlatform utils.VideoPlatform, status utils.ArchiveStatus) *ent.Vod {
		return store.Client.Vod.Create().
			SetChannel(channel).
			SetExtID(streamID).
			SetExtStreamID(streamID).
			SetPlatform(videoPlatform).
			SetType(utils.Live).
			SetTitle("Test live archive").
			SetStatus(status).
			SetWebThumbnailPath("").
			SetVideoPath("").
			SaveX(ctx)
	}

	vod := createVod("318416374626", utils.PlatformTwitch, utils.ArchiveFinalizing)
	active, err := service.activeStreamArchiveExists(ctx, channel.ID, utils.PlatformTwitch, "318416374626")
	require.NoError(t, err)
	require.True(t, active, "a finalizing capture must prevent an automatic duplicate")

	for _, status := range []utils.ArchiveStatus{utils.ArchiveQueued, utils.ArchiveRunning, utils.ArchiveFinalizing} {
		store.Client.Vod.UpdateOneID(vod.ID).SetStatus(status).ExecX(ctx)
		active, err = service.activeStreamArchiveExists(ctx, channel.ID, utils.PlatformTwitch, "318416374626")
		require.NoError(t, err)
		require.True(t, active, string(status))
	}

	for _, status := range []utils.ArchiveStatus{utils.ArchiveCompleted, utils.ArchiveFailed} {
		store.Client.Vod.UpdateOneID(vod.ID).SetStatus(status).ExecX(ctx)
		active, err = service.activeStreamArchiveExists(ctx, channel.ID, utils.PlatformTwitch, "318416374626")
		require.NoError(t, err)
		require.False(t, active, string(status))
	}

	createVod("other-stream", utils.PlatformTwitch, utils.ArchiveQueued)
	active, err = service.activeStreamArchiveExists(ctx, channel.ID, utils.PlatformTwitch, "other-stream")
	require.NoError(t, err)
	require.True(t, active)

	active, err = service.activeStreamArchiveExists(ctx, channel.ID, utils.PlatformTwitch, "unknown-stream")
	require.NoError(t, err)
	require.False(t, active, "a new broadcast has a different stream ID")

	active, err = service.activeStreamArchiveExists(ctx, channel.ID, utils.PlatformTwitch, "")
	require.NoError(t, err)
	require.False(t, active)

	createVod("youtube-stream", utils.PlatformYoutube, utils.ArchiveRunning)
	active, err = service.activeStreamArchiveExists(ctx, channel.ID, utils.PlatformYoutube, "youtube-stream")
	require.NoError(t, err)
	require.True(t, active)
	active, err = service.activeStreamArchiveExists(ctx, channel.ID, utils.PlatformTwitch, "youtube-stream")
	require.NoError(t, err)
	require.False(t, active, "stream identity includes the platform")
}
