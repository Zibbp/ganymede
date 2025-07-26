package playlist

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/playlist"
	playlistrulegroup "github.com/zibbp/ganymede/ent/playlistrulegroup"
	"github.com/zibbp/ganymede/internal/utils"
)

type RuleInput struct {
	Name     string
	Field    utils.PlaylistRuleField
	Operator utils.PlaylistRuleOperator
	Value    string
	Position int
	Enabled  bool
}

type RuleGroupInput struct {
	Operator string
	Position int
	Rules    []RuleInput
}

// SetPlaylistRules replaces all rule groups and rules for a given playlist.
func (s *Service) SetPlaylistRules(ctx context.Context, playlistID uuid.UUID, ruleGroups []RuleGroupInput) ([]*ent.PlaylistRuleGroup, error) {
	// Validate input
	for _, g := range ruleGroups {
		if g.Operator != "AND" && g.Operator != "OR" {
			return nil, fmt.Errorf("invalid group operator: %s", g.Operator)
		}
		for _, r := range g.Rules {
			if err := validateRule(r); err != nil {
				return nil, fmt.Errorf("invalid rule: %w", err)
			}
		}
	}

	// Start transaction
	tx, err := s.Store.Client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	// Delete old rule groups and rules
	_, err = tx.PlaylistRuleGroup.
		Delete().
		Where(playlistrulegroup.HasPlaylistWith(playlist.IDEQ(playlistID))).
		Exec(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("failed to delete old rule groups: %w", err)
	}

	// Create new rule groups and rules
	playlist, err := tx.Playlist.Get(ctx, playlistID)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("playlist not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get playlist: %w", err)
	}
	var createdGroups []*ent.PlaylistRuleGroup

	for _, g := range ruleGroups {
		group, err := tx.PlaylistRuleGroup.Create().
			SetOperator(playlistrulegroup.Operator(g.Operator)).
			SetPosition(g.Position).
			SetPlaylist(playlist).
			Save(ctx)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}

		for _, r := range g.Rules {
			_, err := tx.PlaylistRule.Create().
				SetName(r.Name).
				SetField(r.Field).
				SetOperator(r.Operator).
				SetValue(r.Value).
				SetPosition(r.Position).
				SetEnabled(r.Enabled).
				SetGroup(group).
				Save(ctx)
			if err != nil {
				_ = tx.Rollback()
				return nil, err
			}
		}

		createdGroups = append(createdGroups, group)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return createdGroups, nil
}

// GetPlaylistRules retrieves all rule groups and their rules for a given playlist.
func (s *Service) GetPlaylistRules(ctx context.Context, playlistID uuid.UUID) ([]*ent.PlaylistRuleGroup, error) {
	// Fetch the playlist with its rule groups and rules
	playlist, err := s.Store.Client.Playlist.Query().
		Where(playlist.ID(playlistID)).
		WithRuleGroups(func(q *ent.PlaylistRuleGroupQuery) {
			q.WithRules()
		}).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("playlist not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get playlist rules: %w", err)
	}

	return playlist.Edges.RuleGroups, nil
}

// TestPlaylistRules checks if a video should be included in a playlist based on its rules.
// Primarily used for testing purposes in the frontend.
func (s *Service) TestPlaylistRules(ctx context.Context, playlistID uuid.UUID, videoID uuid.UUID) (bool, error) {
	return s.ShouldVideoBeInPlaylist(ctx, videoID, playlistID)
}

// isValidFieldOperator checks if the operator is valid for the given field.
func isValidFieldOperator(field utils.PlaylistRuleField, op utils.PlaylistRuleOperator) bool {
	// Define valid operators for each field
	validFieldOperators := map[utils.PlaylistRuleField][]utils.PlaylistRuleOperator{
		utils.FieldTitle:       {utils.OperatorEquals, utils.OperatorContains, utils.OperatorRegex},
		utils.FieldType:        {utils.OperatorEquals, utils.OperatorContains, utils.OperatorRegex},
		utils.FieldCategory:    {utils.OperatorEquals, utils.OperatorContains, utils.OperatorRegex},
		utils.FieldPlatform:    {utils.OperatorEquals, utils.OperatorContains, utils.OperatorRegex},
		utils.FieldChannelName: {utils.OperatorEquals, utils.OperatorContains, utils.OperatorRegex},
	}

	for _, allowed := range validFieldOperators[field] {
		if allowed == op {
			return true
		}
	}
	return false
}

// validateRule checks if the rule input is valid.
func validateRule(r RuleInput) error {
	validFields := map[utils.PlaylistRuleField]bool{
		utils.FieldTitle:       true,
		utils.FieldType:        true,
		utils.FieldCategory:    true,
		utils.FieldPlatform:    true,
		utils.FieldChannelName: true,
	}

	if !validFields[r.Field] {
		return fmt.Errorf("invalid rule field: %s", r.Field)
	}

	validOperators := map[utils.PlaylistRuleOperator]bool{
		utils.OperatorEquals:   true,
		utils.OperatorContains: true,
		utils.OperatorRegex:    true,
	}

	if !validOperators[r.Operator] {
		return fmt.Errorf("invalid operator: %s", r.Operator)
	}

	if r.Field == "" || r.Operator == "" || r.Value == "" {
		return fmt.Errorf("field, operator, and value must be provided")
	}

	if !isValidFieldOperator(r.Field, r.Operator) {
		return fmt.Errorf("operator %s is not valid for field %s", r.Operator, r.Field)
	}

	if r.Operator == utils.OperatorRegex {
		if _, err := regexp.Compile(r.Value); err != nil {
			return fmt.Errorf("invalid regex: %v", err)
		}
	}

	return nil
}
