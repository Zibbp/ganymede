package platform

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestParseKickTime(t *testing.T) {
	got := parseKickTime("2026-11-19 21:03:03")
	want := time.Date(2026, 11, 19, 21, 3, 3, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("parseKickTime = %v, want %v", got, want)
	}
}

func TestLargestSrcsetURL(t *testing.T) {
	got := largestSrcsetURL("https://a/1080.webp?v=1 1920w, https://a/720.webp 1280w, https://a/160.webp 284w", "https://a/fallback")
	if got != "https://a/1080.webp?v=1" {
		t.Fatalf("got %q", got)
	}
	if got := largestSrcsetURL("", "https://a/fallback"); got != "https://a/fallback" {
		t.Fatalf("fallback: got %q", got)
	}
}

func TestKickChannelExtID(t *testing.T) {
	ext := KickChannelExtID(668)
	if ext != "kick:668" {
		t.Fatalf("ext id = %q", ext)
	}
	id, err := ParseKickChannelExtID(ext)
	if err != nil || id != 668 {
		t.Fatalf("parse = %d, %v", id, err)
	}
}

// TestKickLive hits the real Kick API; set KICK_LIVE_TEST_CHANNEL to a live channel.
func TestKickLive(t *testing.T) {
	slug := os.Getenv("KICK_LIVE_TEST_CHANNEL")
	if slug == "" {
		t.Skip("KICK_LIVE_TEST_CHANNEL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	k := NewKickConnection()
	ch, err := k.GetChannel(ctx, &slug, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("channel: %+v", *ch)
	ls, err := k.GetLiveStream(ctx, slug)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("live: id=%s title=%q game=%q started=%s thumb=%s", ls.ID, ls.Title, ls.GameName, ls.StartedAt, ls.ThumbnailURL)
	pl, err := k.GetLiveMasterPlaylist(ctx, slug)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range pl.Variants {
		t.Logf("variant video=%q uri=%.60s", v.Video, v.URI)
	}
	live, err := k.CheckIfStreamIsLive(ctx, slug)
	if err != nil || !live {
		t.Fatalf("CheckIfStreamIsLive(%s) = %v, %v; want true", slug, live, err)
	}
	if offline := os.Getenv("KICK_OFFLINE_TEST_CHANNEL"); offline != "" {
		live, err := k.CheckIfStreamIsLive(ctx, offline)
		if err != nil || live {
			t.Fatalf("CheckIfStreamIsLive(%s) = %v, %v; want false", offline, live, err)
		}
	}
	key, cluster := k.KickRealtimeCredentials(ctx, 668)
	t.Logf("pusher %s/%s", key, cluster)
	emotes, err := k.GetChannelEmotes(ctx, slug)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("emotes: %d", len(emotes))
}
