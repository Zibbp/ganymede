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
	"github.com/zibbp/ganymede/ent/channel"
	"github.com/zibbp/ganymede/ent/predicate"
	"github.com/zibbp/ganymede/ent/queue"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/utils"
)

// VodUpdate is the builder for updating Vod entities.
type VodUpdate struct {
	config
	hooks    []Hook
	mutation *VodMutation
}

// Where appends a list predicates to the VodUpdate builder.
func (vu *VodUpdate) Where(ps ...predicate.Vod) *VodUpdate {
	vu.mutation.Where(ps...)
	return vu
}

// SetExtID sets the "ext_id" field.
func (vu *VodUpdate) SetExtID(s string) *VodUpdate {
	vu.mutation.SetExtID(s)
	return vu
}

// SetPlatform sets the "platform" field.
func (vu *VodUpdate) SetPlatform(up utils.VodPlatform) *VodUpdate {
	vu.mutation.SetPlatform(up)
	return vu
}

// SetNillablePlatform sets the "platform" field if the given value is not nil.
func (vu *VodUpdate) SetNillablePlatform(up *utils.VodPlatform) *VodUpdate {
	if up != nil {
		vu.SetPlatform(*up)
	}
	return vu
}

// SetType sets the "type" field.
func (vu *VodUpdate) SetType(ut utils.VodType) *VodUpdate {
	vu.mutation.SetType(ut)
	return vu
}

// SetNillableType sets the "type" field if the given value is not nil.
func (vu *VodUpdate) SetNillableType(ut *utils.VodType) *VodUpdate {
	if ut != nil {
		vu.SetType(*ut)
	}
	return vu
}

// SetTitle sets the "title" field.
func (vu *VodUpdate) SetTitle(s string) *VodUpdate {
	vu.mutation.SetTitle(s)
	return vu
}

// SetDuration sets the "duration" field.
func (vu *VodUpdate) SetDuration(i int) *VodUpdate {
	vu.mutation.ResetDuration()
	vu.mutation.SetDuration(i)
	return vu
}

// SetNillableDuration sets the "duration" field if the given value is not nil.
func (vu *VodUpdate) SetNillableDuration(i *int) *VodUpdate {
	if i != nil {
		vu.SetDuration(*i)
	}
	return vu
}

// AddDuration adds i to the "duration" field.
func (vu *VodUpdate) AddDuration(i int) *VodUpdate {
	vu.mutation.AddDuration(i)
	return vu
}

// SetViews sets the "views" field.
func (vu *VodUpdate) SetViews(i int) *VodUpdate {
	vu.mutation.ResetViews()
	vu.mutation.SetViews(i)
	return vu
}

// SetNillableViews sets the "views" field if the given value is not nil.
func (vu *VodUpdate) SetNillableViews(i *int) *VodUpdate {
	if i != nil {
		vu.SetViews(*i)
	}
	return vu
}

// AddViews adds i to the "views" field.
func (vu *VodUpdate) AddViews(i int) *VodUpdate {
	vu.mutation.AddViews(i)
	return vu
}

// SetResolution sets the "resolution" field.
func (vu *VodUpdate) SetResolution(s string) *VodUpdate {
	vu.mutation.SetResolution(s)
	return vu
}

// SetNillableResolution sets the "resolution" field if the given value is not nil.
func (vu *VodUpdate) SetNillableResolution(s *string) *VodUpdate {
	if s != nil {
		vu.SetResolution(*s)
	}
	return vu
}

// ClearResolution clears the value of the "resolution" field.
func (vu *VodUpdate) ClearResolution() *VodUpdate {
	vu.mutation.ClearResolution()
	return vu
}

// SetProcessing sets the "processing" field.
func (vu *VodUpdate) SetProcessing(b bool) *VodUpdate {
	vu.mutation.SetProcessing(b)
	return vu
}

// SetNillableProcessing sets the "processing" field if the given value is not nil.
func (vu *VodUpdate) SetNillableProcessing(b *bool) *VodUpdate {
	if b != nil {
		vu.SetProcessing(*b)
	}
	return vu
}

// SetThumbnailPath sets the "thumbnail_path" field.
func (vu *VodUpdate) SetThumbnailPath(s string) *VodUpdate {
	vu.mutation.SetThumbnailPath(s)
	return vu
}

// SetNillableThumbnailPath sets the "thumbnail_path" field if the given value is not nil.
func (vu *VodUpdate) SetNillableThumbnailPath(s *string) *VodUpdate {
	if s != nil {
		vu.SetThumbnailPath(*s)
	}
	return vu
}

// ClearThumbnailPath clears the value of the "thumbnail_path" field.
func (vu *VodUpdate) ClearThumbnailPath() *VodUpdate {
	vu.mutation.ClearThumbnailPath()
	return vu
}

// SetWebThumbnailPath sets the "web_thumbnail_path" field.
func (vu *VodUpdate) SetWebThumbnailPath(s string) *VodUpdate {
	vu.mutation.SetWebThumbnailPath(s)
	return vu
}

// SetVideoPath sets the "video_path" field.
func (vu *VodUpdate) SetVideoPath(s string) *VodUpdate {
	vu.mutation.SetVideoPath(s)
	return vu
}

// SetChatPath sets the "chat_path" field.
func (vu *VodUpdate) SetChatPath(s string) *VodUpdate {
	vu.mutation.SetChatPath(s)
	return vu
}

// SetNillableChatPath sets the "chat_path" field if the given value is not nil.
func (vu *VodUpdate) SetNillableChatPath(s *string) *VodUpdate {
	if s != nil {
		vu.SetChatPath(*s)
	}
	return vu
}

// ClearChatPath clears the value of the "chat_path" field.
func (vu *VodUpdate) ClearChatPath() *VodUpdate {
	vu.mutation.ClearChatPath()
	return vu
}

// SetChatVideoPath sets the "chat_video_path" field.
func (vu *VodUpdate) SetChatVideoPath(s string) *VodUpdate {
	vu.mutation.SetChatVideoPath(s)
	return vu
}

// SetNillableChatVideoPath sets the "chat_video_path" field if the given value is not nil.
func (vu *VodUpdate) SetNillableChatVideoPath(s *string) *VodUpdate {
	if s != nil {
		vu.SetChatVideoPath(*s)
	}
	return vu
}

// ClearChatVideoPath clears the value of the "chat_video_path" field.
func (vu *VodUpdate) ClearChatVideoPath() *VodUpdate {
	vu.mutation.ClearChatVideoPath()
	return vu
}

// SetInfoPath sets the "info_path" field.
func (vu *VodUpdate) SetInfoPath(s string) *VodUpdate {
	vu.mutation.SetInfoPath(s)
	return vu
}

// SetNillableInfoPath sets the "info_path" field if the given value is not nil.
func (vu *VodUpdate) SetNillableInfoPath(s *string) *VodUpdate {
	if s != nil {
		vu.SetInfoPath(*s)
	}
	return vu
}

// ClearInfoPath clears the value of the "info_path" field.
func (vu *VodUpdate) ClearInfoPath() *VodUpdate {
	vu.mutation.ClearInfoPath()
	return vu
}

// SetStreamedAt sets the "streamed_at" field.
func (vu *VodUpdate) SetStreamedAt(t time.Time) *VodUpdate {
	vu.mutation.SetStreamedAt(t)
	return vu
}

// SetNillableStreamedAt sets the "streamed_at" field if the given value is not nil.
func (vu *VodUpdate) SetNillableStreamedAt(t *time.Time) *VodUpdate {
	if t != nil {
		vu.SetStreamedAt(*t)
	}
	return vu
}

// SetUpdatedAt sets the "updated_at" field.
func (vu *VodUpdate) SetUpdatedAt(t time.Time) *VodUpdate {
	vu.mutation.SetUpdatedAt(t)
	return vu
}

// SetChannelID sets the "channel" edge to the Channel entity by ID.
func (vu *VodUpdate) SetChannelID(id uuid.UUID) *VodUpdate {
	vu.mutation.SetChannelID(id)
	return vu
}

// SetChannel sets the "channel" edge to the Channel entity.
func (vu *VodUpdate) SetChannel(c *Channel) *VodUpdate {
	return vu.SetChannelID(c.ID)
}

// SetQueueID sets the "queue" edge to the Queue entity by ID.
func (vu *VodUpdate) SetQueueID(id uuid.UUID) *VodUpdate {
	vu.mutation.SetQueueID(id)
	return vu
}

// SetNillableQueueID sets the "queue" edge to the Queue entity by ID if the given value is not nil.
func (vu *VodUpdate) SetNillableQueueID(id *uuid.UUID) *VodUpdate {
	if id != nil {
		vu = vu.SetQueueID(*id)
	}
	return vu
}

// SetQueue sets the "queue" edge to the Queue entity.
func (vu *VodUpdate) SetQueue(q *Queue) *VodUpdate {
	return vu.SetQueueID(q.ID)
}

// Mutation returns the VodMutation object of the builder.
func (vu *VodUpdate) Mutation() *VodMutation {
	return vu.mutation
}

// ClearChannel clears the "channel" edge to the Channel entity.
func (vu *VodUpdate) ClearChannel() *VodUpdate {
	vu.mutation.ClearChannel()
	return vu
}

// ClearQueue clears the "queue" edge to the Queue entity.
func (vu *VodUpdate) ClearQueue() *VodUpdate {
	vu.mutation.ClearQueue()
	return vu
}

// Save executes the query and returns the number of nodes affected by the update operation.
func (vu *VodUpdate) Save(ctx context.Context) (int, error) {
	var (
		err      error
		affected int
	)
	vu.defaults()
	if len(vu.hooks) == 0 {
		if err = vu.check(); err != nil {
			return 0, err
		}
		affected, err = vu.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*VodMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = vu.check(); err != nil {
				return 0, err
			}
			vu.mutation = mutation
			affected, err = vu.sqlSave(ctx)
			mutation.done = true
			return affected, err
		})
		for i := len(vu.hooks) - 1; i >= 0; i-- {
			if vu.hooks[i] == nil {
				return 0, fmt.Errorf("ent: uninitialized hook (forgotten import ent/runtime?)")
			}
			mut = vu.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, vu.mutation); err != nil {
			return 0, err
		}
	}
	return affected, err
}

// SaveX is like Save, but panics if an error occurs.
func (vu *VodUpdate) SaveX(ctx context.Context) int {
	affected, err := vu.Save(ctx)
	if err != nil {
		panic(err)
	}
	return affected
}

// Exec executes the query.
func (vu *VodUpdate) Exec(ctx context.Context) error {
	_, err := vu.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (vu *VodUpdate) ExecX(ctx context.Context) {
	if err := vu.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (vu *VodUpdate) defaults() {
	if _, ok := vu.mutation.UpdatedAt(); !ok {
		v := vod.UpdateDefaultUpdatedAt()
		vu.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (vu *VodUpdate) check() error {
	if v, ok := vu.mutation.Platform(); ok {
		if err := vod.PlatformValidator(v); err != nil {
			return &ValidationError{Name: "platform", err: fmt.Errorf(`ent: validator failed for field "Vod.platform": %w`, err)}
		}
	}
	if v, ok := vu.mutation.GetType(); ok {
		if err := vod.TypeValidator(v); err != nil {
			return &ValidationError{Name: "type", err: fmt.Errorf(`ent: validator failed for field "Vod.type": %w`, err)}
		}
	}
	if _, ok := vu.mutation.ChannelID(); vu.mutation.ChannelCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "Vod.channel"`)
	}
	return nil
}

func (vu *VodUpdate) sqlSave(ctx context.Context) (n int, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   vod.Table,
			Columns: vod.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: vod.FieldID,
			},
		},
	}
	if ps := vu.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := vu.mutation.ExtID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: vod.FieldExtID,
		})
	}
	if value, ok := vu.mutation.Platform(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: vod.FieldPlatform,
		})
	}
	if value, ok := vu.mutation.GetType(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: vod.FieldType,
		})
	}
	if value, ok := vu.mutation.Title(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: vod.FieldTitle,
		})
	}
	if value, ok := vu.mutation.Duration(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: vod.FieldDuration,
		})
	}
	if value, ok := vu.mutation.AddedDuration(); ok {
		_spec.Fields.Add = append(_spec.Fields.Add, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: vod.FieldDuration,
		})
	}
	if value, ok := vu.mutation.Views(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: vod.FieldViews,
		})
	}
	if value, ok := vu.mutation.AddedViews(); ok {
		_spec.Fields.Add = append(_spec.Fields.Add, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: vod.FieldViews,
		})
	}
	if value, ok := vu.mutation.Resolution(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: vod.FieldResolution,
		})
	}
	if vu.mutation.ResolutionCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Column: vod.FieldResolution,
		})
	}
	if value, ok := vu.mutation.Processing(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeBool,
			Value:  value,
			Column: vod.FieldProcessing,
		})
	}
	if value, ok := vu.mutation.ThumbnailPath(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: vod.FieldThumbnailPath,
		})
	}
	if vu.mutation.ThumbnailPathCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Column: vod.FieldThumbnailPath,
		})
	}
	if value, ok := vu.mutation.WebThumbnailPath(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: vod.FieldWebThumbnailPath,
		})
	}
	if value, ok := vu.mutation.VideoPath(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: vod.FieldVideoPath,
		})
	}
	if value, ok := vu.mutation.ChatPath(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: vod.FieldChatPath,
		})
	}
	if vu.mutation.ChatPathCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Column: vod.FieldChatPath,
		})
	}
	if value, ok := vu.mutation.ChatVideoPath(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: vod.FieldChatVideoPath,
		})
	}
	if vu.mutation.ChatVideoPathCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Column: vod.FieldChatVideoPath,
		})
	}
	if value, ok := vu.mutation.InfoPath(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: vod.FieldInfoPath,
		})
	}
	if vu.mutation.InfoPathCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Column: vod.FieldInfoPath,
		})
	}
	if value, ok := vu.mutation.StreamedAt(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: vod.FieldStreamedAt,
		})
	}
	if value, ok := vu.mutation.UpdatedAt(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: vod.FieldUpdatedAt,
		})
	}
	if vu.mutation.ChannelCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   vod.ChannelTable,
			Columns: []string{vod.ChannelColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: channel.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vu.mutation.ChannelIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   vod.ChannelTable,
			Columns: []string{vod.ChannelColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: channel.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if vu.mutation.QueueCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   vod.QueueTable,
			Columns: []string{vod.QueueColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: queue.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vu.mutation.QueueIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   vod.QueueTable,
			Columns: []string{vod.QueueColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: queue.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if n, err = sqlgraph.UpdateNodes(ctx, vu.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{vod.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{err.Error(), err}
		}
		return 0, err
	}
	return n, nil
}

// VodUpdateOne is the builder for updating a single Vod entity.
type VodUpdateOne struct {
	config
	fields   []string
	hooks    []Hook
	mutation *VodMutation
}

// SetExtID sets the "ext_id" field.
func (vuo *VodUpdateOne) SetExtID(s string) *VodUpdateOne {
	vuo.mutation.SetExtID(s)
	return vuo
}

// SetPlatform sets the "platform" field.
func (vuo *VodUpdateOne) SetPlatform(up utils.VodPlatform) *VodUpdateOne {
	vuo.mutation.SetPlatform(up)
	return vuo
}

// SetNillablePlatform sets the "platform" field if the given value is not nil.
func (vuo *VodUpdateOne) SetNillablePlatform(up *utils.VodPlatform) *VodUpdateOne {
	if up != nil {
		vuo.SetPlatform(*up)
	}
	return vuo
}

// SetType sets the "type" field.
func (vuo *VodUpdateOne) SetType(ut utils.VodType) *VodUpdateOne {
	vuo.mutation.SetType(ut)
	return vuo
}

// SetNillableType sets the "type" field if the given value is not nil.
func (vuo *VodUpdateOne) SetNillableType(ut *utils.VodType) *VodUpdateOne {
	if ut != nil {
		vuo.SetType(*ut)
	}
	return vuo
}

// SetTitle sets the "title" field.
func (vuo *VodUpdateOne) SetTitle(s string) *VodUpdateOne {
	vuo.mutation.SetTitle(s)
	return vuo
}

// SetDuration sets the "duration" field.
func (vuo *VodUpdateOne) SetDuration(i int) *VodUpdateOne {
	vuo.mutation.ResetDuration()
	vuo.mutation.SetDuration(i)
	return vuo
}

// SetNillableDuration sets the "duration" field if the given value is not nil.
func (vuo *VodUpdateOne) SetNillableDuration(i *int) *VodUpdateOne {
	if i != nil {
		vuo.SetDuration(*i)
	}
	return vuo
}

// AddDuration adds i to the "duration" field.
func (vuo *VodUpdateOne) AddDuration(i int) *VodUpdateOne {
	vuo.mutation.AddDuration(i)
	return vuo
}

// SetViews sets the "views" field.
func (vuo *VodUpdateOne) SetViews(i int) *VodUpdateOne {
	vuo.mutation.ResetViews()
	vuo.mutation.SetViews(i)
	return vuo
}

// SetNillableViews sets the "views" field if the given value is not nil.
func (vuo *VodUpdateOne) SetNillableViews(i *int) *VodUpdateOne {
	if i != nil {
		vuo.SetViews(*i)
	}
	return vuo
}

// AddViews adds i to the "views" field.
func (vuo *VodUpdateOne) AddViews(i int) *VodUpdateOne {
	vuo.mutation.AddViews(i)
	return vuo
}

// SetResolution sets the "resolution" field.
func (vuo *VodUpdateOne) SetResolution(s string) *VodUpdateOne {
	vuo.mutation.SetResolution(s)
	return vuo
}

// SetNillableResolution sets the "resolution" field if the given value is not nil.
func (vuo *VodUpdateOne) SetNillableResolution(s *string) *VodUpdateOne {
	if s != nil {
		vuo.SetResolution(*s)
	}
	return vuo
}

// ClearResolution clears the value of the "resolution" field.
func (vuo *VodUpdateOne) ClearResolution() *VodUpdateOne {
	vuo.mutation.ClearResolution()
	return vuo
}

// SetProcessing sets the "processing" field.
func (vuo *VodUpdateOne) SetProcessing(b bool) *VodUpdateOne {
	vuo.mutation.SetProcessing(b)
	return vuo
}

// SetNillableProcessing sets the "processing" field if the given value is not nil.
func (vuo *VodUpdateOne) SetNillableProcessing(b *bool) *VodUpdateOne {
	if b != nil {
		vuo.SetProcessing(*b)
	}
	return vuo
}

// SetThumbnailPath sets the "thumbnail_path" field.
func (vuo *VodUpdateOne) SetThumbnailPath(s string) *VodUpdateOne {
	vuo.mutation.SetThumbnailPath(s)
	return vuo
}

// SetNillableThumbnailPath sets the "thumbnail_path" field if the given value is not nil.
func (vuo *VodUpdateOne) SetNillableThumbnailPath(s *string) *VodUpdateOne {
	if s != nil {
		vuo.SetThumbnailPath(*s)
	}
	return vuo
}

// ClearThumbnailPath clears the value of the "thumbnail_path" field.
func (vuo *VodUpdateOne) ClearThumbnailPath() *VodUpdateOne {
	vuo.mutation.ClearThumbnailPath()
	return vuo
}

// SetWebThumbnailPath sets the "web_thumbnail_path" field.
func (vuo *VodUpdateOne) SetWebThumbnailPath(s string) *VodUpdateOne {
	vuo.mutation.SetWebThumbnailPath(s)
	return vuo
}

// SetVideoPath sets the "video_path" field.
func (vuo *VodUpdateOne) SetVideoPath(s string) *VodUpdateOne {
	vuo.mutation.SetVideoPath(s)
	return vuo
}

// SetChatPath sets the "chat_path" field.
func (vuo *VodUpdateOne) SetChatPath(s string) *VodUpdateOne {
	vuo.mutation.SetChatPath(s)
	return vuo
}

// SetNillableChatPath sets the "chat_path" field if the given value is not nil.
func (vuo *VodUpdateOne) SetNillableChatPath(s *string) *VodUpdateOne {
	if s != nil {
		vuo.SetChatPath(*s)
	}
	return vuo
}

// ClearChatPath clears the value of the "chat_path" field.
func (vuo *VodUpdateOne) ClearChatPath() *VodUpdateOne {
	vuo.mutation.ClearChatPath()
	return vuo
}

// SetChatVideoPath sets the "chat_video_path" field.
func (vuo *VodUpdateOne) SetChatVideoPath(s string) *VodUpdateOne {
	vuo.mutation.SetChatVideoPath(s)
	return vuo
}

// SetNillableChatVideoPath sets the "chat_video_path" field if the given value is not nil.
func (vuo *VodUpdateOne) SetNillableChatVideoPath(s *string) *VodUpdateOne {
	if s != nil {
		vuo.SetChatVideoPath(*s)
	}
	return vuo
}

// ClearChatVideoPath clears the value of the "chat_video_path" field.
func (vuo *VodUpdateOne) ClearChatVideoPath() *VodUpdateOne {
	vuo.mutation.ClearChatVideoPath()
	return vuo
}

// SetInfoPath sets the "info_path" field.
func (vuo *VodUpdateOne) SetInfoPath(s string) *VodUpdateOne {
	vuo.mutation.SetInfoPath(s)
	return vuo
}

// SetNillableInfoPath sets the "info_path" field if the given value is not nil.
func (vuo *VodUpdateOne) SetNillableInfoPath(s *string) *VodUpdateOne {
	if s != nil {
		vuo.SetInfoPath(*s)
	}
	return vuo
}

// ClearInfoPath clears the value of the "info_path" field.
func (vuo *VodUpdateOne) ClearInfoPath() *VodUpdateOne {
	vuo.mutation.ClearInfoPath()
	return vuo
}

// SetStreamedAt sets the "streamed_at" field.
func (vuo *VodUpdateOne) SetStreamedAt(t time.Time) *VodUpdateOne {
	vuo.mutation.SetStreamedAt(t)
	return vuo
}

// SetNillableStreamedAt sets the "streamed_at" field if the given value is not nil.
func (vuo *VodUpdateOne) SetNillableStreamedAt(t *time.Time) *VodUpdateOne {
	if t != nil {
		vuo.SetStreamedAt(*t)
	}
	return vuo
}

// SetUpdatedAt sets the "updated_at" field.
func (vuo *VodUpdateOne) SetUpdatedAt(t time.Time) *VodUpdateOne {
	vuo.mutation.SetUpdatedAt(t)
	return vuo
}

// SetChannelID sets the "channel" edge to the Channel entity by ID.
func (vuo *VodUpdateOne) SetChannelID(id uuid.UUID) *VodUpdateOne {
	vuo.mutation.SetChannelID(id)
	return vuo
}

// SetChannel sets the "channel" edge to the Channel entity.
func (vuo *VodUpdateOne) SetChannel(c *Channel) *VodUpdateOne {
	return vuo.SetChannelID(c.ID)
}

// SetQueueID sets the "queue" edge to the Queue entity by ID.
func (vuo *VodUpdateOne) SetQueueID(id uuid.UUID) *VodUpdateOne {
	vuo.mutation.SetQueueID(id)
	return vuo
}

// SetNillableQueueID sets the "queue" edge to the Queue entity by ID if the given value is not nil.
func (vuo *VodUpdateOne) SetNillableQueueID(id *uuid.UUID) *VodUpdateOne {
	if id != nil {
		vuo = vuo.SetQueueID(*id)
	}
	return vuo
}

// SetQueue sets the "queue" edge to the Queue entity.
func (vuo *VodUpdateOne) SetQueue(q *Queue) *VodUpdateOne {
	return vuo.SetQueueID(q.ID)
}

// Mutation returns the VodMutation object of the builder.
func (vuo *VodUpdateOne) Mutation() *VodMutation {
	return vuo.mutation
}

// ClearChannel clears the "channel" edge to the Channel entity.
func (vuo *VodUpdateOne) ClearChannel() *VodUpdateOne {
	vuo.mutation.ClearChannel()
	return vuo
}

// ClearQueue clears the "queue" edge to the Queue entity.
func (vuo *VodUpdateOne) ClearQueue() *VodUpdateOne {
	vuo.mutation.ClearQueue()
	return vuo
}

// Select allows selecting one or more fields (columns) of the returned entity.
// The default is selecting all fields defined in the entity schema.
func (vuo *VodUpdateOne) Select(field string, fields ...string) *VodUpdateOne {
	vuo.fields = append([]string{field}, fields...)
	return vuo
}

// Save executes the query and returns the updated Vod entity.
func (vuo *VodUpdateOne) Save(ctx context.Context) (*Vod, error) {
	var (
		err  error
		node *Vod
	)
	vuo.defaults()
	if len(vuo.hooks) == 0 {
		if err = vuo.check(); err != nil {
			return nil, err
		}
		node, err = vuo.sqlSave(ctx)
	} else {
		var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
			mutation, ok := m.(*VodMutation)
			if !ok {
				return nil, fmt.Errorf("unexpected mutation type %T", m)
			}
			if err = vuo.check(); err != nil {
				return nil, err
			}
			vuo.mutation = mutation
			node, err = vuo.sqlSave(ctx)
			mutation.done = true
			return node, err
		})
		for i := len(vuo.hooks) - 1; i >= 0; i-- {
			if vuo.hooks[i] == nil {
				return nil, fmt.Errorf("ent: uninitialized hook (forgotten import ent/runtime?)")
			}
			mut = vuo.hooks[i](mut)
		}
		if _, err := mut.Mutate(ctx, vuo.mutation); err != nil {
			return nil, err
		}
	}
	return node, err
}

// SaveX is like Save, but panics if an error occurs.
func (vuo *VodUpdateOne) SaveX(ctx context.Context) *Vod {
	node, err := vuo.Save(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// Exec executes the query on the entity.
func (vuo *VodUpdateOne) Exec(ctx context.Context) error {
	_, err := vuo.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (vuo *VodUpdateOne) ExecX(ctx context.Context) {
	if err := vuo.Exec(ctx); err != nil {
		panic(err)
	}
}

// defaults sets the default values of the builder before save.
func (vuo *VodUpdateOne) defaults() {
	if _, ok := vuo.mutation.UpdatedAt(); !ok {
		v := vod.UpdateDefaultUpdatedAt()
		vuo.mutation.SetUpdatedAt(v)
	}
}

// check runs all checks and user-defined validators on the builder.
func (vuo *VodUpdateOne) check() error {
	if v, ok := vuo.mutation.Platform(); ok {
		if err := vod.PlatformValidator(v); err != nil {
			return &ValidationError{Name: "platform", err: fmt.Errorf(`ent: validator failed for field "Vod.platform": %w`, err)}
		}
	}
	if v, ok := vuo.mutation.GetType(); ok {
		if err := vod.TypeValidator(v); err != nil {
			return &ValidationError{Name: "type", err: fmt.Errorf(`ent: validator failed for field "Vod.type": %w`, err)}
		}
	}
	if _, ok := vuo.mutation.ChannelID(); vuo.mutation.ChannelCleared() && !ok {
		return errors.New(`ent: clearing a required unique edge "Vod.channel"`)
	}
	return nil
}

func (vuo *VodUpdateOne) sqlSave(ctx context.Context) (_node *Vod, err error) {
	_spec := &sqlgraph.UpdateSpec{
		Node: &sqlgraph.NodeSpec{
			Table:   vod.Table,
			Columns: vod.Columns,
			ID: &sqlgraph.FieldSpec{
				Type:   field.TypeUUID,
				Column: vod.FieldID,
			},
		},
	}
	id, ok := vuo.mutation.ID()
	if !ok {
		return nil, &ValidationError{Name: "id", err: errors.New(`ent: missing "Vod.id" for update`)}
	}
	_spec.Node.ID.Value = id
	if fields := vuo.fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, vod.FieldID)
		for _, f := range fields {
			if !vod.ValidColumn(f) {
				return nil, &ValidationError{Name: f, err: fmt.Errorf("ent: invalid field %q for query", f)}
			}
			if f != vod.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, f)
			}
		}
	}
	if ps := vuo.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if value, ok := vuo.mutation.ExtID(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: vod.FieldExtID,
		})
	}
	if value, ok := vuo.mutation.Platform(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: vod.FieldPlatform,
		})
	}
	if value, ok := vuo.mutation.GetType(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeEnum,
			Value:  value,
			Column: vod.FieldType,
		})
	}
	if value, ok := vuo.mutation.Title(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: vod.FieldTitle,
		})
	}
	if value, ok := vuo.mutation.Duration(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: vod.FieldDuration,
		})
	}
	if value, ok := vuo.mutation.AddedDuration(); ok {
		_spec.Fields.Add = append(_spec.Fields.Add, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: vod.FieldDuration,
		})
	}
	if value, ok := vuo.mutation.Views(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: vod.FieldViews,
		})
	}
	if value, ok := vuo.mutation.AddedViews(); ok {
		_spec.Fields.Add = append(_spec.Fields.Add, &sqlgraph.FieldSpec{
			Type:   field.TypeInt,
			Value:  value,
			Column: vod.FieldViews,
		})
	}
	if value, ok := vuo.mutation.Resolution(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: vod.FieldResolution,
		})
	}
	if vuo.mutation.ResolutionCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Column: vod.FieldResolution,
		})
	}
	if value, ok := vuo.mutation.Processing(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeBool,
			Value:  value,
			Column: vod.FieldProcessing,
		})
	}
	if value, ok := vuo.mutation.ThumbnailPath(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: vod.FieldThumbnailPath,
		})
	}
	if vuo.mutation.ThumbnailPathCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Column: vod.FieldThumbnailPath,
		})
	}
	if value, ok := vuo.mutation.WebThumbnailPath(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: vod.FieldWebThumbnailPath,
		})
	}
	if value, ok := vuo.mutation.VideoPath(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: vod.FieldVideoPath,
		})
	}
	if value, ok := vuo.mutation.ChatPath(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: vod.FieldChatPath,
		})
	}
	if vuo.mutation.ChatPathCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Column: vod.FieldChatPath,
		})
	}
	if value, ok := vuo.mutation.ChatVideoPath(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: vod.FieldChatVideoPath,
		})
	}
	if vuo.mutation.ChatVideoPathCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Column: vod.FieldChatVideoPath,
		})
	}
	if value, ok := vuo.mutation.InfoPath(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Value:  value,
			Column: vod.FieldInfoPath,
		})
	}
	if vuo.mutation.InfoPathCleared() {
		_spec.Fields.Clear = append(_spec.Fields.Clear, &sqlgraph.FieldSpec{
			Type:   field.TypeString,
			Column: vod.FieldInfoPath,
		})
	}
	if value, ok := vuo.mutation.StreamedAt(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: vod.FieldStreamedAt,
		})
	}
	if value, ok := vuo.mutation.UpdatedAt(); ok {
		_spec.Fields.Set = append(_spec.Fields.Set, &sqlgraph.FieldSpec{
			Type:   field.TypeTime,
			Value:  value,
			Column: vod.FieldUpdatedAt,
		})
	}
	if vuo.mutation.ChannelCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   vod.ChannelTable,
			Columns: []string{vod.ChannelColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: channel.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vuo.mutation.ChannelIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.M2O,
			Inverse: true,
			Table:   vod.ChannelTable,
			Columns: []string{vod.ChannelColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: channel.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	if vuo.mutation.QueueCleared() {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   vod.QueueTable,
			Columns: []string{vod.QueueColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: queue.FieldID,
				},
			},
		}
		_spec.Edges.Clear = append(_spec.Edges.Clear, edge)
	}
	if nodes := vuo.mutation.QueueIDs(); len(nodes) > 0 {
		edge := &sqlgraph.EdgeSpec{
			Rel:     sqlgraph.O2O,
			Inverse: false,
			Table:   vod.QueueTable,
			Columns: []string{vod.QueueColumn},
			Bidi:    false,
			Target: &sqlgraph.EdgeTarget{
				IDSpec: &sqlgraph.FieldSpec{
					Type:   field.TypeUUID,
					Column: queue.FieldID,
				},
			},
		}
		for _, k := range nodes {
			edge.Target.Nodes = append(edge.Target.Nodes, k)
		}
		_spec.Edges.Add = append(_spec.Edges.Add, edge)
	}
	_node = &Vod{config: vuo.config}
	_spec.Assign = _node.assignValues
	_spec.ScanValues = _node.scanValues
	if err = sqlgraph.UpdateNode(ctx, vuo.driver, _spec); err != nil {
		if _, ok := err.(*sqlgraph.NotFoundError); ok {
			err = &NotFoundError{vod.Label}
		} else if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{err.Error(), err}
		}
		return nil, err
	}
	return _node, nil
}
