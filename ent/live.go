// Code generated by ent, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/zibbp/ganymede/ent/channel"
	"github.com/zibbp/ganymede/ent/live"
)

// Live is the model entity for the Live schema.
type Live struct {
	config `json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id"`
	// Watch live streams
	WatchLive bool `json:"watch_live"`
	// Watch new VODs
	WatchVod bool `json:"watch_vod"`
	// Download archives
	DownloadArchives bool `json:"download_archives"`
	// Download highlights
	DownloadHighlights bool `json:"download_highlights"`
	// Download uploads
	DownloadUploads bool `json:"download_uploads"`
	// Download sub only VODs
	DownloadSubOnly bool `json:"download_sub_only"`
	// Whether the channel is currently live.
	IsLive bool `json:"is_live"`
	// Whether the chat archive is enabled.
	ArchiveChat bool `json:"archive_chat"`
	// Resolution holds the value of the "resolution" field.
	Resolution string `json:"resolution"`
	// The time the channel last went live.
	LastLive time.Time `json:"last_live"`
	// Whether the chat should be rendered.
	RenderChat bool `json:"render_chat"`
	// Restrict fetching videos to a certain age.
	VideoAge int64 `json:"video_age"`
	// Whether the categories should be applied to livestreams.
	ApplyCategoriesToLive bool `json:"apply_categories_to_live"`
	// Whether to download clips on a schedule.
	WatchClips bool `json:"watch_clips"`
	// The number of clips to archive.
	ClipsLimit int `json:"clips_limit"`
	// How often channel should be checked for clips to archive in days.
	ClipsIntervalDays int `json:"clips_interval_days"`
	// Time when clips were last checked.
	ClipsLastChecked time.Time `json:"clips_last_checked"`
	// UpdatedAt holds the value of the "updated_at" field.
	UpdatedAt time.Time `json:"updated_at"`
	// CreatedAt holds the value of the "created_at" field.
	CreatedAt time.Time `json:"created_at"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the LiveQuery when eager-loading is set.
	Edges        LiveEdges `json:"edges"`
	channel_live *uuid.UUID
	selectValues sql.SelectValues
}

// LiveEdges holds the relations/edges for other nodes in the graph.
type LiveEdges struct {
	// Channel holds the value of the channel edge.
	Channel *Channel `json:"channel,omitempty"`
	// Categories holds the value of the categories edge.
	Categories []*LiveCategory `json:"categories,omitempty"`
	// TitleRegex holds the value of the title_regex edge.
	TitleRegex []*LiveTitleRegex `json:"title_regex,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [3]bool
}

// ChannelOrErr returns the Channel value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e LiveEdges) ChannelOrErr() (*Channel, error) {
	if e.Channel != nil {
		return e.Channel, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: channel.Label}
	}
	return nil, &NotLoadedError{edge: "channel"}
}

// CategoriesOrErr returns the Categories value or an error if the edge
// was not loaded in eager-loading.
func (e LiveEdges) CategoriesOrErr() ([]*LiveCategory, error) {
	if e.loadedTypes[1] {
		return e.Categories, nil
	}
	return nil, &NotLoadedError{edge: "categories"}
}

// TitleRegexOrErr returns the TitleRegex value or an error if the edge
// was not loaded in eager-loading.
func (e LiveEdges) TitleRegexOrErr() ([]*LiveTitleRegex, error) {
	if e.loadedTypes[2] {
		return e.TitleRegex, nil
	}
	return nil, &NotLoadedError{edge: "title_regex"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Live) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case live.FieldWatchLive, live.FieldWatchVod, live.FieldDownloadArchives, live.FieldDownloadHighlights, live.FieldDownloadUploads, live.FieldDownloadSubOnly, live.FieldIsLive, live.FieldArchiveChat, live.FieldRenderChat, live.FieldApplyCategoriesToLive, live.FieldWatchClips:
			values[i] = new(sql.NullBool)
		case live.FieldVideoAge, live.FieldClipsLimit, live.FieldClipsIntervalDays:
			values[i] = new(sql.NullInt64)
		case live.FieldResolution:
			values[i] = new(sql.NullString)
		case live.FieldLastLive, live.FieldClipsLastChecked, live.FieldUpdatedAt, live.FieldCreatedAt:
			values[i] = new(sql.NullTime)
		case live.FieldID:
			values[i] = new(uuid.UUID)
		case live.ForeignKeys[0]: // channel_live
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Live fields.
func (l *Live) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case live.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				l.ID = *value
			}
		case live.FieldWatchLive:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field watch_live", values[i])
			} else if value.Valid {
				l.WatchLive = value.Bool
			}
		case live.FieldWatchVod:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field watch_vod", values[i])
			} else if value.Valid {
				l.WatchVod = value.Bool
			}
		case live.FieldDownloadArchives:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field download_archives", values[i])
			} else if value.Valid {
				l.DownloadArchives = value.Bool
			}
		case live.FieldDownloadHighlights:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field download_highlights", values[i])
			} else if value.Valid {
				l.DownloadHighlights = value.Bool
			}
		case live.FieldDownloadUploads:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field download_uploads", values[i])
			} else if value.Valid {
				l.DownloadUploads = value.Bool
			}
		case live.FieldDownloadSubOnly:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field download_sub_only", values[i])
			} else if value.Valid {
				l.DownloadSubOnly = value.Bool
			}
		case live.FieldIsLive:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field is_live", values[i])
			} else if value.Valid {
				l.IsLive = value.Bool
			}
		case live.FieldArchiveChat:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field archive_chat", values[i])
			} else if value.Valid {
				l.ArchiveChat = value.Bool
			}
		case live.FieldResolution:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field resolution", values[i])
			} else if value.Valid {
				l.Resolution = value.String
			}
		case live.FieldLastLive:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field last_live", values[i])
			} else if value.Valid {
				l.LastLive = value.Time
			}
		case live.FieldRenderChat:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field render_chat", values[i])
			} else if value.Valid {
				l.RenderChat = value.Bool
			}
		case live.FieldVideoAge:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field video_age", values[i])
			} else if value.Valid {
				l.VideoAge = value.Int64
			}
		case live.FieldApplyCategoriesToLive:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field apply_categories_to_live", values[i])
			} else if value.Valid {
				l.ApplyCategoriesToLive = value.Bool
			}
		case live.FieldWatchClips:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field watch_clips", values[i])
			} else if value.Valid {
				l.WatchClips = value.Bool
			}
		case live.FieldClipsLimit:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field clips_limit", values[i])
			} else if value.Valid {
				l.ClipsLimit = int(value.Int64)
			}
		case live.FieldClipsIntervalDays:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field clips_interval_days", values[i])
			} else if value.Valid {
				l.ClipsIntervalDays = int(value.Int64)
			}
		case live.FieldClipsLastChecked:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field clips_last_checked", values[i])
			} else if value.Valid {
				l.ClipsLastChecked = value.Time
			}
		case live.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				l.UpdatedAt = value.Time
			}
		case live.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				l.CreatedAt = value.Time
			}
		case live.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field channel_live", values[i])
			} else if value.Valid {
				l.channel_live = new(uuid.UUID)
				*l.channel_live = *value.S.(*uuid.UUID)
			}
		default:
			l.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Live.
// This includes values selected through modifiers, order, etc.
func (l *Live) Value(name string) (ent.Value, error) {
	return l.selectValues.Get(name)
}

// QueryChannel queries the "channel" edge of the Live entity.
func (l *Live) QueryChannel() *ChannelQuery {
	return NewLiveClient(l.config).QueryChannel(l)
}

// QueryCategories queries the "categories" edge of the Live entity.
func (l *Live) QueryCategories() *LiveCategoryQuery {
	return NewLiveClient(l.config).QueryCategories(l)
}

// QueryTitleRegex queries the "title_regex" edge of the Live entity.
func (l *Live) QueryTitleRegex() *LiveTitleRegexQuery {
	return NewLiveClient(l.config).QueryTitleRegex(l)
}

// Update returns a builder for updating this Live.
// Note that you need to call Live.Unwrap() before calling this method if this Live
// was returned from a transaction, and the transaction was committed or rolled back.
func (l *Live) Update() *LiveUpdateOne {
	return NewLiveClient(l.config).UpdateOne(l)
}

// Unwrap unwraps the Live entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (l *Live) Unwrap() *Live {
	_tx, ok := l.config.driver.(*txDriver)
	if !ok {
		panic("ent: Live is not a transactional entity")
	}
	l.config.driver = _tx.drv
	return l
}

// String implements the fmt.Stringer.
func (l *Live) String() string {
	var builder strings.Builder
	builder.WriteString("Live(")
	builder.WriteString(fmt.Sprintf("id=%v, ", l.ID))
	builder.WriteString("watch_live=")
	builder.WriteString(fmt.Sprintf("%v", l.WatchLive))
	builder.WriteString(", ")
	builder.WriteString("watch_vod=")
	builder.WriteString(fmt.Sprintf("%v", l.WatchVod))
	builder.WriteString(", ")
	builder.WriteString("download_archives=")
	builder.WriteString(fmt.Sprintf("%v", l.DownloadArchives))
	builder.WriteString(", ")
	builder.WriteString("download_highlights=")
	builder.WriteString(fmt.Sprintf("%v", l.DownloadHighlights))
	builder.WriteString(", ")
	builder.WriteString("download_uploads=")
	builder.WriteString(fmt.Sprintf("%v", l.DownloadUploads))
	builder.WriteString(", ")
	builder.WriteString("download_sub_only=")
	builder.WriteString(fmt.Sprintf("%v", l.DownloadSubOnly))
	builder.WriteString(", ")
	builder.WriteString("is_live=")
	builder.WriteString(fmt.Sprintf("%v", l.IsLive))
	builder.WriteString(", ")
	builder.WriteString("archive_chat=")
	builder.WriteString(fmt.Sprintf("%v", l.ArchiveChat))
	builder.WriteString(", ")
	builder.WriteString("resolution=")
	builder.WriteString(l.Resolution)
	builder.WriteString(", ")
	builder.WriteString("last_live=")
	builder.WriteString(l.LastLive.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("render_chat=")
	builder.WriteString(fmt.Sprintf("%v", l.RenderChat))
	builder.WriteString(", ")
	builder.WriteString("video_age=")
	builder.WriteString(fmt.Sprintf("%v", l.VideoAge))
	builder.WriteString(", ")
	builder.WriteString("apply_categories_to_live=")
	builder.WriteString(fmt.Sprintf("%v", l.ApplyCategoriesToLive))
	builder.WriteString(", ")
	builder.WriteString("watch_clips=")
	builder.WriteString(fmt.Sprintf("%v", l.WatchClips))
	builder.WriteString(", ")
	builder.WriteString("clips_limit=")
	builder.WriteString(fmt.Sprintf("%v", l.ClipsLimit))
	builder.WriteString(", ")
	builder.WriteString("clips_interval_days=")
	builder.WriteString(fmt.Sprintf("%v", l.ClipsIntervalDays))
	builder.WriteString(", ")
	builder.WriteString("clips_last_checked=")
	builder.WriteString(l.ClipsLastChecked.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(l.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(l.CreatedAt.Format(time.ANSIC))
	builder.WriteByte(')')
	return builder.String()
}

// Lives is a parsable slice of Live.
type Lives []*Live
