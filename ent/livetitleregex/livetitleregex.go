// Code generated by ent, DO NOT EDIT.

package livetitleregex

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/google/uuid"
)

const (
	// Label holds the string label denoting the livetitleregex type in the database.
	Label = "live_title_regex"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldNegative holds the string denoting the negative field in the database.
	FieldNegative = "negative"
	// FieldRegex holds the string denoting the regex field in the database.
	FieldRegex = "regex"
	// FieldApplyToVideos holds the string denoting the apply_to_videos field in the database.
	FieldApplyToVideos = "apply_to_videos"
	// EdgeLive holds the string denoting the live edge name in mutations.
	EdgeLive = "live"
	// Table holds the table name of the livetitleregex in the database.
	Table = "live_title_regexes"
	// LiveTable is the table that holds the live relation/edge.
	LiveTable = "live_title_regexes"
	// LiveInverseTable is the table name for the Live entity.
	// It exists in this package in order to avoid circular dependency with the "live" package.
	LiveInverseTable = "lives"
	// LiveColumn is the table column denoting the live relation/edge.
	LiveColumn = "live_id"
)

// Columns holds all SQL columns for livetitleregex fields.
var Columns = []string{
	FieldID,
	FieldNegative,
	FieldRegex,
	FieldApplyToVideos,
}

// ForeignKeys holds the SQL foreign-keys that are owned by the "live_title_regexes"
// table and are not defined as standalone fields in the schema.
var ForeignKeys = []string{
	"live_id",
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
	// DefaultNegative holds the default value on creation for the "negative" field.
	DefaultNegative bool
	// DefaultApplyToVideos holds the default value on creation for the "apply_to_videos" field.
	DefaultApplyToVideos bool
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() uuid.UUID
)

// OrderOption defines the ordering options for the LiveTitleRegex queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByNegative orders the results by the negative field.
func ByNegative(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldNegative, opts...).ToFunc()
}

// ByRegex orders the results by the regex field.
func ByRegex(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldRegex, opts...).ToFunc()
}

// ByApplyToVideos orders the results by the apply_to_videos field.
func ByApplyToVideos(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldApplyToVideos, opts...).ToFunc()
}

// ByLiveField orders the results by live field.
func ByLiveField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newLiveStep(), sql.OrderByField(field, opts...))
	}
}
func newLiveStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(LiveInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, LiveTable, LiveColumn),
	)
}
