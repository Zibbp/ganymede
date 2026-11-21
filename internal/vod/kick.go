package vod

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/utils"
)

// kickFallback runs fetch against the Kick platform for a Kick VOD whose chat
// has no embedded assets. Failures degrade to no assets rather than an error so
// chat playback still works.
func kickFallback[T any](ctx context.Context, s *Service, v *ent.Vod, fetch func(platform.Platform, string) ([]T, error)) []T {
	kick, err := s.Platforms.Get(utils.PlatformKick)
	if err != nil {
		return nil
	}
	ch, err := v.QueryChannel().Only(ctx)
	if err != nil {
		log.Debug().Err(err).Str("vod_id", v.ID.String()).Msg("kick chat fallback: channel not found")
		return nil
	}
	items, err := fetch(kick, platform.ChannelLogin(ch.Name, ch.ExtLogin))
	if err != nil {
		log.Warn().Err(err).Str("vod_id", v.ID.String()).Msg("kick chat fallback fetch failed")
		return nil
	}
	return items
}
