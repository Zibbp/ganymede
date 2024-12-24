package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Sessions holds the schema definition for the Sessions entity.
type Sessions struct {
	ent.Schema
}

// Fields of the Sessions.
func (Sessions) Fields() []ent.Field {
	return []ent.Field{
		field.Text("token").Unique().NotEmpty().Immutable(),
		field.Bytes("data").NotEmpty(),
		field.Time("expiry"),
	}
}

func (Sessions) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("expiry"),
	}
}

// Edges of the Sessions.
func (Sessions) Edges() []ent.Edge {
	return nil
}
