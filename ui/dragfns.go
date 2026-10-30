package ui

import (
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/transfer"
)

// The engine reaches the dragging of values and the targets of native
// transfers through the functions of dragHooks, which the API that makes a
// source or a target of a drag sets (linkDrag): an app that uses none has no
// source and no target, so nothing is dragged and nothing takes what is
// dropped, and it does not link that code. What the engine keeps for every
// app stays outside them: it follows a native transfer over its window
// (dataDragOver: incoming, the pointer, the frame) and scrolls under it
// (dragScroll), whether or not a target exists, as an app that takes files
// with DroppedFiles needs. The engine reads the hooks at every pointer event
// whether or not its app drags, so they are published as one value, loaded
// atomically, and each is called through a wrapper that does what an engine
// with no drag does when they are not set.
type dragFns struct {
	move         func(rt *engine, x, y float32)
	pressDragger func(rt *engine, chain []uint64, target *state) uint64
	end          func(rt *engine) bool
	cancel       func(rt *engine) bool
	paint        func(rt *engine, p *Painter, w, h float32)
	dataTargetAt func(rt *engine, x, y float32, d *platform.DataDragEvent) *state
	dropData     func(rt *engine, s *state, x, y float32, d *platform.DataDragEvent) bool
}

var (
	dragHooks atomic.Pointer[dragFns]
	dragOnce  sync.Once
)

// linkDrag sets dragHooks, once. It is the first call of Drag, DragData and
// their variants, of Drop, DragOver, DragHover, DropData and DataDragOver,
// and of the reordering of lists and grids, which make the sources and the
// targets.
func linkDrag() {
	dragOnce.Do(func() {
		dragHooks.Store(&dragFns{
			move:         (*engine).dragMove,
			pressDragger: pressDragger,
			end:          (*engine).dragEnd,
			cancel:       (*engine).dragCancel,
			paint:        (*engine).paintDrag,
			dataTargetAt: (*engine).dataTargetAt,
			dropData:     (*engine).dropData,
		})
	})
}

func (rt *engine) dragMoveLinked(x, y float32) {
	if f := dragHooks.Load(); f != nil {
		f.move(rt, x, y)
	}
}

func (rt *engine) pressDraggerLinked(chain []uint64, target *state) uint64 {
	if f := dragHooks.Load(); f != nil {
		return f.pressDragger(rt, chain, target)
	}
	return 0
}

func (rt *engine) dragEndLinked() bool {
	f := dragHooks.Load()
	return f != nil && f.end(rt)
}

func (rt *engine) dragCancelLinked() bool {
	f := dragHooks.Load()
	return f != nil && f.cancel(rt)
}

func (rt *engine) paintDragLinked(p *Painter, w, h float32) {
	if f := dragHooks.Load(); f != nil {
		f.paint(rt, p, w, h)
	}
}

// dataTargetAtLinked returns the target of the transfer d at (x, y), nil
// when none takes it, which is always so with no target made.
func (rt *engine) dataTargetAtLinked(x, y float32, d *platform.DataDragEvent) *state {
	if f := dragHooks.Load(); f != nil {
		return f.dataTargetAt(rt, x, y, d)
	}
	d.Operation, d.Formats = transfer.None, nil
	return nil
}

// dropDataLinked gives the target s what was dropped. A target exists only
// once a hook is set.
func (rt *engine) dropDataLinked(s *state, x, y float32, d *platform.DataDragEvent) bool {
	f := dragHooks.Load()
	return f != nil && f.dropData(rt, s, x, y, d)
}
