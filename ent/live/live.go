// Code generated by ent, DO NOT EDIT.

package live

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the live type in the database.
	Label = "live"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldWatchLive holds the string denoting the watch_live field in the database.
	FieldWatchLive = "watch_live"
	// FieldWatchVod holds the string denoting the watch_vod field in the database.
	FieldWatchVod = "watch_vod"
	// FieldDownloadArchives holds the string denoting the download_archives field in the database.
	FieldDownloadArchives = "download_archives"
	// FieldDownloadHighlights holds the string denoting the download_highlights field in the database.
	FieldDownloadHighlights = "download_highlights"
	// FieldDownloadUploads holds the string denoting the download_uploads field in the database.
	FieldDownloadUploads = "download_uploads"
	// FieldDownloadSubOnly holds the string denoting the download_sub_only field in the database.
	FieldDownloadSubOnly = "download_sub_only"
	// FieldIsLive holds the string denoting the is_live field in the database.
	FieldIsLive = "is_live"
	// FieldArchiveChat holds the string denoting the archive_chat field in the database.
	FieldArchiveChat = "archive_chat"
	// FieldResolution holds the string denoting the resolution field in the database.
	FieldResolution = "resolution"
	// FieldLastLive holds the string denoting the last_live field in the database.
	FieldLastLive = "last_live"
	// FieldRenderChat holds the string denoting the render_chat field in the database.
	FieldRenderChat = "render_chat"
	// FieldVideoAge holds the string denoting the video_age field in the database.
	FieldVideoAge = "video_age"
	// FieldApplyCategoriesToLive holds the string denoting the apply_categories_to_live field in the database.
	FieldApplyCategoriesToLive = "apply_categories_to_live"
	// FieldClipsWatch holds the string denoting the clips_watch field in the database.
	FieldClipsWatch = "clips_watch"
	// FieldClipsLimit holds the string denoting the clips_limit field in the database.
	FieldClipsLimit = "clips_limit"
	// FieldClipsIntervalDays holds the string denoting the clips_interval_days field in the database.
	FieldClipsIntervalDays = "clips_interval_days"
	// FieldClipsLastChecked holds the string denoting the clips_last_checked field in the database.
	FieldClipsLastChecked = "clips_last_checked"
	// FieldUpdatedAt holds the string denoting the updated_at field in the database.
	FieldUpdatedAt = "updated_at"
	// FieldCreatedAt holds the string denoting the created_at field in the database.
	FieldCreatedAt = "created_at"
	// EdgeChannel holds the string denoting the channel edge name in mutations.
	EdgeChannel = "channel"
	// EdgeCategories holds the string denoting the categories edge name in mutations.
	EdgeCategories = "categories"
	// EdgeTitleRegex holds the string denoting the title_regex edge name in mutations.
	EdgeTitleRegex = "title_regex"
	// Table holds the table name of the live in the database.
	Table = "lives"
	// ChannelTable is the table that holds the channel relation/edge.
	ChannelTable = "lives"
	// ChannelInverseTable is the table name for the Channel entity.
	// It exists in this package in order to avoid circular dependency with the "channel" package.
	ChannelInverseTable = "channels"
	// ChannelColumn is the table column denoting the channel relation/edge.
	ChannelColumn = "channel_live"
	// CategoriesTable is the table that holds the categories relation/edge.
	CategoriesTable = "live_categories"
	// CategoriesInverseTable is the table name for the LiveCategory entity.
	// It exists in this package in order to avoid circular dependency with the "livecategory" package.
	CategoriesInverseTable = "live_categories"
	// CategoriesColumn is the table column denoting the categories relation/edge.
	CategoriesColumn = "live_id"
	// TitleRegexTable is the table that holds the title_regex relation/edge.
	TitleRegexTable = "live_title_regexes"
	// TitleRegexInverseTable is the table name for the LiveTitleRegex entity.
	// It exists in this package in order to avoid circular dependency with the "livetitleregex" package.
	TitleRegexInverseTable = "live_title_regexes"
	// TitleRegexColumn is the table column denoting the title_regex relation/edge.
	TitleRegexColumn = "live_id"
)

// Columns holds all SQL columns for live fields.
var Columns = []string{
	FieldID,
	FieldWatchLive,
	FieldWatchVod,
	FieldDownloadArchives,
	FieldDownloadHighlights,
	FieldDownloadUploads,
	FieldDownloadSubOnly,
	FieldIsLive,
	FieldArchiveChat,
	FieldResolution,
	FieldLastLive,
	FieldRenderChat,
	FieldVideoAge,
	FieldApplyCategoriesToLive,
	FieldClipsWatch,
	FieldClipsLimit,
	FieldClipsIntervalDays,
	FieldClipsLastChecked,
	FieldUpdatedAt,
	FieldCreatedAt,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "lives"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"channel_live",
}

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	for i := range ForeignKeys {
		if column == ForeignKeys[i] {
			return true
		}
	}
	return false
}

var (
	// DefaultWatchLive holds the default value on creation for the "watch_live" field.
	DefaultWatchLive bool
	// DefaultWatchVod holds the default value on creation for the "watch_vod" field.
	DefaultWatchVod bool
	// DefaultDownloadArchives holds the default value on creation for the "download_archives" field.
	DefaultDownloadArchives bool
	// DefaultDownloadHighlights holds the default value on creation for the "download_highlights" field.
	DefaultDownloadHighlights bool
	// DefaultDownloadUploads holds the default value on creation for the "download_uploads" field.
	DefaultDownloadUploads bool
	// DefaultDownloadSubOnly holds the default value on creation for the "download_sub_only" field.
	DefaultDownloadSubOnly bool
	// DefaultIsLive holds the default value on creation for the "is_live" field.
	DefaultIsLive bool
	// DefaultArchiveChat holds the default value on creation for the "archive_chat" field.
	DefaultArchiveChat bool
	// DefaultResolution holds the default value on creation for the "resolution" field.
	DefaultResolution string
	// DefaultLastLive holds the default value on creation for the "last_live" field.
	DefaultLastLive func() time.Time
	// DefaultRenderChat holds the default value on creation for the "render_chat" field.
	DefaultRenderChat bool
	// DefaultVideoAge holds the default value on creation for the "video_age" field.
	DefaultVideoAge int64
	// DefaultApplyCategoriesToLive holds the default value on creation for the "apply_categories_to_live" field.
	DefaultApplyCategoriesToLive bool
	// DefaultClipsWatch holds the default value on creation for the "clips_watch" field.
	DefaultClipsWatch bool
	// DefaultClipsLimit holds the default value on creation for the "clips_limit" field.
	DefaultClipsLimit int
	// DefaultClipsIntervalDays holds the default value on creation for the "clips_interval_days" field.
	DefaultClipsIntervalDays int
	// DefaultUpdatedAt holds the default value on creation for the "updated_at" field.
	DefaultUpdatedAt func() time.Time
	// UpdateDefaultUpdatedAt holds the default value on update for the "updated_at" field.
	UpdateDefaultUpdatedAt func() time.Time
	// DefaultCreatedAt holds the default value on creation for the "created_at" field.
	DefaultCreatedAt func() time.Time
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() uuid.UUID
)

// OrderOption defines the ordering options for the Live queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByWatchLive orders the results by the watch_live field.
func ByWatchLive(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldWatchLive, opts...).ToFunc()
}

// ByWatchVod orders the results by the watch_vod field.
func ByWatchVod(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldWatchVod, opts...).ToFunc()
}

// ByDownloadArchives orders the results by the download_archives field.
func ByDownloadArchives(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDownloadArchives, opts...).ToFunc()
}

// ByDownloadHighlights orders the results by the download_highlights field.
func ByDownloadHighlights(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDownloadHighlights, opts...).ToFunc()
}

// ByDownloadUploads orders the results by the download_uploads field.
func ByDownloadUploads(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDownloadUploads, opts...).ToFunc()
}

// ByDownloadSubOnly orders the results by the download_sub_only field.
func ByDownloadSubOnly(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDownloadSubOnly, opts...).ToFunc()
}

// ByIsLive orders the results by the is_live field.
func ByIsLive(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldIsLive, opts...).ToFunc()
}

// ByArchiveChat orders the results by the archive_chat field.
func ByArchiveChat(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldArchiveChat, opts...).ToFunc()
}

// ByResolution orders the results by the resolution field.
func ByResolution(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldResolution, opts...).ToFunc()
}

// ByLastLive orders the results by the last_live field.
func ByLastLive(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldLastLive, opts...).ToFunc()
}

// ByRenderChat orders the results by the render_chat field.
func ByRenderChat(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldRenderChat, opts...).ToFunc()
}

// ByVideoAge orders the results by the video_age field.
func ByVideoAge(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldVideoAge, opts...).ToFunc()
}

// ByApplyCategoriesToLive orders the results by the apply_categories_to_live field.
func ByApplyCategoriesToLive(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldApplyCategoriesToLive, opts...).ToFunc()
}

// ByClipsWatch orders the results by the clips_watch field.
func ByClipsWatch(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldClipsWatch, opts...).ToFunc()
}

// ByClipsLimit orders the results by the clips_limit field.
func ByClipsLimit(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldClipsLimit, opts...).ToFunc()
}

// ByClipsIntervalDays orders the results by the clips_interval_days field.
func ByClipsIntervalDays(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldClipsIntervalDays, opts...).ToFunc()
}

// ByClipsLastChecked orders the results by the clips_last_checked field.
func ByClipsLastChecked(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldClipsLastChecked, opts...).ToFunc()
}

// ByUpdatedAt orders the results by the updated_at field.
func ByUpdatedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldUpdatedAt, opts...).ToFunc()
}

// ByCreatedAt orders the results by the created_at field.
func ByCreatedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCreatedAt, opts...).ToFunc()
}

// ByChannelField orders the results by channel field.
func ByChannelField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newChannelStep(), sql.OrderByField(field, opts...))
	}
}

// ByCategoriesCount orders the results by categories count.
func ByCategoriesCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newCategoriesStep(), opts...)
	}
}

// ByCategories orders the results by categories terms.
func ByCategories(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newCategoriesStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// ByTitleRegexCount orders the results by title_regex count.
func ByTitleRegexCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newTitleRegexStep(), opts...)
	}
}

// ByTitleRegex orders the results by title_regex terms.
func ByTitleRegex(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newTitleRegexStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}
func newChannelStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(ChannelInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, ChannelTable, ChannelColumn),
	)
}
func newCategoriesStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(CategoriesInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, CategoriesTable, CategoriesColumn),
	)
}
func newTitleRegexStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(TitleRegexInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, TitleRegexTable, TitleRegexColumn),
	)
}
