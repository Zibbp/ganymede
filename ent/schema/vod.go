package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/zibbp/ganymede/internal/utils"
	"time"
)

// Vod holds the schema definition for the Vod entity.
type Vod struct {
	ent.Schema
}

// Fields of the Vod.
func (Vod) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("ext_id"),
		field.Enum("platform").GoType(utils.VodPlatform("")).Default(string(utils.PlatformTwitch)).Comment("The platform the VOD is from, takes an enum."),
		field.Enum("type").GoType(utils.VodType("")).Default(string(utils.Archive)).Comment("The type of VOD, takes an enum."),
		field.String("title"),
		field.Int("duration").Default(0),
		field.Int("views").Default(0),
		field.String("resolution").Optional(),
		field.Bool("processing").Default(false).Comment("Whether the VOD is currently processing."),
		field.String("thumbnail_path").Optional(),
		field.String("web_thumbnail_path"),
		field.String("video_path"),
		field.String("chat_path").Optional(),
		field.String("chat_video_path").Optional(),
		field.String("info_path").Optional(),
		field.Time("streamed_at").Default(time.Now).Comment("The time the VOD was streamed."),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the Vod.
func (Vod) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("channel", Channel.Type).Ref("vods").Unique().Required(),
		edge.To("queue", Queue.Type).Unique(),
	}
}
