package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// MutedSegment holds the schema definition for the MutedSegment entity.
type MutedSegment struct {
	ent.Schema
}

// Fields of the MutedSegment.
func (MutedSegment) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.Int("start").Comment("The start time of the muted segment"),
		field.Int("end").Comment("The end time of the muted segment"),
	}
}

// Edges of the MutedSegment.
func (MutedSegment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("vod", Vod.Type).Ref("muted_segments").Unique().Required(),
	}
}
