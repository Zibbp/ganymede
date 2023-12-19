package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Chapter holds the schema definition for the Chapter entity.
type Chapter struct {
	ent.Schema
}

// Fields of the Chapter.
func (Chapter) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("type").Optional(),
		field.String("title").Optional(),
		field.Int("start").Optional(),
		field.Int("end").Optional(),
	}
}

// Edges of the Chapter.
func (Chapter) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("vod", Vod.Type).Ref("chapters").Unique().Required(),
	}
}
