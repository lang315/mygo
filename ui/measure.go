package ui

import "time"

// fitRows is how many rows of a List count in the height of a view measured
// to fit a window (MeasureHeight): it scrolls, and shows more when its window
// is taller.
const fitRows = 8

// MeasureHeight returns the height the content takes laid out width DIPs
// wide, which a window sized to its content has: the view is built and laid
// out once, headless, and its natural height taken. Elements that grow (Grow)
// take their content's height, and a List counts at most 8 rows. The view
// runs as it does in a window, on the calling goroutine: it must not need
// the main thread. Call it before the window shows; later changes of the
// content do not change a window's size.
func (v *Content) MeasureHeight(width float32) float32 {
	h := &headless{w: width, h: 600, scale: 1}
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
