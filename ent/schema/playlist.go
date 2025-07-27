package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Playlist holds the schema definition for the Playlist entity.
type Playlist struct {
	ent.Schema
}

// Fields of the Playlist.
func (Playlist) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("name").Unique(),
		field.String("description").Optional(),
		field.String("thumbnail_path").Optional(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the Playlist.
func (Playlist) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("vods", Vod.Type),
		edge.To("multistream_info", MultistreamInfo.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("rule_groups", PlaylistRuleGroup.Type).Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
