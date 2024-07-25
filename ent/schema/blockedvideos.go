package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// BlockedVideos holds the schema definition for the BlockedVideos entity.
type BlockedVideos struct {
	ent.Schema
}

// Fields of the BlockedVideos.
func (BlockedVideos) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Comment("The ID of the blocked vod."),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the BlockedVideos.
func (BlockedVideos) Edges() []ent.Edge {
	return nil
}
