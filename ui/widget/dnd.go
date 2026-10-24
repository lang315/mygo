package widget

import (
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// dragDrop is what a control keeps of drag and drop, under mu.
type dragDrop struct {
	value    any // DragSource's
	hasValue bool
	data     func() transfer.Data // DragData's
	dataOpts []transfer.DragOptions
	targets  []dropTarget
}

// dropTarget is a drop handler: key says what it replaces, apply asks ui for
// the drops and hovers on e (with the state the target keeps) and posts the
// handler.
type dropTarget struct {
	key   any
	apply func(e ui.Element) (fire func())
	end   func() // ends a hover the target shows, nil for the other kinds
}

// apply declares the control's sources and targets on e, as it builds.
func (d *dragDrop) apply(c *ui.Context, e ui.Element) {
	if d.hasValue {
		e.Drag(d.value)
	}
	if d.data != nil {
		fn := d.data
		e.DragDataFrom(func() transfer.Data {
			// ui asks on the main thread with no lock of ours held, so fn
			// may read widgets.
			return fn()
		}, d.dataOpts...)
	}
	// Hovers first, so that the end of a hover is posted before the drop.
	// A drop goes to the first registered handler whose type matches, only:
	// every target is asked (ui needs each type), the first to fire runs.
	fired := false
	for _, over := range []bool{true, false} {
		for _, t := range d.targets {
			if _, isOver := t.key.(interface{ hover() }); isOver != over {
				continue
			}
			if fire := t.apply(e); fire != nil && !fired {
				fired = true
				fire()
			}
		}
	}
}

func (d *dragDrop) setTarget(key any, apply func(e ui.Element) func(), end func()) {
	for i, t := range d.targets {
		if t.key == key {
			if t.end != nil {
				t.end() // the old handler sees the hover end
			}
			if apply == nil {
				d.targets = append(d.targets[:i:i], d.targets[i+1:]...)
			} else {
				d.targets[i].apply, d.targets[i].end = apply, end
			}
			return
		}
	}
	if apply != nil {
		d.targets = append(d.targets, dropTarget{key, apply, end})
	}
}

// endHovers ends the hovers the control shows, as when it is hidden.
func (d *dragDrop) endHovers() {
	for _, t := range d.targets {
		if t.end != nil {
			t.end()
		}
	}
}

// DragSource makes the control the source of value, dragged within the
// window (a source hidden mid-drag cancels the drag silently): once the pointer pressing it moves a few DIPs, a copy of the
// control follows the pointer, and the OnDrop[T] target under it that takes
// value's type gets it when the pointer is released. A press that does not
// move still clicks, and a drag does not. nil stops it.
func (c *common[W]) DragSource(value any) *W {
	c.change(func() { c.dnd.value, c.dnd.hasValue = value, value != nil })
	return c.self
}

// DragData makes the control the source of a system drag: fn returns the
// data (text, files, URLs: package transfer) when the gesture starts, and
// other apps, or this window's OnDropData targets, can take it. fn runs on
// the main thread while the user's pointer waits, with no lock of the
// package held, so it may read widgets; it must be quick and must not wait
// for the event goroutine. nil stops it.
func (c *common[W]) DragData(fn func() transfer.Data, opts ...transfer.DragOptions) *W {
	if len(opts) > 1 {
		panic("widget: DragData takes at most one transfer.DragOptions")
	}
	c.change(func() { c.dnd.data, c.dnd.dataOpts = fn, opts })
	return c.self
}

type dropKey[T any] struct{}
type overKey[T any] struct{}

func (overKey[T]) hover() {}

type dataKey struct{}
type filesKey struct{}

// OnDrop makes the control take values of type T dragged with DragSource
// within the window, and runs fn with the one dropped, on the event
// goroutine. A control takes several types with several OnDrop calls; a drop goes to the first registered one whose type matches, only. The
// innermost control under the pointer that takes the type gets it; values of
// other types do not drop. T is matched as a Go type assertion: for an
// interface T, a value whose pointer type has the methods does not match a
// value that is not a pointer, and OnDrop[any] takes everything. A nil fn removes the target.
// There is no WinForms AllowDrop: a control takes drops while it has a
// handler.
func OnDrop[T any](c Control, fn func(T)) {
	c.base().change(func() {
		if fn == nil {
			c.base().dnd.setTarget(dropKey[T]{}, nil, nil)
			return
		}
		c.base().dnd.setTarget(dropKey[T]{}, func(e ui.Element) func() {
			if v, ok := ui.Drop[T](e); ok {
				return func() { Post(func() { fn(v) }) }
			}
			return nil
		}, nil)
	})
}

// OnDragOver runs fn, on the event goroutine, when a value of type T
// dragged with DragSource comes over the control or something inside it
// (over true, with the value) and when it leaves, is dropped, the drag is
// cancelled, or the control is hidden or its handler replaced (over false,
// with the zero value): for highlighting while a value is over. It does not
// make the control take the value, so a drop on it falls to the innermost
// control around with an OnDrop of the type, as under a drop zone with a
// highlighted child. The end of the hover always comes before OnDrop's
// handler. A nil fn removes it.
func OnDragOver[T any](c Control, fn func(over bool, v T)) {
	c.base().change(func() {
		if fn == nil {
			c.base().dnd.setTarget(overKey[T]{}, nil, nil)
			return
		}
		was := false
		c.base().dnd.setTarget(overKey[T]{}, func(e ui.Element) func() {
			v, over := ui.DragHover[T](e)
			if over != was {
				was = over
				Post(func() { fn(over, v) })
			}
			return nil
		}, func() {
			if was {
				was = false
				var zero T
				Post(func() { fn(false, zero) })
			}
		})
	})
}

// OnDropData makes the control take drops of data in one of formats (text,
// files, ...: package transfer) from other apps or from DragData, and runs
// fn with the data, which can be read later, on the event goroutine. A nil
// fn removes it.
func (c *common[W]) OnDropData(formats []transfer.Format, fn func(transfer.Data)) *W {
	c.change(func() {
		if fn == nil {
			c.dnd.setTarget(dataKey{}, nil, nil)
			return
		}
		opts := transfer.DropOptions{Formats: formats}
		c.dnd.setTarget(dataKey{}, func(e ui.Element) func() {
			if d, ok := ui.DropData(e, opts); ok {
				Post(func() { fn(d.Data) })
			}
			return nil
		}, nil)
	})
	return c.self
}

// OnDropFiles runs fn with the paths of the files dropped on the control from
// the system's file manager, on the event goroutine. A nil fn removes it.
func (c *common[W]) OnDropFiles(fn func(paths []string)) *W {
	c.change(func() {
		if fn == nil {
			c.dnd.setTarget(filesKey{}, nil, nil)
			return
		}
		c.dnd.setTarget(filesKey{}, func(e ui.Element) func() {
			if p := e.DroppedFiles(); len(p) > 0 {
				Post(func() { fn(p) })
			}
			return nil
		}, nil)
	})
	return c.self
}
