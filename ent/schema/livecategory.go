package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// LiveCategory holds the schema definition for the LiveCategory entity.
type LiveCategory struct {
	ent.Schema
}

// Fields of the LiveCategory.
func (LiveCategory) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("name").Optional().Nillable(),
	}
}

// Edges of the LiveCategory.
func (LiveCategory) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("live", Live.Type).Ref("categories").Unique().Required(),
	}
}
