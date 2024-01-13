package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// MultistreamInfo holds the schema definition for the MultistreamInfo entity.
type MultistreamInfo struct {
	ent.Schema
}

// Fields of the MultistreamInfo.
func (MultistreamInfo) Fields() []ent.Field {
	return []ent.Field{
		field.Int("delay_ms").Optional(),
	}
}

// Edges of the MultistreamInfo.
func (MultistreamInfo) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("vod", Vod.Type).Immutable().Unique().Required(),
		edge.From("playlist", Playlist.Type).Ref("multistream_info").Unique().Required(),
	}
}
