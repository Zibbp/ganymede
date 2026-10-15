package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// StorageFinding holds the schema definition for the StorageFinding entity. A finding is a
// directory in the videos directory that the storage reconciliation job could not tie to a
// video in the database.
type StorageFinding struct {
	ent.Schema
}

// Fields of the StorageFinding.
func (StorageFinding) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.Enum("kind").Values("orphaned_directory").Comment("What kind of finding this is."),
		field.String("path").Unique().NotEmpty().Comment("Absolute path of the directory."),
		field.Int64("size_bytes").Default(0).Comment("Size of the directory in bytes."),
		field.Time("detected_at").Default(time.Now).Comment("When the directory was first found."),
	}
}

// Edges of the StorageFinding.
func (StorageFinding) Edges() []ent.Edge {
	return nil
}
