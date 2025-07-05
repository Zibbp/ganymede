package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Channel holds the schema definition for the Channel entity.
type Channel struct {
	ent.Schema
}

// Fields of the Channel.
func (Channel) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("ext_id").Unique().Comment("The external ID of the channel.").Optional(),
		field.String("name").Unique(),
		field.String("display_name").Unique(),
		field.String("image_path"),
		field.Bool("retention").Default(false),
		field.Int64("retention_days").Optional(),
		field.Int64("storage_size_bytes").Default(0).Comment("Total storage size in bytes for the channel's videos."),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the Channel.
func (Channel) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("vods", Vod.Type),
		edge.To("live", Live.Type),
	}
}
