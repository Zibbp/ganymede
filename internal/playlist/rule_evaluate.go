package playlist

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/playlist"
	playlistrulegroup "github.com/zibbp/ganymede/ent/playlistrulegroup"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/utils"
)

// EvaluateRule tests a single rule against a video.
func EvaluateRule(logger zerolog.Logger, video *ent.Vod, rule *ent.PlaylistRule) bool {
	val := rule.Value
	logger.Debug().
		Str("rule_id", rule.ID.String()).
		Str("field", string(rule.Field)).
		Str("operator", string(rule.Operator)).
		Str("value", val).
		Msg("Evaluating rule")

	switch rule.Field {
	case utils.FieldTitle:
		result := matchString(video.Title, rule.Operator, val)

		logger.Debug().
			Str("field_value", video.Title).
			Bool("result", result).
			Msg("Rule evaluation result (title)")

		return result

	case utils.FieldType:
		result := matchString(string(video.Type), rule.Operator, val)

		logger.Debug().
			Str("field_value", string(video.Type)).
			Bool("result", result).
			Msg("Rule evaluation result (type)")

		return result

	case utils.FieldPlatform:
		result := matchString(string(video.Platform), rule.Operator, val)

		logger.Debug().
			Str("field_value", string(video.Platform)).
			Bool("result", result).
			Msg("Rule evaluation result (platform)")

		return result

	case utils.FieldCategory:
		if len(video.Edges.Chapters) == 0 {
			logger.Debug().Msg("No chapters to match against for category rule")
			return false
		}

		chapters := []string{}
		for _, chapter := range video.Edges.Chapters {
			if chapter != nil && chapter.Title != "" {
				chapters = append(chapters, chapter.Title)
			}
		}

		result := matchCategory(chapters, rule.Operator, val)
		logger.Debug().
			Strs("categories", chapters).
			Bool("result", result).
			Msg("Rule evaluation result (category)")
		return result

	case utils.FieldChannelName:
		if video.Edges.Channel == nil || video.Edges.Channel.Name == "" {
			logger.Debug().Msg("No channel name to match against for channel name rule")
			return false
		}
		result := matchString(video.Edges.Channel.Name, rule.Operator, val)
		logger.Debug().
			Str("channel_name", video.Edges.Channel.Name).
			Bool("result", result).
			Msg("Rule evaluation result (channel name)")
		return result

	default:
		logger.Debug().Msg("Unknown rule field")
		return false
	}
}

// matchString checks if the field matches the value based on the operator.
func matchString(field string, op utils.PlaylistRuleOperator, val string) bool {
	switch op {
	case utils.OperatorEquals:
		return field == val
	case utils.OperatorContains:
		return strings.Contains(strings.ToLower(field), strings.ToLower(val))
	case utils.OperatorRegex:
		re, err := regexp.Compile(val)
		if err != nil {
			return false
		}
		return re.MatchString(field)
	default:
		return false
	}
}

// matchCategory checks if any of the categories match the value based on the operator.
func matchCategory(categories []string, op utils.PlaylistRuleOperator, val string) bool {
	for _, cat := range categories {
		if matchString(cat, op, val) {
			return true
		}
	}
	return false
}

// EvaluateRuleGroup evaluates all rules in a group using group's operator (AND/OR).
func EvaluateRuleGroup(logger zerolog.Logger, video *ent.Vod, group *ent.PlaylistRuleGroup) bool {
	rules := group.Edges.Rules
	logger.Debug().
		Str("group_id", group.ID.String()).
		Str("operator", string(group.Operator)).
		Int("rule_count", len(rules)).
		Msg("Evaluating rule group")
	if group.Operator == playlistrulegroup.OperatorAND {
		for _, r := range rules {
			if !r.Enabled {
				logger.Debug().
					Str("rule_id", r.ID.String()).
					Msg("Skipping disabled rule")
				continue
			}
			if !EvaluateRule(logger, video, r) {
				logger.Debug().
					Str("group_id", group.ID.String()).
					Str("rule_id", r.ID.String()).
					Msg("Rule group AND failed")
				return false
			}
		}
		logger.Debug().
			Str("group_id", group.ID.String()).
			Msg("Rule group AND passed")
		return true
	}
	// OR logic
	for _, r := range rules {
		if !r.Enabled {
			logger.Debug().
				Str("rule_id", r.ID.String()).
				Msg("Skipping disabled rule")
			continue
		}
		if EvaluateRule(logger, video, r) {
			logger.Debug().
				Str("group_id", group.ID.String()).
				Str("rule_id", r.ID.String()).
				Msg("Rule group OR passed")
			return true
		}
	}
	logger.Debug().
		Str("group_id", group.ID.String()).
		Msg("Rule group OR failed")
	return false
}

// EvaluatePlaylist returns true if any rule group matches the video.
func EvaluatePlaylist(logger zerolog.Logger, video *ent.Vod, groups []*ent.PlaylistRuleGroup) bool {
	logger.Debug().
		Int("group_count", len(groups)).
		Msg("Evaluating playlist rule groups")
	for _, group := range groups {
		if EvaluateRuleGroup(logger, video, group) {
			logger.Debug().
				Str("group_id", group.ID.String()).
				Msg("Playlist evaluation passed for group")
			return true
		}
	}
	logger.Debug().Msg("Playlist evaluation failed for all groups")
	return false
}

// ShouldVideoBeInPlaylist loads rules and evaluates the video against them.
func (s *Service) ShouldVideoBeInPlaylist(ctx context.Context, videoId uuid.UUID, playlistID uuid.UUID) (bool, error) {
	logger := log.With().Str("playlist_id", playlistID.String()).
		Str("video_id", videoId.String()).Logger()

	// Query the video here rather than passing it in
	// This is necessary to ensure we have the latest data and necessary edges loaded
	video, err := s.Store.Client.Vod.Query().
		Where(entVod.ID(videoId)).
		WithChapters().
		WithChannel().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return false, nil // Video not found, so it can't match any rules
		}
		return false, err
	}

	logger.Debug().Msg("Evaluating video against playlist rules")

	groups, err := s.Store.Client.PlaylistRuleGroup.
		Query().
		Where(playlistrulegroup.HasPlaylistWith(playlist.IDEQ(playlistID))).
		WithRules().
		All(ctx)
	if err != nil {
		return false, err
	}

	// If no groups, no rules to evaluate
	if len(groups) == 0 {
		logger.Debug().Msg("No rule groups found for playlist, video cannot match any rules")
		return false, nil
	}

	return EvaluatePlaylist(logger, video, groups), nil
}
