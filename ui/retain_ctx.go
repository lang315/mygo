package ui

import "github.com/egoist/mygo/i18n"

// ctxRead is the set of values of the window that the callback of a row read
// through the Context, which the row's own states do not say: a row built
// with them is linked only while they are the same (like Modifiers).
type ctxRead uint8

const (
	ctxSize ctxRead = 1 << iota
	ctxZoom
	ctxPrefs
	ctxLocale
	ctxRegion
	ctxVibrancy
	ctxTitleBar
	ctxRouter
)

// ctxView is the values read, as they were when the row was built.
type ctxView struct {
	w, h     float32
	zoom     float32
	prefs    Preferences
	lang     string
	region   i18n.Region
	vibrancy bool
	titleBar TitleBar
	// the router and what a read of it can tell: the entry showing, its place
	// in the history and the length of it.
	router *Router
	entry  *routeEntry
	loc    string
	title  string
	at, n  int
}

// readsCtx notes that the callback of the row being built read a value of the
// window, and the value it read: the row is linked later while the window still
// has it (not the value of the end of the frame, which a pass that changes it,
// as SetZoom does, would have moved). A row that reads a value twice and gets
// two answers is built in every frame.
func (c *context) readsCtx(what ctxRead) {
	if c == nil || c.row == nil {
		return
	}
	c.row.mergeCtx(what, c.viewOf(what))
}

// mergeCtx adds the values read for what to the row's, once per value.
func (rb *rowBuild) mergeCtx(what ctxRead, v ctxView) {
	if old := rb.ctx & what; old != 0 && rb.view.pick(old) != v.pick(old) {
		rb.outside = true
	}
	rb.view.take(what&^rb.ctx, v)
	rb.ctx |= what
}

// neverKept makes the rows being built rows that are built in every frame,
// for a read that nothing in the row tells has changed (the time, the
// clipboard) or a call that must run in every build (Shortcut).
func (c *context) neverKept() {
	if c == nil {
		return
	}
	for rb := c.row; rb != nil; rb = rb.outer {
		rb.outside = true
	}
}

// viewOf returns the values of the window that what names.
func (c *context) viewOf(what ctxRead) (v ctxView) {
	if what&ctxSize != 0 {
		v.w, v.h = c.w, c.h
	}
	if what&ctxZoom != 0 {
		v.zoom = c.rt.zoomFactor()
	}
	if what&ctxPrefs != 0 {
		v.prefs = c.rt.preferences()
	}
	if what&ctxLocale != 0 {
		v.lang = c.rt.lang
	}
	if what&ctxRegion != 0 {
		v.region = c.rt.host.region()
	}
	if what&ctxVibrancy != 0 {
		v.vibrancy = c.vibrancy
	}
	if what&ctxTitleBar != 0 {
		v.titleBar = c.titleBar
	}
	if what&ctxRouter != 0 && c.router != nil {
		r := c.router
		e := r.current()
		v.router, v.entry, v.loc, v.title, v.at, v.n = r, e, e.loc, e.title, r.at, len(r.entries)
	}
	return v
}

// pick returns the values that what names and none of the others.
func (v ctxView) pick(what ctxRead) (p ctxView) {
	p.take(what, v)
	return p
}

// take copies the values that what names from o.
func (v *ctxView) take(what ctxRead, o ctxView) {
	if what&ctxSize != 0 {
		v.w, v.h = o.w, o.h
	}
	if what&ctxZoom != 0 {
		v.zoom = o.zoom
	}
	if what&ctxPrefs != 0 {
		v.prefs = o.prefs
	}
	if what&ctxLocale != 0 {
		v.lang = o.lang
	}
	if what&ctxRegion != 0 {
		v.region = o.region
	}
	if what&ctxVibrancy != 0 {
		v.vibrancy = o.vibrancy
	}
	if what&ctxTitleBar != 0 {
		v.titleBar = o.titleBar
	}
	if what&ctxRouter != 0 {
		v.router, v.entry, v.loc, v.title, v.at, v.n = o.router, o.entry, o.loc, o.title, o.at, o.n
	}
}

// sameCtx reports whether the values that what names are those of v.
func (c *context) sameCtx(what ctxRead, v *ctxView) bool {
	return c.viewOf(what) == v.pick(what)
}

// addReads records what a row read through the Context for listRow.reads and
// returns its place plus one, 0 when there are too many to number.
func (f *listFrame) addReads(r rowReads) uint16 {
	if len(f.reads) >= 1<<16-1 {
		return 0
	}
	f.reads = append(f.reads, r)
	return uint16(len(f.reads))
}
