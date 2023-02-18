package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// TwitchCategory holds the schema definition for the TwitchCategory entity.
type TwitchCategory struct {
	ent.Schema
}

// Fields of the TwitchCategory.
func (TwitchCategory) Fields() []ent.Field {
	return []ent.Field{
		field.String("id"),
		field.String("name"),
		field.String("box_art_url").Optional(),
		field.String("igdb_id").Optional(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the TwitchCategory.
func (TwitchCategory) Edges() []ent.Edge {
	return nil
}
