package ui

import "time"

// fitRows is how many rows of a List count in the height of a view measured
// to fit a window (MeasureHeight): it scrolls, and shows more when its window
// is taller.
const fitRows = 8

// MeasureHeight returns the height the content takes laid out width DIPs
// wide on a display of the scale (1 for none), which a window sized to its
// content has: the view is built and laid out once, headless, and its natural
// height taken. Elements that grow (Grow) take their content's height, and a
// List counts at most 8 rows. The view runs as it does in a window, on the
// calling goroutine: it must not need the main thread, and it should start
// nothing and post nothing, nor consume a one-shot request, while
// Context.Measuring reports true. Call it before the window shows; later
// changes of the content do not change a window's size. A width narrower
// than the content's least width gives the height at that least width.
func (v *Content) MeasureHeight(width, scale float32) float32 {
	if scale <= 0 {
		scale = 1
	}
	h := &headless{w: width, h: 600, scale: scale}
	rt := newRuntime(v.view, h)
	rt.fitting = true
	t := &Tester{rt: rt, h: h, now: time.Now()}
	rt.clock = t.Now
	h.clock = t.Now
	t.settle()
	root := rt.c.root
	height := contentHeight(root, width-root.padX()) + root.padY()
	rt.close()
	return height
}

// Measuring reports whether the view is being laid out only to measure its
// height (Content.MeasureHeight): a headless build before the window exists.
// A control whose build starts something, posts, or consumes a request that
// is meant for the real window (a focus, a scroll) must leave it alone then.
func (c *Context) Measuring() bool { return c.rt.fitting }
