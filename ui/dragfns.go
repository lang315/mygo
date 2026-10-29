package ui

import (
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo/internal/platform"
)

// The engine reaches the dragging of values and of native transfers through
// the functions of dragHooks, which the API that makes a source or a target
// of a drag sets (linkDrag): an app that uses none has no source and no
// target, so nothing is dragged, and does not link the code that follows a
// drag. The engine runs for every app, and reads them at every pointer event
// whether or not the app drags, so they are published as one value, loaded
// atomically, and each is called through a wrapper that does what an engine
// with no drag does when they are not set.
type dragFns struct {
	move         func(rt *engine, x, y float32)
	pressDragger func(rt *engine, chain []uint64, target *state) uint64
	end          func(rt *engine) bool
	cancel       func(rt *engine) bool
	scroll       func(rt *engine)
	paint        func(rt *engine, p *Painter, w, h float32)
	dataDragOver func(rt *engine, x, y float32, d *platform.DataDragEvent) bool
	dataDrop     func(rt *engine, x, y float32, d *platform.DataDragEvent) bool
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
			scroll:       (*engine).dragScroll,
			paint:        (*engine).paintDrag,
			dataDragOver: (*engine).dataDragOver,
			dataDrop:     (*engine).dataDrop,
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

func (rt *engine) dragScrollLinked() {
	if f := dragHooks.Load(); f != nil {
		f.scroll(rt)
	}
}

func (rt *engine) paintDragLinked(p *Painter, w, h float32) {
	if f := dragHooks.Load(); f != nil {
		f.paint(rt, p, w, h)
	}
}

func (rt *engine) dataDragOverLinked(x, y float32, d *platform.DataDragEvent) bool {
	f := dragHooks.Load()
	return f != nil && f.dataDragOver(rt, x, y, d)
}

func (rt *engine) dataDropLinked(x, y float32, d *platform.DataDragEvent) bool {
	f := dragHooks.Load()
	return f != nil && f.dataDrop(rt, x, y, d)
}
