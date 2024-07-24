package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// BlockedVods holds the schema definition for the BlockedVods entity.
type BlockedVods struct {
	ent.Schema
}

// Fields of the BlockedVods.
func (BlockedVods) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Comment("The ID of the blocked vod."),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the BlockedVods.
func (BlockedVods) Edges() []ent.Edge {
	return nil
}
