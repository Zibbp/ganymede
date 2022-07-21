// Code generated by entc, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/zibbp/ganymede/ent/playback"
	"github.com/zibbp/ganymede/ent/predicate"
	"github.com/zibbp/ganymede/internal/utils"
)

// PlaybackUpdate is the builder for updating Playback entities.
type PlaybackUpdate struct {
	config
	hooks    []Hook
	mutation *PlaybackMutation
}

// Where appends a list predicates to the PlaybackUpdate builder.
func (pu *PlaybackUpdate) Where(ps ...predicate.Playback) *PlaybackUpdate {
	pu.mutation.Where(ps...)
	return pu
}

// SetVodID sets the "vod_id" field.
func (pu *PlaybackUpdate) SetVodID(u uuid.UUID) *PlaybackUpdate {
	pu.mutation.SetVodID(u)
	return pu
}

// SetUserID sets the "user_id" field.
func (pu *PlaybackUpdate) SetUserID(u uuid.UUID) *PlaybackUpdate {
	pu.mutation.SetUserID(u)
	return pu
}

// SetTime sets the "time" field.
func (pu *PlaybackUpdate) SetTime(i int) *PlaybackUpdate {
	pu.mutation.ResetTime()
	pu.mutation.SetTime(i)
	return pu
}

// SetNillableTime sets the "time" field if the given value is not nil.
func (pu *PlaybackUpdate) SetNillableTime(i *int) *PlaybackUpdate {
	if i != nil {
		pu.SetTime(*i)
	}
	return pu
}

// AddTime adds i to the "time" field.
func (pu *PlaybackUpdate) AddTime(i int) *PlaybackUpdate {
	pu.mutation.AddTime(i)
	return pu
}

// SetStatus sets the "status" field.
func (pu *PlaybackUpdate) SetStatus(us utils.PlaybackStatus) *PlaybackUpdate {
	pu.mutation.SetStatus(us)
	return pu
}

// SetNillableStatus sets the "status" field if the given value is not nil.
func (pu *PlaybackUpdate) SetNillableStatus(us *utils.PlaybackStatus) *PlaybackUpdate {
	if us != nil {
		pu.SetStatus(*us)
	}
	return pu
}

// ClearStatus clears the value of the "status" field.
func (pu *PlaybackUpdate) ClearStatus() *PlaybackUpdate {
	pu.mutation.ClearStatus()
	return pu
}

// SetUpdatedAt sets the "updated_at" field.
func (pu *PlaybackUpdate) SetUpdatedAt(t time.Time) *PlaybackUpdate {
	pu.mutation.SetUpdatedAt(t)
	return pu
}

// Mutation returns the PlaybackMutation object of the builder.
func (pu *PlaybackUpdate) Mutation() *PlaybackMutation {
	return pu.mutation
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (pu *PlaybackUpdate) Save(ctx context.Context) (int, error) {
	var (
		err      error
		affected int
	)
	pu.defaults()
	if len(pu.hooks) == 0 {
		if err = pu.check(); err != nil {
			return 0, err
		}
		affected, err = pu.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*PlaybackMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = pu.check(); err != nil {
				return 0, err
			}
			pu.mutation = mutation
			affected, err = pu.sqlSave(ctx)
			mutation.done = true
			return affected, err
		})
		for i := len(pu.hooks) - 1; i >= 0; i-- {
			if pu.hooks[i] == nil {
				return 0, fmt.Errorf("ent: uninitialized hook (forgotten import ent/runtime?)")
			}
			mut = pu.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, pu.mutation); err != nil {
			return 0, err
		}
	}
	return affected, err
}

// SaveX is like Save, but panics if an error occurs.
func (pu *PlaybackUpdate) SaveX(ctx context.Context) int {
	affected, err := pu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (pu *PlaybackUpdate) Exec(ctx context.Context) error {
	_, err := pu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (pu *PlaybackUpdate) ExecX(ctx context.Context) {
	if err := pu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (pu *PlaybackUpdate) defaults() {
	if _, ok := pu.mutation.UpdatedAt(); !ok {
		v := playback.UpdateDefaultUpdatedAt()
		pu.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (pu *PlaybackUpdate) check() error {
	if v, ok := pu.mutation.Status(); ok {
		if err := playback.StatusValidator(v); err != nil {
			return &ValidationError{Name: "status", err: fmt.Errorf(`ent: validator failed for field "Playback.status": %w`, err)}
		}
	}
	return nil
}

func (pu *PlaybackUpdate) sqlSave(ctx context.Context) (n int, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   playback.Table,
			Columns: playback.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: playback.FieldID,
			},
		},
	}
	if ps := pu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := pu.mutation.VodID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeUUID,
			Value:  value,
			Column: playback.FieldVodID,
		})
	}
	if value, ok := pu.mutation.UserID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeUUID,
			Value:  value,
			Column: playback.FieldUserID,
		})
	}
	if value, ok := pu.mutation.Time(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: playback.FieldTime,
		})
	}
	if value, ok := pu.mutation.AddedTime(); ok {
		_spec.Fields.Add = append(_spec.Fields.Add, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: playback.FieldTime,
		})
	}
	if value, ok := pu.mutation.Status(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: playback.FieldStatus,
		})
	}
	if pu.mutation.StatusCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Column: playback.FieldStatus,
		})
	}
	if value, ok := pu.mutation.UpdatedAt(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: playback.FieldUpdatedAt,
		})
	}
	if n, err = sqlgraph.UpdateNodes(ctx, pu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{playback.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{err.Error(), err}
		}
		return 0, err
	}
	return n, nil
}

// PlaybackUpdateOne is the builder for updating a single Playback entity.
type PlaybackUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *PlaybackMutation
}

// SetVodID sets the "vod_id" field.
func (puo *PlaybackUpdateOne) SetVodID(u uuid.UUID) *PlaybackUpdateOne {
	puo.mutation.SetVodID(u)
	return puo
}

// SetUserID sets the "user_id" field.
func (puo *PlaybackUpdateOne) SetUserID(u uuid.UUID) *PlaybackUpdateOne {
	puo.mutation.SetUserID(u)
	return puo
}

// SetTime sets the "time" field.
func (puo *PlaybackUpdateOne) SetTime(i int) *PlaybackUpdateOne {
	puo.mutation.ResetTime()
	puo.mutation.SetTime(i)
	return puo
}

// SetNillableTime sets the "time" field if the given value is not nil.
func (puo *PlaybackUpdateOne) SetNillableTime(i *int) *PlaybackUpdateOne {
	if i != nil {
		puo.SetTime(*i)
	}
	return puo
}

// AddTime adds i to the "time" field.
func (puo *PlaybackUpdateOne) AddTime(i int) *PlaybackUpdateOne {
	puo.mutation.AddTime(i)
	return puo
}

// SetStatus sets the "status" field.
func (puo *PlaybackUpdateOne) SetStatus(us utils.PlaybackStatus) *PlaybackUpdateOne {
	puo.mutation.SetStatus(us)
	return puo
}

// SetNillableStatus sets the "status" field if the given value is not nil.
func (puo *PlaybackUpdateOne) SetNillableStatus(us *utils.PlaybackStatus) *PlaybackUpdateOne {
	if us != nil {
		puo.SetStatus(*us)
	}
	return puo
}

// ClearStatus clears the value of the "status" field.
func (puo *PlaybackUpdateOne) ClearStatus() *PlaybackUpdateOne {
	puo.mutation.ClearStatus()
	return puo
}

// SetUpdatedAt sets the "updated_at" field.
func (puo *PlaybackUpdateOne) SetUpdatedAt(t time.Time) *PlaybackUpdateOne {
	puo.mutation.SetUpdatedAt(t)
	return puo
}

// Mutation returns the PlaybackMutation object of the builder.
func (puo *PlaybackUpdateOne) Mutation() *PlaybackMutation {
	return puo.mutation
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (puo *PlaybackUpdateOne) Select(field string, fields ...string) *PlaybackUpdateOne {
	puo.fields = append([]string{field}, fields...)
	return puo
}

// Save executes the query and returns the updated Playback entity.
func (puo *PlaybackUpdateOne) Save(ctx context.Context) (*Playback, error) {
	var (
		err  error
		node *Playback
	)
	puo.defaults()
	if len(puo.hooks) == 0 {
		if err = puo.check(); err != nil {
			return nil, err
		}
		node, err = puo.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*PlaybackMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = puo.check(); err != nil {
				return nil, err
			}
			puo.mutation = mutation
			node, err = puo.sqlSave(ctx)
			mutation.done = true
			return node, err
		})
		for i := len(puo.hooks) - 1; i >= 0; i-- {
			if puo.hooks[i] == nil {
				return nil, fmt.Errorf("ent: uninitialized hook (forgotten import ent/runtime?)")
			}
			mut = puo.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, puo.mutation); err != nil {
			return nil, err
		}
	}
	return node, err
}

// SaveX is like Save, but panics if an error occurs.
func (puo *PlaybackUpdateOne) SaveX(ctx context.Context) *Playback {
	node, err := puo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (puo *PlaybackUpdateOne) Exec(ctx context.Context) error {
	_, err := puo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (puo *PlaybackUpdateOne) ExecX(ctx context.Context) {
	if err := puo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (puo *PlaybackUpdateOne) defaults() {
	if _, ok := puo.mutation.UpdatedAt(); !ok {
		v := playback.UpdateDefaultUpdatedAt()
		puo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (puo *PlaybackUpdateOne) check() error {
	if v, ok := puo.mutation.Status(); ok {
		if err := playback.StatusValidator(v); err != nil {
			return &ValidationError{Name: "status", err: fmt.Errorf(`ent: validator failed for field "Playback.status": %w`, err)}
		}
	}
	return nil
}

func (puo *PlaybackUpdateOne) sqlSave(ctx context.Context) (_node *Playback, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   playback.Table,
			Columns: playback.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: playback.FieldID,
			},
		},
	}
	id, ok := puo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "Playback.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := puo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, playback.FieldID)
		for _, f := range fields {
			if !playback.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != playback.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := puo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := puo.mutation.VodID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeUUID,
			Value:  value,
			Column: playback.FieldVodID,
		})
	}
	if value, ok := puo.mutation.UserID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeUUID,
			Value:  value,
			Column: playback.FieldUserID,
		})
	}
	if value, ok := puo.mutation.Time(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: playback.FieldTime,
		})
	}
	if value, ok := puo.mutation.AddedTime(); ok {
		_spec.Fields.Add = append(_spec.Fields.Add, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: playback.FieldTime,
		})
	}
	if value, ok := puo.mutation.Status(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: playback.FieldStatus,
		})
	}
	if puo.mutation.StatusCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Column: playback.FieldStatus,
		})
	}
	if value, ok := puo.mutation.UpdatedAt(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: playback.FieldUpdatedAt,
		})
	}
	_node = &Playback{config: puo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, puo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{playback.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{err.Error(), err}
		}
		return nil, err
	}
	return _node, nil
}
