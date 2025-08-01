// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/zibbp/ganymede/ent/playlistrule"
	"github.com/zibbp/ganymede/ent/playlistrulegroup"
	"github.com/zibbp/ganymede/internal/utils"
)

// PlaylistRuleCreate is the builder for creating a PlaylistRule entity.
type PlaylistRuleCreate struct {
	config
	mutation *PlaylistRuleMutation
	hooks    []Hook
	conflict []sql.ConflictOption
}

// SetName sets the "name" field.
func (prc *PlaylistRuleCreate) SetName(s string) *PlaylistRuleCreate {
	prc.mutation.SetName(s)
	return prc
}

// SetNillableName sets the "name" field if the given value is not nil.
func (prc *PlaylistRuleCreate) SetNillableName(s *string) *PlaylistRuleCreate {
	if s != nil {
		prc.SetName(*s)
	}
	return prc
}

// SetField sets the "field" field.
func (prc *PlaylistRuleCreate) SetField(urf utils.PlaylistRuleField) *PlaylistRuleCreate {
	prc.mutation.SetFieldField(urf)
	return prc
}

// SetNillableField sets the "field" field if the given value is not nil.
func (prc *PlaylistRuleCreate) SetNillableField(urf *utils.PlaylistRuleField) *PlaylistRuleCreate {
	if urf != nil {
		prc.SetField(*urf)
	}
	return prc
}

// SetOperator sets the "operator" field.
func (prc *PlaylistRuleCreate) SetOperator(uro utils.PlaylistRuleOperator) *PlaylistRuleCreate {
	prc.mutation.SetOperator(uro)
	return prc
}

// SetNillableOperator sets the "operator" field if the given value is not nil.
func (prc *PlaylistRuleCreate) SetNillableOperator(uro *utils.PlaylistRuleOperator) *PlaylistRuleCreate {
	if uro != nil {
		prc.SetOperator(*uro)
	}
	return prc
}

// SetValue sets the "value" field.
func (prc *PlaylistRuleCreate) SetValue(s string) *PlaylistRuleCreate {
	prc.mutation.SetValue(s)
	return prc
}

// SetPosition sets the "position" field.
func (prc *PlaylistRuleCreate) SetPosition(i int) *PlaylistRuleCreate {
	prc.mutation.SetPosition(i)
	return prc
}

// SetNillablePosition sets the "position" field if the given value is not nil.
func (prc *PlaylistRuleCreate) SetNillablePosition(i *int) *PlaylistRuleCreate {
	if i != nil {
		prc.SetPosition(*i)
	}
	return prc
}

// SetEnabled sets the "enabled" field.
func (prc *PlaylistRuleCreate) SetEnabled(b bool) *PlaylistRuleCreate {
	prc.mutation.SetEnabled(b)
	return prc
}

// SetNillableEnabled sets the "enabled" field if the given value is not nil.
func (prc *PlaylistRuleCreate) SetNillableEnabled(b *bool) *PlaylistRuleCreate {
	if b != nil {
		prc.SetEnabled(*b)
	}
	return prc
}

// SetID sets the "id" field.
func (prc *PlaylistRuleCreate) SetID(u uuid.UUID) *PlaylistRuleCreate {
	prc.mutation.SetID(u)
	return prc
}

// SetNillableID sets the "id" field if the given value is not nil.
func (prc *PlaylistRuleCreate) SetNillableID(u *uuid.UUID) *PlaylistRuleCreate {
	if u != nil {
		prc.SetID(*u)
	}
	return prc
}

// SetGroupID sets the "group" edge to the PlaylistRuleGroup entity by ID.
func (prc *PlaylistRuleCreate) SetGroupID(id uuid.UUID) *PlaylistRuleCreate {
	prc.mutation.SetGroupID(id)
	return prc
}

// SetGroup sets the "group" edge to the PlaylistRuleGroup entity.
func (prc *PlaylistRuleCreate) SetGroup(p *PlaylistRuleGroup) *PlaylistRuleCreate {
	return prc.SetGroupID(p.ID)
}

// Mutation returns the PlaylistRuleMutation object of the builder.
func (prc *PlaylistRuleCreate) Mutation() *PlaylistRuleMutation {
	return prc.mutation
}

// Save creates the PlaylistRule in the database.
func (prc *PlaylistRuleCreate) Save(ctx context.Context) (*PlaylistRule, error) {
	prc.defaults()
	return withHooks(ctx, prc.sqlSave, prc.mutation, prc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (prc *PlaylistRuleCreate) SaveX(ctx context.Context) *PlaylistRule {
	v, err := prc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (prc *PlaylistRuleCreate) Exec(ctx context.Context) error {
	_, err := prc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (prc *PlaylistRuleCreate) ExecX(ctx context.Context) {
	if err := prc.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (prc *PlaylistRuleCreate) defaults() {
	if _, ok := prc.mutation.GetField(); !ok {
		v := playlistrule.DefaultField
		prc.mutation.SetFieldField(v)
	}
	if _, ok := prc.mutation.Operator(); !ok {
		v := playlistrule.DefaultOperator
		prc.mutation.SetOperator(v)
	}
	if _, ok := prc.mutation.Position(); !ok {
		v := playlistrule.DefaultPosition
		prc.mutation.SetPosition(v)
	}
	if _, ok := prc.mutation.Enabled(); !ok {
		v := playlistrule.DefaultEnabled
		prc.mutation.SetEnabled(v)
	}
	if _, ok := prc.mutation.ID(); !ok {
		v := playlistrule.DefaultID()
		prc.mutation.SetID(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (prc *PlaylistRuleCreate) check() error {
	if _, ok := prc.mutation.GetField(); !ok {
		return &ValidationError{Name: "field", err: errors.New(`ent: missing required field "PlaylistRule.field"`)}
	}
	if v, ok := prc.mutation.GetField(); ok {
		if err := playlistrule.FieldValidator(v); err != nil {
			return &ValidationError{Name: "field", err: fmt.Errorf(`ent: validator failed for field "PlaylistRule.field": %w`, err)}
		}
	}
	if _, ok := prc.mutation.Operator(); !ok {
		return &ValidationError{Name: "operator", err: errors.New(`ent: missing required field "PlaylistRule.operator"`)}
	}
	if v, ok := prc.mutation.Operator(); ok {
		if err := playlistrule.OperatorValidator(v); err != nil {
			return &ValidationError{Name: "operator", err: fmt.Errorf(`ent: validator failed for field "PlaylistRule.operator": %w`, err)}
		}
	}
	if _, ok := prc.mutation.Value(); !ok {
		return &ValidationError{Name: "value", err: errors.New(`ent: missing required field "PlaylistRule.value"`)}
	}
	if _, ok := prc.mutation.Position(); !ok {
		return &ValidationError{Name: "position", err: errors.New(`ent: missing required field "PlaylistRule.position"`)}
	}
	if _, ok := prc.mutation.Enabled(); !ok {
		return &ValidationError{Name: "enabled", err: errors.New(`ent: missing required field "PlaylistRule.enabled"`)}
	}
	if len(prc.mutation.GroupIDs()) == 0 {
		return &ValidationError{Name: "group", err: errors.New(`ent: missing required edge "PlaylistRule.group"`)}
	}
	return nil
}

func (prc *PlaylistRuleCreate) sqlSave(ctx context.Context) (*PlaylistRule, error) {
	if err := prc.check(); err != nil {
		return nil, err
	}
	_node, _spec := prc.createSpec()
	if err := sqlgraph.CreateNode(ctx, prc.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	if _spec.ID.Value != nil {
		if id, ok := _spec.ID.Value.(*uuid.UUID); ok {
			_node.ID = *id
		} else if err := _node.ID.Scan(_spec.ID.Value); err != nil {
			return nil, err
		}
	}
	prc.mutation.id = &_node.ID
	prc.mutation.done = true
	return _node, nil
}

func (prc *PlaylistRuleCreate) createSpec() (*PlaylistRule, *sqlgraph.CreateSpec) {
	var (
		_node = &PlaylistRule{config: prc.config}
		_spec = sqlgraph.NewCreateSpec(playlistrule.Table, sqlgraph.NewFieldSpec(playlistrule.FieldID, field.TypeUUID))
	)
	_spec.OnConflict = prc.conflict
	if id, ok := prc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = &id
	}
	if value, ok := prc.mutation.Name(); ok {
		_spec.SetField(playlistrule.FieldName, field.TypeString, value)
		_node.Name = value
	}
	if value, ok := prc.mutation.GetField(); ok {
		_spec.SetField(playlistrule.FieldField, field.TypeEnum, value)
		_node.Field = value
	}
	if value, ok := prc.mutation.Operator(); ok {
		_spec.SetField(playlistrule.FieldOperator, field.TypeEnum, value)
		_node.Operator = value
	}
	if value, ok := prc.mutation.Value(); ok {
		_spec.SetField(playlistrule.FieldValue, field.TypeString, value)
		_node.Value = value
	}
	if value, ok := prc.mutation.Position(); ok {
		_spec.SetField(playlistrule.FieldPosition, field.TypeInt, value)
		_node.Position = value
	}
	if value, ok := prc.mutation.Enabled(); ok {
		_spec.SetField(playlistrule.FieldEnabled, field.TypeBool, value)
		_node.Enabled = value
	}
	if nodes := prc.mutation.GroupIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   playlistrule.GroupTable,
			Columns: []string{playlistrule.GroupColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: sqlgraph.NewFieldSpec(playlistrulegroup.FieldID, field.TypeUUID),
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_node.playlist_rule_group_rules = &nodes[0]
		_spec.Edges = append(_spec.Edges, edge)
	}
	return _node, _spec
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.PlaylistRule.Create().
//		SetName(v).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.PlaylistRuleUpsert) {
//			SetName(v+v).
//		}).
//		Exec(ctx)
func (prc *PlaylistRuleCreate) OnConflict(opts ...sql.ConflictOption) *PlaylistRuleUpsertOne {
	prc.conflict = opts
	return &PlaylistRuleUpsertOne{
		create: prc,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.PlaylistRule.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (prc *PlaylistRuleCreate) OnConflictColumns(columns ...string) *PlaylistRuleUpsertOne {
	prc.conflict = append(prc.conflict, sql.ConflictColumns(columns...))
	return &PlaylistRuleUpsertOne{
		create: prc,
	}
}

type (
	// PlaylistRuleUpsertOne is the builder for "upsert"-ing
	//  one PlaylistRule node.
	PlaylistRuleUpsertOne struct {
		create *PlaylistRuleCreate
	}

	// PlaylistRuleUpsert is the "OnConflict" setter.
	PlaylistRuleUpsert struct {
		*sql.UpdateSet
	}
)

// SetName sets the "name" field.
func (u *PlaylistRuleUpsert) SetName(v string) *PlaylistRuleUpsert {
	u.Set(playlistrule.FieldName, v)
	return u
}

// UpdateName sets the "name" field to the value that was provided on create.
func (u *PlaylistRuleUpsert) UpdateName() *PlaylistRuleUpsert {
	u.SetExcluded(playlistrule.FieldName)
	return u
}

// ClearName clears the value of the "name" field.
func (u *PlaylistRuleUpsert) ClearName() *PlaylistRuleUpsert {
	u.SetNull(playlistrule.FieldName)
	return u
}

// SetField sets the "field" field.
func (u *PlaylistRuleUpsert) SetField(v utils.PlaylistRuleField) *PlaylistRuleUpsert {
	u.Set(playlistrule.FieldField, v)
	return u
}

// UpdateField sets the "field" field to the value that was provided on create.
func (u *PlaylistRuleUpsert) UpdateField() *PlaylistRuleUpsert {
	u.SetExcluded(playlistrule.FieldField)
	return u
}

// SetOperator sets the "operator" field.
func (u *PlaylistRuleUpsert) SetOperator(v utils.PlaylistRuleOperator) *PlaylistRuleUpsert {
	u.Set(playlistrule.FieldOperator, v)
	return u
}

// UpdateOperator sets the "operator" field to the value that was provided on create.
func (u *PlaylistRuleUpsert) UpdateOperator() *PlaylistRuleUpsert {
	u.SetExcluded(playlistrule.FieldOperator)
	return u
}

// SetValue sets the "value" field.
func (u *PlaylistRuleUpsert) SetValue(v string) *PlaylistRuleUpsert {
	u.Set(playlistrule.FieldValue, v)
	return u
}

// UpdateValue sets the "value" field to the value that was provided on create.
func (u *PlaylistRuleUpsert) UpdateValue() *PlaylistRuleUpsert {
	u.SetExcluded(playlistrule.FieldValue)
	return u
}

// SetPosition sets the "position" field.
func (u *PlaylistRuleUpsert) SetPosition(v int) *PlaylistRuleUpsert {
	u.Set(playlistrule.FieldPosition, v)
	return u
}

// UpdatePosition sets the "position" field to the value that was provided on create.
func (u *PlaylistRuleUpsert) UpdatePosition() *PlaylistRuleUpsert {
	u.SetExcluded(playlistrule.FieldPosition)
	return u
}

// AddPosition adds v to the "position" field.
func (u *PlaylistRuleUpsert) AddPosition(v int) *PlaylistRuleUpsert {
	u.Add(playlistrule.FieldPosition, v)
	return u
}

// SetEnabled sets the "enabled" field.
func (u *PlaylistRuleUpsert) SetEnabled(v bool) *PlaylistRuleUpsert {
	u.Set(playlistrule.FieldEnabled, v)
	return u
}

// UpdateEnabled sets the "enabled" field to the value that was provided on create.
func (u *PlaylistRuleUpsert) UpdateEnabled() *PlaylistRuleUpsert {
	u.SetExcluded(playlistrule.FieldEnabled)
	return u
}

// UpdateNewValues updates the mutable fields using the new values that were set on create except the ID field.
// Using this option is equivalent to using:
//
//	client.PlaylistRule.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(playlistrule.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *PlaylistRuleUpsertOne) UpdateNewValues() *PlaylistRuleUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		if _, exists := u.create.mutation.ID(); exists {
			s.SetIgnore(playlistrule.FieldID)
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.PlaylistRule.Create().
//	    OnConflict(sql.ResolveWithIgnore()).
//	    Exec(ctx)
func (u *PlaylistRuleUpsertOne) Ignore() *PlaylistRuleUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *PlaylistRuleUpsertOne) DoNothing() *PlaylistRuleUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the PlaylistRuleCreate.OnConflict
// documentation for more info.
func (u *PlaylistRuleUpsertOne) Update(set func(*PlaylistRuleUpsert)) *PlaylistRuleUpsertOne {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&PlaylistRuleUpsert{UpdateSet: update})
	}))
	return u
}

// SetName sets the "name" field.
func (u *PlaylistRuleUpsertOne) SetName(v string) *PlaylistRuleUpsertOne {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.SetName(v)
	})
}

// UpdateName sets the "name" field to the value that was provided on create.
func (u *PlaylistRuleUpsertOne) UpdateName() *PlaylistRuleUpsertOne {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.UpdateName()
	})
}

// ClearName clears the value of the "name" field.
func (u *PlaylistRuleUpsertOne) ClearName() *PlaylistRuleUpsertOne {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.ClearName()
	})
}

// SetField sets the "field" field.
func (u *PlaylistRuleUpsertOne) SetField(v utils.PlaylistRuleField) *PlaylistRuleUpsertOne {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.SetField(v)
	})
}

// UpdateField sets the "field" field to the value that was provided on create.
func (u *PlaylistRuleUpsertOne) UpdateField() *PlaylistRuleUpsertOne {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.UpdateField()
	})
}

// SetOperator sets the "operator" field.
func (u *PlaylistRuleUpsertOne) SetOperator(v utils.PlaylistRuleOperator) *PlaylistRuleUpsertOne {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.SetOperator(v)
	})
}

// UpdateOperator sets the "operator" field to the value that was provided on create.
func (u *PlaylistRuleUpsertOne) UpdateOperator() *PlaylistRuleUpsertOne {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.UpdateOperator()
	})
}

// SetValue sets the "value" field.
func (u *PlaylistRuleUpsertOne) SetValue(v string) *PlaylistRuleUpsertOne {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.SetValue(v)
	})
}

// UpdateValue sets the "value" field to the value that was provided on create.
func (u *PlaylistRuleUpsertOne) UpdateValue() *PlaylistRuleUpsertOne {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.UpdateValue()
	})
}

// SetPosition sets the "position" field.
func (u *PlaylistRuleUpsertOne) SetPosition(v int) *PlaylistRuleUpsertOne {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.SetPosition(v)
	})
}

// AddPosition adds v to the "position" field.
func (u *PlaylistRuleUpsertOne) AddPosition(v int) *PlaylistRuleUpsertOne {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.AddPosition(v)
	})
}

// UpdatePosition sets the "position" field to the value that was provided on create.
func (u *PlaylistRuleUpsertOne) UpdatePosition() *PlaylistRuleUpsertOne {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.UpdatePosition()
	})
}

// SetEnabled sets the "enabled" field.
func (u *PlaylistRuleUpsertOne) SetEnabled(v bool) *PlaylistRuleUpsertOne {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.SetEnabled(v)
	})
}

// UpdateEnabled sets the "enabled" field to the value that was provided on create.
func (u *PlaylistRuleUpsertOne) UpdateEnabled() *PlaylistRuleUpsertOne {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.UpdateEnabled()
	})
}

// Exec executes the query.
func (u *PlaylistRuleUpsertOne) Exec(ctx context.Context) error {
	if len(u.create.conflict) == 0 {
		return errors.New("ent: missing options for PlaylistRuleCreate.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *PlaylistRuleUpsertOne) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}

// Exec executes the UPSERT query and returns the inserted/updated ID.
func (u *PlaylistRuleUpsertOne) ID(ctx context.Context) (id uuid.UUID, err error) {
	if u.create.driver.Dialect() == dialect.MySQL {
		// In case of "ON CONFLICT", there is no way to get back non-numeric ID
		// fields from the database since MySQL does not support the RETURNING clause.
		return id, errors.New("ent: PlaylistRuleUpsertOne.ID is not supported by MySQL driver. Use PlaylistRuleUpsertOne.Exec instead")
	}
	node, err := u.create.Save(ctx)
	if err != nil {
		return id, err
	}
	return node.ID, nil
}

// IDX is like ID, but panics if an error occurs.
func (u *PlaylistRuleUpsertOne) IDX(ctx context.Context) uuid.UUID {
	id, err := u.ID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// PlaylistRuleCreateBulk is the builder for creating many PlaylistRule entities in bulk.
type PlaylistRuleCreateBulk struct {
	config
	err      error
	builders []*PlaylistRuleCreate
	conflict []sql.ConflictOption
}

// Save creates the PlaylistRule entities in the database.
func (prcb *PlaylistRuleCreateBulk) Save(ctx context.Context) ([]*PlaylistRule, error) {
	if prcb.err != nil {
		return nil, prcb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(prcb.builders))
	nodes := make([]*PlaylistRule, len(prcb.builders))
	mutators := make([]Mutator, len(prcb.builders))
	for i := range prcb.builders {
		func(i int, root context.Context) {
			builder := prcb.builders[i]
			builder.defaults()
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*PlaylistRuleMutation)
				if !ok {
					return nil, fmt.Errorf("unexpected mutation type %T", m)
				}
				if err := builder.check(); err != nil {
					return nil, err
				}
				builder.mutation = mutation
				var err error
				nodes[i], specs[i] = builder.createSpec()
				if i < len(mutators)-1 {
					_, err = mutators[i+1].Mutate(root, prcb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					spec.OnConflict = prcb.conflict
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, prcb.driver, spec); err != nil {
						if sqlgraph.IsConstraintError(err) {
							err = &ConstraintError{msg: err.Error(), wrap: err}
						}
					}
				}
				if err != nil {
					return nil, err
				}
				mutation.id = &nodes[i].ID
				mutation.done = true
				return nodes[i], nil
			})
			for i := len(builder.hooks) - 1; i >= 0; i-- {
				mut = builder.hooks[i](mut)
			}
			mutators[i] = mut
		}(i, ctx)
	}
	if len(mutators) > 0 {
		if _, err := mutators[0].Mutate(ctx, prcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (prcb *PlaylistRuleCreateBulk) SaveX(ctx context.Context) []*PlaylistRule {
	v, err := prcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (prcb *PlaylistRuleCreateBulk) Exec(ctx context.Context) error {
	_, err := prcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (prcb *PlaylistRuleCreateBulk) ExecX(ctx context.Context) {
	if err := prcb.Exec(ctx); err != nil {
		panic(err)
	}
}

// OnConflict allows configuring the `ON CONFLICT` / `ON DUPLICATE KEY` clause
// of the `INSERT` statement. For example:
//
//	client.PlaylistRule.CreateBulk(builders...).
//		OnConflict(
//			// Update the row with the new values
//			// the was proposed for insertion.
//			sql.ResolveWithNewValues(),
//		).
//		// Override some of the fields with custom
//		// update values.
//		Update(func(u *ent.PlaylistRuleUpsert) {
//			SetName(v+v).
//		}).
//		Exec(ctx)
func (prcb *PlaylistRuleCreateBulk) OnConflict(opts ...sql.ConflictOption) *PlaylistRuleUpsertBulk {
	prcb.conflict = opts
	return &PlaylistRuleUpsertBulk{
		create: prcb,
	}
}

// OnConflictColumns calls `OnConflict` and configures the columns
// as conflict target. Using this option is equivalent to using:
//
//	client.PlaylistRule.Create().
//		OnConflict(sql.ConflictColumns(columns...)).
//		Exec(ctx)
func (prcb *PlaylistRuleCreateBulk) OnConflictColumns(columns ...string) *PlaylistRuleUpsertBulk {
	prcb.conflict = append(prcb.conflict, sql.ConflictColumns(columns...))
	return &PlaylistRuleUpsertBulk{
		create: prcb,
	}
}

// PlaylistRuleUpsertBulk is the builder for "upsert"-ing
// a bulk of PlaylistRule nodes.
type PlaylistRuleUpsertBulk struct {
	create *PlaylistRuleCreateBulk
}

// UpdateNewValues updates the mutable fields using the new values that
// were set on create. Using this option is equivalent to using:
//
//	client.PlaylistRule.Create().
//		OnConflict(
//			sql.ResolveWithNewValues(),
//			sql.ResolveWith(func(u *sql.UpdateSet) {
//				u.SetIgnore(playlistrule.FieldID)
//			}),
//		).
//		Exec(ctx)
func (u *PlaylistRuleUpsertBulk) UpdateNewValues() *PlaylistRuleUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithNewValues())
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(s *sql.UpdateSet) {
		for _, b := range u.create.builders {
			if _, exists := b.mutation.ID(); exists {
				s.SetIgnore(playlistrule.FieldID)
			}
		}
	}))
	return u
}

// Ignore sets each column to itself in case of conflict.
// Using this option is equivalent to using:
//
//	client.PlaylistRule.Create().
//		OnConflict(sql.ResolveWithIgnore()).
//		Exec(ctx)
func (u *PlaylistRuleUpsertBulk) Ignore() *PlaylistRuleUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWithIgnore())
	return u
}

// DoNothing configures the conflict_action to `DO NOTHING`.
// Supported only by SQLite and PostgreSQL.
func (u *PlaylistRuleUpsertBulk) DoNothing() *PlaylistRuleUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.DoNothing())
	return u
}

// Update allows overriding fields `UPDATE` values. See the PlaylistRuleCreateBulk.OnConflict
// documentation for more info.
func (u *PlaylistRuleUpsertBulk) Update(set func(*PlaylistRuleUpsert)) *PlaylistRuleUpsertBulk {
	u.create.conflict = append(u.create.conflict, sql.ResolveWith(func(update *sql.UpdateSet) {
		set(&PlaylistRuleUpsert{UpdateSet: update})
	}))
	return u
}

// SetName sets the "name" field.
func (u *PlaylistRuleUpsertBulk) SetName(v string) *PlaylistRuleUpsertBulk {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.SetName(v)
	})
}

// UpdateName sets the "name" field to the value that was provided on create.
func (u *PlaylistRuleUpsertBulk) UpdateName() *PlaylistRuleUpsertBulk {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.UpdateName()
	})
}

// ClearName clears the value of the "name" field.
func (u *PlaylistRuleUpsertBulk) ClearName() *PlaylistRuleUpsertBulk {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.ClearName()
	})
}

// SetField sets the "field" field.
func (u *PlaylistRuleUpsertBulk) SetField(v utils.PlaylistRuleField) *PlaylistRuleUpsertBulk {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.SetField(v)
	})
}

// UpdateField sets the "field" field to the value that was provided on create.
func (u *PlaylistRuleUpsertBulk) UpdateField() *PlaylistRuleUpsertBulk {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.UpdateField()
	})
}

// SetOperator sets the "operator" field.
func (u *PlaylistRuleUpsertBulk) SetOperator(v utils.PlaylistRuleOperator) *PlaylistRuleUpsertBulk {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.SetOperator(v)
	})
}

// UpdateOperator sets the "operator" field to the value that was provided on create.
func (u *PlaylistRuleUpsertBulk) UpdateOperator() *PlaylistRuleUpsertBulk {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.UpdateOperator()
	})
}

// SetValue sets the "value" field.
func (u *PlaylistRuleUpsertBulk) SetValue(v string) *PlaylistRuleUpsertBulk {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.SetValue(v)
	})
}

// UpdateValue sets the "value" field to the value that was provided on create.
func (u *PlaylistRuleUpsertBulk) UpdateValue() *PlaylistRuleUpsertBulk {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.UpdateValue()
	})
}

// SetPosition sets the "position" field.
func (u *PlaylistRuleUpsertBulk) SetPosition(v int) *PlaylistRuleUpsertBulk {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.SetPosition(v)
	})
}

// AddPosition adds v to the "position" field.
func (u *PlaylistRuleUpsertBulk) AddPosition(v int) *PlaylistRuleUpsertBulk {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.AddPosition(v)
	})
}

// UpdatePosition sets the "position" field to the value that was provided on create.
func (u *PlaylistRuleUpsertBulk) UpdatePosition() *PlaylistRuleUpsertBulk {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.UpdatePosition()
	})
}

// SetEnabled sets the "enabled" field.
func (u *PlaylistRuleUpsertBulk) SetEnabled(v bool) *PlaylistRuleUpsertBulk {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.SetEnabled(v)
	})
}

// UpdateEnabled sets the "enabled" field to the value that was provided on create.
func (u *PlaylistRuleUpsertBulk) UpdateEnabled() *PlaylistRuleUpsertBulk {
	return u.Update(func(s *PlaylistRuleUpsert) {
		s.UpdateEnabled()
	})
}

// Exec executes the query.
func (u *PlaylistRuleUpsertBulk) Exec(ctx context.Context) error {
	if u.create.err != nil {
		return u.create.err
	}
	for i, b := range u.create.builders {
		if len(b.conflict) != 0 {
			return fmt.Errorf("ent: OnConflict was set for builder %d. Set it on the PlaylistRuleCreateBulk instead", i)
		}
	}
	if len(u.create.conflict) == 0 {
		return errors.New("ent: missing options for PlaylistRuleCreateBulk.OnConflict")
	}
	return u.create.Exec(ctx)
}

// ExecX is like Exec, but panics if an error occurs.
func (u *PlaylistRuleUpsertBulk) ExecX(ctx context.Context) {
	if err := u.create.Exec(ctx); err != nil {
		panic(err)
	}
}
