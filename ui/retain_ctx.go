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
}

// readsCtx notes that the callback of the row being built read a value of the
// window.
func (c *context) readsCtx(what ctxRead) {
	if c != nil && c.row != nil {
		c.row.ctx |= what
	}
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
	return v
}

// sameCtx reports whether the values that what names are those of v.
func (c *context) sameCtx(what ctxRead, v *ctxView) bool {
	return c.viewOf(what) == *v
}
