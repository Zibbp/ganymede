package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// LiveTitleRegex holds the schema definition for the LiveTitleRegex entity.
type LiveTitleRegex struct {
	ent.Schema
}

// Fields of the LiveTitleRegex.
func (LiveTitleRegex) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.Bool("negative").Comment("Negative match of the regex").Default(false).StructTag(`json:"negative"`),
		field.String("regex").Comment("Title regex to match"),
		field.Bool("apply_to_videos").Comment("Apply regex to videos and vods").Default(false).StructTag(`json:"apply_to_videos"`),
	}
}

// Edges of the LiveTitleRegex.
func (LiveTitleRegex) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("live", Live.Type).Ref("title_regex").Required().Unique(),
	}
}
