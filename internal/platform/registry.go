package platform

import (
	"fmt"

	"github.com/zibbp/ganymede/internal/utils"
)

// Registry resolves the Platform implementation for a utils.VideoPlatform.
// Services select providers through it instead of holding one field per
// platform.
type Registry struct {
	platforms map[utils.VideoPlatform]Platform
}

// NewRegistry builds a registry; nil implementations (e.g. Twitch without
// credentials) are left unregistered.
func NewRegistry(platforms map[utils.VideoPlatform]Platform) Registry {
	r := Registry{platforms: make(map[utils.VideoPlatform]Platform, len(platforms))}
	for p, impl := range platforms {
		if impl != nil {
			r.platforms[p] = impl
		}
	}
	return r
}

// Get returns the implementation for p. An empty platform means Twitch, the
// database default for rows created before platforms were stored.
func (r Registry) Get(p utils.VideoPlatform) (Platform, error) {
	if p == "" {
		p = utils.PlatformTwitch
	}
	impl, ok := r.platforms[p]
	if !ok {
		if p == utils.PlatformTwitch {
			return nil, fmt.Errorf("twitch platform is not configured; check TWITCH_CLIENT_ID and TWITCH_CLIENT_SECRET")
		}
		return nil, fmt.Errorf("%s platform is not configured", p)
	}
	return impl, nil
}
