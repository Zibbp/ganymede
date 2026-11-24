package live

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/zibbp/ganymede/ent/channel"
	"github.com/zibbp/ganymede/ent/live"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/utils"
)

// clipsWindowStart returns the lower bound for the clips query. Clips since
// the last successful check are fetched so a clip deferred by low storage
// stays in the window on retry instead of aging out.
func clipsWindowStart(now time.Time, intervalDays int, ignoreLastChecked bool, lastChecked time.Time) time.Time {
	if intervalDays == 0 {
		return time.Time{}
	}
	fixed := now.AddDate(0, 0, -intervalDays)
	if ignoreLastChecked || lastChecked.IsZero() {
		return fixed
	}
	return lastChecked
}

// clipsDeferralAnchor returns the window start to persist when a first
// check is deferred by low storage. The retry then covers the same window.
func clipsDeferralAnchor(lastChecked, startedAt time.Time) (time.Time, bool) {
	if !lastChecked.IsZero() || startedAt.IsZero() {
		return time.Time{}, false
	}
	return startedAt, true
}

// CheckWatchedChannelClips checks watched channels for clips to arhive
func (s *Service) CheckWatchedChannelClips(ctx context.Context, logger zerolog.Logger) error {
	// Get channels from DB
	channels, err := s.Store.Client.Live.Query().Where(live.WatchClips(true)).WithChannel().All(context.Background())
	if err != nil {
		return err
	}

	if len(channels) == 0 {
		logger.Info().Msg("no channels to check")
		return nil
	}

	logger.Info().Msgf("checking %d channels for new clips", len(channels))

	for _, watchedChannel := range channels {

		if watchedChannel.ClipsLimit == 0 || watchedChannel.ClipsIntervalDays == 0 {
			logger.Error().Err(err).Str("channel", watchedChannel.Edges.Channel.Name).Msg("clip limit and clips interval must be greater than 0")
			continue
		}

		now := time.Now()
		// Check if channel clips should be checked
		if !watchedChannel.ClipsIgnoreLastChecked {
			// Check if clips need to be checked for channel
			lastChecked := watchedChannel.ClipsLastChecked

			// Round times down to the nearest hour to avoid partial hour comparisons
			roundedNow := now.Truncate(time.Hour)
			roundedLastChecked := lastChecked.Truncate(time.Hour)

			// Calculate full hours between checks
			diffHours := roundedNow.Sub(roundedLastChecked).Hours()
			// Calculate channel interval in hours
			// Default to 24 hours if not set (archiving all clips)
			var channelIntervalHours int
			if watchedChannel.ClipsIntervalDays == 0 {
				channelIntervalHours = 24
			} else {
				channelIntervalHours = watchedChannel.ClipsIntervalDays * 24
			}

			if diffHours < float64(channelIntervalHours) {
				logger.Info().
					Str("channel", watchedChannel.Edges.Channel.DisplayName).
					Float64("hours_passed", diffHours).
					Int("hours_required", channelIntervalHours).
					Msg("skipping clip check, not enough time has passed")
				continue
			}
		}

		startedAt := clipsWindowStart(now, watchedChannel.ClipsIntervalDays, watchedChannel.ClipsIgnoreLastChecked, watchedChannel.ClipsLastChecked)

		logger.Info().Str("limit", strconv.Itoa(watchedChannel.ClipsLimit)).Str("started_at", startedAt.String()).Msgf("getting clips for channel %s", watchedChannel.Edges.Channel.Name)

		// Get clips
		clips, err := s.PlatformTwitch.GetChannelClips(ctx, watchedChannel.Edges.Channel.ExtID, platform.ClipsFilter{
			Limit:     watchedChannel.ClipsLimit,
			StartedAt: startedAt,
			EndedAt:   now,
		})
		if err != nil {
			logger.Error().Err(err).Str("channel", watchedChannel.Edges.Channel.Name).Msg("error getting clips for channel")
			continue
		}

		logger.Info().Str("channel", watchedChannel.Edges.Channel.Name).Int("clips", len(clips)).Msg("got clips")

		// Fetch all videos from DB
		dbVideos, err := s.Store.Client.Vod.Query().Where(vod.HasChannelWith(channel.ID(watchedChannel.Edges.Channel.ID))).All(context.Background())
		if err != nil {
			logger.Error().Str("channel", watchedChannel.Edges.Channel.Name).Err(err).Msg("error getting videos from database")
			continue
		}

		storageDeferred := false

		// Check if video is already in DB
		for _, clip := range clips {
			logger.Debug().Str("clip_id", clip.ID).Msg("checking if clip should be archived")
			// Video is not in DB
			if !contains(dbVideos, clip.ID) {
				clipQuality := ResolveClipResolutionForLive(watchedChannel)
				if err := ValidateClipResolution(clipQuality); err != nil {
					logger.Error().Err(err).Str("channel", watchedChannel.Edges.Channel.Name).Str("clip_id", clip.ID).Msg("skipping clip with audio-only quality")
					continue
				}
				// Archive clip
				input := archive.ArchiveClipInput{
					ID:          clip.ID,
					Quality:     utils.VodQuality(clipQuality),
					ArchiveChat: watchedChannel.ArchiveChat,
					RenderChat:  watchedChannel.RenderChat,
				}
				_, err = s.ArchiveService.ArchiveClip(ctx, input)
				if errors.Is(err, archive.ErrTempDirLowSpace) {
					logger.Warn().Err(err).Str("clip_id", clip.ID).Msg("temp directory low on space; skipping new clip archive")
					storageDeferred = true
					continue
				}
				if err != nil {
					logger.Error().Err(err).Str("clip_id", clip.ID).Msgf("error archiving clip")
					continue
				}
				logger.Info().Str("clip_id", clip.ID).Msgf("archiving clip")
			} else {
				logger.Debug().Str("clip_id", clip.ID).Msgf("clip already archived")
			}
		}

		// Leave the last checked time so deferred clips are retried on the
		// next check instead of waiting a full interval.
		if storageDeferred {
			// On a first check there is no timestamp to keep, so store this
			// window's start; the retry then covers the same clips.
			if anchor, ok := clipsDeferralAnchor(watchedChannel.ClipsLastChecked, startedAt); ok {
				if err := s.Store.Client.Live.Update().Where(live.ID(watchedChannel.ID)).SetClipsLastChecked(anchor).Exec(ctx); err != nil {
					logger.Error().Err(err).Str("channel", watchedChannel.Edges.Channel.Name).Msg("error storing clip retry anchor")
				}
			}
			continue
		}

		// Set clip last checked time on channel
		err = s.Store.Client.Live.Update().Where(live.ID(watchedChannel.ID)).SetClipsLastChecked(now).Exec(ctx)
		if err != nil {
			logger.Error().Err(err).Str("channe", watchedChannel.Edges.Channel.Name).Msg("error setting clip last check time on channel")
		}
	}
	return nil
}
