package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/zibbp/ganymede/internal/utils"
)

// Queue holds the schema definition for the Queue entity.
type Queue struct {
	ent.Schema
}

// Fields of the Queue.
func (Queue) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.Bool("live_archive").Default(false),
		field.Bool("on_hold").Default(false),
		field.Bool("video_processing").Default(true),
		field.Bool("chat_processing").Default(true),
		field.Bool("processing").Default(true),
		field.Enum("task_vod_create_folder").GoType(utils.TaskStatus("")).Default(string(utils.Pending)).Optional(),
		field.Enum("task_vod_download_thumbnail").GoType(utils.TaskStatus("")).Default(string(utils.Pending)).Optional(),
		field.Enum("task_vod_save_info").GoType(utils.TaskStatus("")).Default(string(utils.Pending)).Optional(),
		field.Enum("task_video_download").GoType(utils.TaskStatus("")).Default(string(utils.Pending)).Optional(),
		field.Enum("task_video_convert").GoType(utils.TaskStatus("")).Default(string(utils.Pending)).Optional(),
		field.Enum("task_video_move").GoType(utils.TaskStatus("")).Default(string(utils.Pending)).Optional(),
		field.Enum("task_chat_download").GoType(utils.TaskStatus("")).Default(string(utils.Pending)).Optional(),
		field.Enum("task_chat_convert").GoType(utils.TaskStatus("")).Default(string(utils.Pending)).Optional(),
		field.Enum("task_chat_render").GoType(utils.TaskStatus("")).Default(string(utils.Pending)).Optional(),
		field.Enum("task_chat_move").GoType(utils.TaskStatus("")).Default(string(utils.Pending)).Optional(),
		field.Time("chat_start").Optional(),
		field.Bool("archive_chat").Optional().Default(true),
		field.Bool("render_chat").Optional().Default(true),
		field.String("workflow_id").Optional(),
		field.String("workflow_run_id").Optional(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the Queue.
func (Queue) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("vod", Vod.Type).Ref("queue").Unique().Required(),
	}
}
