package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/zibbp/ganymede/internal/utils"
)

// Vod holds the schema definition for the Vod entity.
type Vod struct {
	ent.Schema
}

// Fields of the Vod.
func (Vod) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("ext_id").Comment("The ID of the video on the external platform."),
		field.String("clip_ext_vod_id").Optional().Comment("The external VOD ID of a clip. This is only populated if the clip is linked to a video."),
		field.String("ext_stream_id").Optional().Comment("The ID of the stream on the external platform, if applicable."),
		field.Enum("platform").GoType(utils.VideoPlatform("")).Default(string(utils.PlatformTwitch)).Comment("The platform the VOD is from, takes an enum."),
		field.Enum("type").GoType(utils.VodType("")).Default(string(utils.Archive)).Comment("The type of VOD, takes an enum."),
		field.String("title"),
		field.Int("duration").Default(1),
		field.Int("clip_vod_offset").Optional().Comment("The offset in seconds to where the clip starts in the VOD. This is only populdated if the video is a clip."),
		field.Int("views").Default(1),
		field.String("resolution").Optional(),
		field.Bool("processing").Default(false).Comment("Whether the VOD is currently processing."),
		field.String("thumbnail_path").Optional(),
		field.String("web_thumbnail_path"),
		field.String("video_path"),
		field.String("video_hls_path").Optional().Comment("The path where the video hls files are"),
		field.String("chat_path").Optional(),
		field.String("live_chat_path").Optional().Comment("Path to the raw live chat file"),
		field.String("live_chat_convert_path").Optional().Comment("Path to the converted live chat file"),
		field.String("chat_video_path").Optional(),
		field.String("info_path").Optional(),
		field.String("caption_path").Optional(),
		field.String("folder_name").Optional(),
		field.String("file_name").Optional(),
		field.String("tmp_video_download_path").Optional().Comment("The path where the video is downloaded to"),
		field.String("tmp_video_convert_path").Optional().Comment("The path where the converted video is"),
		field.String("tmp_chat_download_path").Optional().Comment("The path where the chat is downloaded to"),
		field.String("tmp_live_chat_download_path").Optional().Comment("The path where the converted chat is"),
		field.String("tmp_live_chat_convert_path").Optional().Comment("The path where the converted chat is"),
		field.String("tmp_chat_render_path").Optional().Comment("The path where the rendered chat is"),
		field.String("tmp_video_hls_path").Optional().Comment("The path where the temporary video hls files are"),
		field.Bool("locked").Default(false),
		field.Int("local_views").Default(0),
		field.Bool("sprite_thumbnails_enabled").Default(false),
		field.Strings("sprite_thumbnails_images").Optional(),
		field.Int("sprite_thumbnails_interval").Optional(),
		field.Int("sprite_thumbnails_width").Optional(),
		field.Int("sprite_thumbnails_height").Optional(),
		field.Int("sprite_thumbnails_rows").Optional(),
		field.Int("sprite_thumbnails_columns").Optional(),
		field.Int64("storage_size_bytes").Default(0).Comment("The size of the VOD in bytes."),
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
		edge.From("playlists", Playlist.Type).Ref("vods"),
		edge.To("chapters", Chapter.Type),
		edge.To("muted_segments", MutedSegment.Type),
		edge.From("multistream_info", MultistreamInfo.Type).Ref("vod"),
	}
}
