package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/zibbp/ganymede/internal/utils"
)

// Playback holds the schema definition for the Playback entity.
type Playback struct {
	ent.Schema
}

// Fields of the Playback.
func (Playback) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.UUID("vod_id", uuid.UUID{}),
		field.UUID("user_id", uuid.UUID{}),
		field.Int("time").Default(0),
		field.Enum("status").GoType(utils.PlaybackStatus("")).Default(string(utils.InProgress)).Optional(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the Playback.
func (Playback) Edges() []ent.Edge {
	return nil
}

func (Playback) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("vod_id", "user_id").Unique(),
	}
}
