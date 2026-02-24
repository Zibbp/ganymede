package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Notification holds the schema definition for the Notification entity.
type Notification struct {
	ent.Schema
}

// Fields of the Notification.
func (Notification) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("name").NotEmpty().MaxLen(255).Comment("User-given name for this notification configuration."),
		field.Bool("enabled").Default(true).Comment("Whether this notification configuration is active."),
		field.Enum("type").Values("webhook", "apprise").Comment("The provider type for this notification."),
		field.String("url").NotEmpty().MaxLen(2048).Comment("The webhook or Apprise API URL."),

		// Event triggers
		field.Bool("trigger_video_success").Default(false).Comment("Fire on video archive success."),
		field.Bool("trigger_live_success").Default(false).Comment("Fire on live archive success."),
		field.Bool("trigger_error").Default(false).Comment("Fire on task error."),
		field.Bool("trigger_is_live").Default(false).Comment("Fire when a channel goes live."),

		// Templates
		field.String("video_success_template").MaxLen(4096).Default("✅ Video Archived: {{vod_title}} by {{channel_display_name}}.").Comment("Template for video archive success body."),
		field.String("live_success_template").MaxLen(4096).Default("✅ Live Stream Archived: {{vod_title}} by {{channel_display_name}}.").Comment("Template for live archive success body."),
		field.String("error_template").MaxLen(4096).Default("⚠️ Error: Queue {{queue_id}} failed at task {{failed_task}}.").Comment("Template for error body."),
		field.String("is_live_template").MaxLen(4096).Default("🔴 {{channel_display_name}} is live!").Comment("Template for is-live body."),

		// Apprise-specific fields (optional, only used when type=apprise)
		field.String("apprise_urls").Optional().Default("").MaxLen(4096).Comment("Stateless Apprise URLs parameter."),
		field.String("apprise_title").Optional().Default("").MaxLen(4096).Comment("Apprise notification title template."),
		field.Enum("apprise_type").Values("info", "success", "warning", "failure").Default("info").Optional().Comment("Apprise notification type."),
		field.String("apprise_tag").Optional().Default("").MaxLen(255).Comment("Apprise tag for stateful mode."),
		field.Enum("apprise_format").Values("text", "html", "markdown").Default("text").Optional().Comment("Apprise message format."),

		// Timestamps
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the Notification.
func (Notification) Edges() []ent.Edge {
	return nil
}
