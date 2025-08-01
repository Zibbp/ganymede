// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/zibbp/ganymede/ent/playlistrule"
	"github.com/zibbp/ganymede/ent/predicate"
)

// PlaylistRuleDelete is the builder for deleting a PlaylistRule entity.
type PlaylistRuleDelete struct {
	config
	hooks    []Hook
	mutation *PlaylistRuleMutation
}

// Where appends a list predicates to the PlaylistRuleDelete builder.
func (prd *PlaylistRuleDelete) Where(ps ...predicate.PlaylistRule) *PlaylistRuleDelete {
	prd.mutation.Where(ps...)
	return prd
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (prd *PlaylistRuleDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, prd.sqlExec, prd.mutation, prd.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (prd *PlaylistRuleDelete) ExecX(ctx context.Context) int {
	n, err := prd.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (prd *PlaylistRuleDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(playlistrule.Table, sqlgraph.NewFieldSpec(playlistrule.FieldID, field.TypeUUID))
	if ps := prd.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, prd.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	prd.mutation.done = true
	return affected, err
}

// PlaylistRuleDeleteOne is the builder for deleting a single PlaylistRule entity.
type PlaylistRuleDeleteOne struct {
	prd *PlaylistRuleDelete
}

// Where appends a list predicates to the PlaylistRuleDelete builder.
func (prdo *PlaylistRuleDeleteOne) Where(ps ...predicate.PlaylistRule) *PlaylistRuleDeleteOne {
	prdo.prd.mutation.Where(ps...)
	return prdo
}

// Exec executes the deletion query.
func (prdo *PlaylistRuleDeleteOne) Exec(ctx context.Context) error {
	n, err := prdo.prd.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{playlistrule.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (prdo *PlaylistRuleDeleteOne) ExecX(ctx context.Context) {
	if err := prdo.Exec(ctx); err != nil {
		panic(err)
	}
}
