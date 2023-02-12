package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Live holds the schema definition for the Live entity.
type Live struct {
	ent.Schema
}

// Fields of the Live.
func (Live) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.Bool("watch_live").Default(true).Comment("Watch live streams"),
		field.Bool("watch_vod").Default(false).Comment("Watch new VODs"),
		field.Bool("download_archives").Default(false).Comment("Download archives"),
		field.Bool("download_highlights").Default(false).Comment("Download highlights"),
		field.Bool("download_uploads").Default(false).Comment("Download uploads"),
		field.Bool("download_sub_only").Default(false).Comment("Download sub only VODs"),
		field.Bool("is_live").Default(false).Comment("Whether the channel is currently live."),
		field.Bool("archive_chat").Default(true).Comment("Whether the chat archive is enabled."),
		field.String("resolution").Default("best").Optional(),
		field.Time("last_live").Default(time.Now).Comment("The time the channel last went live."),
		field.Bool("render_chat").Default(true).Comment("Whether the chat should be rendered."),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the Live.
func (Live) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("channel", Channel.Type).Ref("live").Unique().Required(),
		edge.To("categories", LiveCategory.Type).StorageKey(edge.Column("live_id")),
	}
}
