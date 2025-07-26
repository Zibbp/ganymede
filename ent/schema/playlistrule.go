package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/zibbp/ganymede/internal/utils"
)

// PlaylistRule holds the schema definition for the PlaylistRule entity.
type PlaylistRule struct {
	ent.Schema
}

// Fields of the PlaylistRule.
func (PlaylistRule) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("name").Comment("The name of the rule, used for display purposes.").Optional(),
		field.Enum("field").GoType(utils.PlaylistRuleField("")).Comment("The field of the rule, used to determine which property of the VOD the rule applies to.").Default(string(utils.FieldTitle)),
		field.Enum("operator").GoType(utils.PlaylistRuleOperator("")).Comment("The operator of the rule, used to determine how the rule is applied.").Default(string(utils.OperatorContains)),
		field.String("value").Comment("Value to match against."),
		field.Int("position").Default(0).Comment("Order within group"),
		field.Bool("enabled").Default(true).Comment("Is the rule active?"),
	}
}

// Edges of the PlaylistRule.
func (PlaylistRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("group", PlaylistRuleGroup.Type).
			Ref("rules").
			Required().
			Unique(),
	}
}
