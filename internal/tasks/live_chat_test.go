package tasks

import (
	"context"
	"errors"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/tests/pgtest"
)

func TestPickStreamVod(t *testing.T) {
	t.Parallel()

	videos := []platform.VideoInfo{
		{ID: "10", StreamID: "100"},
		{ID: "20", StreamID: "200"},
		{ID: "30", StreamID: "300"},
	}

	tests := []struct {
		name        string
		videos      []platform.VideoInfo
		extStreamID string
		wantID      string
	}{
		{
			name:        "matches archived broadcast by stream id",
			videos:      videos,
			extStreamID: "200",
			wantID:      "20",
		},
		{
			name:        "first match wins when stream ids repeat",
			videos:      []platform.VideoInfo{{ID: "11", StreamID: "100"}, {ID: "12", StreamID: "100"}},
			extStreamID: "100",
			wantID:      "11",
		},
		{
			name:        "no matching stream id",
			videos:      videos,
			extStreamID: "999",
			wantID:      "",
		},
		{
			name:        "archived broadcast matches the last video",
			videos:      videos,
			extStreamID: "300",
			wantID:      "30",
		},
		{
			name:        "no videos",
			videos:      nil,
			extStreamID: "100",
			wantID:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := pickStreamVod(tt.videos, tt.extStreamID)
			if tt.wantID == "" && got != nil {
				t.Fatalf("pickStreamVod() = %+v, want nil", got)
			}
			if tt.wantID != "" && (got == nil || got.ID != tt.wantID) {
				t.Fatalf("pickStreamVod() = %+v, want video id %s", got, tt.wantID)
			}
		})
	}
}

type liveChatAnchorPlatform struct {
	platform.Platform
	liveStream    *platform.LiveStreamInfo
	liveErr       error
	liveCalls     int
	videoCalls    int
	requestedName string
}

func (p *liveChatAnchorPlatform) GetLiveStream(_ context.Context, channelName string) (*platform.LiveStreamInfo, error) {
	p.liveCalls++
	p.requestedName = channelName
	return p.liveStream, p.liveErr
}

func (p *liveChatAnchorPlatform) GetVideos(context.Context, string, platform.VideoType, bool, bool) ([]platform.VideoInfo, error) {
	p.videoCalls++
	return nil, errors.New("unexpected VOD poll")
}

func TestResolveLiveChatVideoStartTime(t *testing.T) {
	db := pgtest.Open(t)
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	store := &database.Database{SQLDB: db, Client: client}
	require.NoError(t, client.Schema.Create(t.Context()))
	channel := client.Channel.Create().SetName("test").SetDisplayName("Test").SetImagePath("").SaveX(t.Context())
	reference := time.Now().UTC().Truncate(time.Second)
	published := []platform.VideoInfo{{
		ID: "vod", StreamID: "broadcast",
		CreatedAt: reference.Add(-time.Hour), Duration: time.Hour + 55*time.Second,
	}}
	for _, tc := range []struct {
		name       string
		liveStream *platform.LiveStreamInfo
		liveErr    error
		videos     []platform.VideoInfo
		want       time.Time
	}{
		{name: "stopped capture of ongoing broadcast without VOD", liveStream: &platform.LiveStreamInfo{ID: "broadcast"}, want: reference},
		{name: "ongoing broadcast has only a partial VOD", liveStream: &platform.LiveStreamInfo{ID: "broadcast"}, videos: published, want: reference},
		{name: "ended broadcast", videos: published, want: reference.Add(-5 * time.Second)},
		{name: "channel started another broadcast", liveStream: &platform.LiveStreamInfo{ID: "new-broadcast"}, videos: published, want: reference.Add(-5 * time.Second)},
		{name: "live lookup unavailable", liveErr: errors.New("platform unavailable"), videos: published, want: reference.Add(-5 * time.Second)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			video := client.Vod.Create().SetChannel(channel).SetExtID("capture").SetExtStreamID("broadcast").
				SetTitle("Test").SetWebThumbnailPath("").SetVideoPath("").SaveX(t.Context())
			queue := client.Queue.Create().SetVod(video).SetChatStart(reference).SaveX(t.Context())
			items := &GetDatabaseItemsResponse{Queue: *queue, Video: *video, Channel: *channel}
			provider := &liveChatAnchorPlatform{liveStream: tc.liveStream, liveErr: tc.liveErr}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			got, err := resolveLiveChatVideoStartTime(ctx, store, provider, items, tc.videos, 60)
			require.NoError(t, err)
			require.NoError(t, ctx.Err(), "chat conversion must not wait for a broadcast that is still live")
			require.True(t, tc.want.Equal(got), "got %s, want %s", got, tc.want)
			require.Equal(t, 1, provider.liveCalls)
			require.Equal(t, channel.Name, provider.requestedName)
			require.Zero(t, provider.videoCalls)
			persisted := client.Vod.GetX(t.Context(), video.ID)
			require.NotNil(t, persisted.VideoStartAt)
			require.True(t, tc.want.Equal(*persisted.VideoStartAt))
		})
	}
}
