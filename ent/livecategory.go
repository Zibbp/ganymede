// Code generated by ent, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/zibbp/ganymede/ent/live"
	"github.com/zibbp/ganymede/ent/livecategory"
)

// LiveCategory is the model entity for the LiveCategory schema.
type LiveCategory struct {
	config `json:"-"`
	// ID of the ent.
	ID uuid.UUID `json:"id,omitempty"`
	// Name holds the value of the "name" field.
	Name string `json:"name,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the LiveCategoryQuery when eager-loading is set.
	Edges   LiveCategoryEdges `json:"edges"`
	live_id *uuid.UUID
}

// LiveCategoryEdges holds the relations/edges for other nodes in the graph.
type LiveCategoryEdges struct {
	// Live holds the value of the live edge.
	Live *Live `json:"live,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [1]bool
}

// LiveOrErr returns the Live value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e LiveCategoryEdges) LiveOrErr() (*Live, error) {
	if e.loadedTypes[0] {
		if e.Live == nil {
			// Edge was loaded but was not found.
			return nil, &NotFoundError{label: live.Label}
		}
		return e.Live, nil
	}
	return nil, &NotLoadedError{edge: "live"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*LiveCategory) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case livecategory.FieldName:
			values[i] = new(sql.NullString)
		case livecategory.FieldID:
			values[i] = new(uuid.UUID)
		case livecategory.ForeignKeys[0]: // live_id
			values[i] = &sql.NullScanner{S: new(uuid.UUID)}
		default:
			return nil, fmt.Errorf("unexpected column %q for type LiveCategory", columns[i])
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the LiveCategory fields.
func (lc *LiveCategory) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case livecategory.FieldID:
			if value, ok := values[i].(*uuid.UUID); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value != nil {
				lc.ID = *value
			}
		case livecategory.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				lc.Name = value.String
			}
		case livecategory.ForeignKeys[0]:
			if value, ok := values[i].(*sql.NullScanner); !ok {
				return fmt.Errorf("unexpected type %T for field live_id", values[i])
			} else if value.Valid {
				lc.live_id = new(uuid.UUID)
				*lc.live_id = *value.S.(*uuid.UUID)
			}
		}
	}
	return nil
}

// QueryLive queries the "live" edge of the LiveCategory entity.
func (lc *LiveCategory) QueryLive() *LiveQuery {
	return NewLiveCategoryClient(lc.config).QueryLive(lc)
}

// Update returns a builder for updating this LiveCategory.
// Note that you need to call LiveCategory.Unwrap() before calling this method if this LiveCategory
// was returned from a transaction, and the transaction was committed or rolled back.
func (lc *LiveCategory) Update() *LiveCategoryUpdateOne {
	return NewLiveCategoryClient(lc.config).UpdateOne(lc)
}

// Unwrap unwraps the LiveCategory entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (lc *LiveCategory) Unwrap() *LiveCategory {
	_tx, ok := lc.config.driver.(*txDriver)
	if !ok {
		panic("ent: LiveCategory is not a transactional entity")
	}
	lc.config.driver = _tx.drv
	return lc
}

// String implements the fmt.Stringer.
func (lc *LiveCategory) String() string {
	var builder strings.Builder
	builder.WriteString("LiveCategory(")
	builder.WriteString(fmt.Sprintf("id=%v, ", lc.ID))
	builder.WriteString("name=")
	builder.WriteString(lc.Name)
	builder.WriteByte(')')
	return builder.String()
}

// LiveCategories is a parsable slice of LiveCategory.
type LiveCategories []*LiveCategory