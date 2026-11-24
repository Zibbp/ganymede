package platform

import (
	"testing"

	"github.com/zibbp/ganymede/internal/utils"
)

func TestRegistryGet(t *testing.T) {
	kick := NewKickConnection()
	r := NewRegistry(map[utils.VideoPlatform]Platform{
		utils.PlatformTwitch: nil, // unconfigured Twitch
		utils.PlatformKick:   kick,
	})

	got, err := r.Get(utils.PlatformKick)
	if err != nil || got != kick {
		t.Fatalf("Get(kick) = %v, %v", got, err)
	}
	if _, err := r.Get(utils.PlatformTwitch); err == nil {
		t.Fatal("Get(twitch) should fail when Twitch is not configured")
	}
	if _, err := r.Get(""); err == nil {
		t.Fatal("empty platform should resolve to the unconfigured Twitch and fail")
	}
	if _, err := r.Get(utils.PlatformYoutube); err == nil {
		t.Fatal("Get(youtube) should fail")
	}
}
