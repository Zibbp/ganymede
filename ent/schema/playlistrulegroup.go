package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// PlaylistRuleGroup holds the schema definition for the PlaylistRuleGroup entity.
type PlaylistRuleGroup struct {
	ent.Schema
}

// Fields of the PlaylistRuleGroup.
func (PlaylistRuleGroup) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.Enum("operator").
			Values("AND", "OR").
			Default("AND").
			Comment("Logical operator to combine rules in this group"),
		field.Int("position").
			Default(0).
			Comment("Used to order groups within the playlist"),
	}
}

// Edges of the PlaylistRuleGroup.
func (PlaylistRuleGroup) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("playlist", Playlist.Type).
			Ref("rule_groups").
			Required().
			Unique(),
		edge.To("rules", PlaylistRule.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
