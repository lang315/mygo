//go:build darwin && !mygo_nowebview

package darwin

// The placement of web views in a window of native UI: PlaceWebViews, the
// corners of their panes and their stacking. A build with nativeOnly
// (mygo_nowebview) has none of it (webview_none.go).

import (
	"math"
	"slices"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/egoist/mygo/internal/platform"
)

var (
	paneOnce        sync.Once
	msgPointToLayer func(obj id, sel objc.SEL, p NSPoint) NSPoint
	cgPathCreate    func() uintptr
	cgPathMoveTo    func(path, transform uintptr, x, y float64)
	cgPathArcTo     func(path, transform uintptr, x1, y1, x2, y2, radius float64)
	cgPathClose     func(path uintptr)
	cgPathRelease   func(path uintptr)
)

// PlaceWebViews shows the window's web views where the content shows
// them, in its paint order under the surface, and hides the others.
func (s *surface) PlaceWebViews(views []platform.WebViewPlacement) {
	host := s.w
	if host.closed {
		return
	}
	s.placed = append(s.placed[:0], views...)
	height := msgRect(host.view, sel("bounds")).Size.Height
	shown := make([]*window, 0, len(views))
	withPool(func() {
		for _, p := range views {
			v, ok := p.WebView.(*window)
			if !ok || v.host != host || v.closed {
				continue
			}
			c, f := p.Clip, p.Frame
			pw, ph := p.Size()
			msgSetRect(v.clip, sel("setFrame:"), NSRect{Origin: NSPoint{c.X, height - c.Y - c.H}, Size: NSSize{c.W, c.H}})
			// The clip view is flipped, and so is the pane, whose bounds
			// are the page's size, scaled to its frame.
			msgSetRect(v.pane, sel("setFrame:"), NSRect{Origin: NSPoint{f.X - c.X, f.Y - c.Y}, Size: NSSize{f.W, f.H}})
			msgSetSize(v.pane, sel("setBoundsSize:"), NSSize{pw, ph})
			msgSetRect(v.web, sel("setFrame:"), NSRect{Size: NSSize{pw, ph}})
			v.roundPane(p.Radii, pw, ph)
			if sendBool(v.clip, "isHidden") {
				send(v.clip, "setHidden:", 0)
			}
			shown = append(shown, v)
		}
		for _, v := range host.webViews {
			if !slices.Contains(shown, v) && !sendBool(v.clip, "isHidden") {
				if v.hasKeyboard() {
					// A hidden view keeps no keyboard: it goes back to the
					// content.
					send(host.win, "makeFirstResponder:", uintptr(s.view))
				}
				send(v.clip, "setHidden:", 1)
			}
		}
		s.stackWebViews(shown)
	})
}

// roundPane cuts the pane of the web view w, its bounds width by height,
// to corners of radii, in its bounds' units, from the top-left clockwise,
// as continuous as the content's when they are the same.
func (w *window) roundPane(radii [4]float64, width, height float64) {
	paneOnce.Do(func() {
		purego.RegisterFunc(&msgPointToLayer, msgSendAddr)
		purego.RegisterLibFunc(&cgPathCreate, libCG, "CGPathCreateMutable")
		purego.RegisterLibFunc(&cgPathMoveTo, libCG, "CGPathMoveToPoint")
		purego.RegisterLibFunc(&cgPathArcTo, libCG, "CGPathAddArcToPoint")
		purego.RegisterLibFunc(&cgPathClose, libCG, "CGPathCloseSubpath")
		purego.RegisterLibFunc(&cgPathRelease, libCG, "CGPathRelease")
	})
	layer := send(w.pane, "layer")
	if radii == [4]float64{} {
		msgSetFloat(layer, sel("setCornerRadius:"), 0)
		w.unmask(layer)
		return
	}
	// The pane's corners in its layer, which AppKit scales and flips as
	// it does: by the layer's own transform, or its sublayers'.
	toLayer := func(x, y float64) NSPoint { return msgPointToLayer(w.pane, sel("convertPointToLayer:"), NSPoint{x, y}) }
	corners := [4]NSPoint{toLayer(0, 0), toLayer(width, 0), toLayer(width, height), toLayer(0, height)}
	k := math.Abs(corners[1].X-corners[0].X) / width // the layer's units per the bounds'
	same, r := true, 0.0
	for _, v := range radii {
		if v > 0 && r > 0 && v != r {
			same = false
		}
		r = max(r, v)
	}
	if same {
		// kCALayerMinXMinYCorner and the others, for the corners whose
		// radius is not 0, by where they are in the layer.
		var mask uint
		for i, v := range radii {
			if v <= 0 {
				continue
			}
			p, o := corners[i], corners[(i+2)%4]
			bit := uint(1)
			if p.X > o.X {
				bit <<= 1
			}
			if p.Y > o.Y {
				bit <<= 2
			}
			mask |= bit
		}
		w.unmask(layer)
		msgSetFloat(layer, sel("setCornerRadius:"), r*k)
		send(layer, "setCornerCurve:", uintptr(nsString("continuous")))
		send(layer, "setMaskedCorners:", uintptr(mask))
		return
	}
	// Corners of different radii, circular: a mask of their path, at the
	// layer's bounds, which may not start at 0, 0.
	msgSetFloat(layer, sel("setCornerRadius:"), 0)
	b := msgRect(layer, sel("bounds"))
	for i := range corners {
		corners[i].X -= b.Origin.X
		corners[i].Y -= b.Origin.Y
	}
	path := cgPathCreate()
	mid := func(a, b NSPoint) NSPoint { return NSPoint{(a.X + b.X) / 2, (a.Y + b.Y) / 2} }
	start := mid(corners[3], corners[0])
	cgPathMoveTo(path, 0, start.X, start.Y)
	for i := range 4 {
		p, next := corners[i], mid(corners[i], corners[(i+1)%4])
		cgPathArcTo(path, 0, p.X, p.Y, next.X, next.Y, radii[i]*k)
	}
	cgPathClose(path)
	send(class("CATransaction"), "begin")
	send(class("CATransaction"), "setDisableActions:", 1)
	if w.mask == 0 {
		w.mask = send(send(class("CAShapeLayer"), "alloc"), "init")
		send(layer, "setMask:", uintptr(w.mask))
	}
	msgSetRect(w.mask, sel("setFrame:"), b)
	send(w.mask, "setPath:", path)
	send(class("CATransaction"), "commit")
	cgPathRelease(path)
}

// unmask takes away the mask of the pane's layer.
func (w *window) unmask(layer id) {
	if w.mask != 0 {
		send(layer, "setMask:", 0)
		release(w.mask)
		w.mask = 0
	}
}

// stackWebViews puts the clip views of the web views shown in the order
// the content painted them, under the surface, unless they are.
func (s *surface) stackWebViews(shown []*window) {
	if len(shown) < 2 {
		return
	}
	subviews := send(s.w.view, "subviews")
	n := sendInt(subviews, "count")
	at := 0
	for i := range n {
		if at < len(shown) && send(subviews, "objectAtIndex:", uintptr(i)) == shown[at].clip {
			at++
		}
	}
	if at == len(shown) {
		return
	}
	// Moving a view out of its superview takes the keyboard from it: the
	// first responder goes back to it.
	first := send(s.w.win, "firstResponder")
	for _, v := range shown {
		retain(v.clip)
		send(v.clip, "removeFromSuperview")
		send(s.w.view, "addSubview:positioned:relativeTo:", uintptr(v.clip), nsWindowBelow, uintptr(s.view))
		release(v.clip)
	}
	if first != 0 && first != send(s.w.win, "firstResponder") {
		send(s.w.win, "makeFirstResponder:", uintptr(first))
	}
}
