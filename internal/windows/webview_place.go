//go:build windows && (amd64 || arm64) && !mygo_nowebview

package windows

// The placement of web views in a window of native UI: PlaceWebViews, their
// visuals and the corners they are cut to. A build with nativeOnly
// (mygo_nowebview) has none of it (webview_none.go).

import (
	"math"
	"slices"
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

// PlaceWebViews shows the window's web views where the content shows
// them, in its paint order under the surface's frames, and hides the
// others.
func (s *surface) PlaceWebViews(views []platform.WebViewPlacement) {
	if s.w.closed || s.layer == nil {
		return
	}
	s.placed = append(s.placed[:0], views...)
	var shown []*window
	for _, p := range views {
		if v, ok := p.WebView.(*window); ok && v.host == s.w && !v.closed {
			v.placement, v.shown = p, true
			v.placement.Covers = nil
			shown = append(shown, v)
		}
	}
	for _, v := range s.w.webViews {
		if !slices.Contains(shown, v) && v.shown {
			v.shown = false
			if v.focused {
				// A hidden view keeps no keyboard: it goes back to the
				// content.
				procSetFocus.Call(s.hwnd)
			}
		}
	}
	if !slices.Equal(shown, s.stacked) {
		// The visuals of those shown, in the order painted.
		comCall(s.layer.root, dcompVisualRemoveAllVisuals)
		prev := uintptr(0)
		for _, v := range shown {
			comCall(s.layer.root, dcompVisualAddVisual, v.clipVis, 1, prev)
			prev = v.clipVis
		}
		s.stacked = shown
	}
	for _, v := range s.w.webViews {
		v.placeEmbedded()
	}
	s.layer.commit()
}

// placeEmbedded shows a web view where the content last showed it, or
// hides it, once it has its controller. It stays hidden while its host
// is.
func (w *window) placeEmbedded() {
	s := w.host.surf()
	if s == nil {
		return
	}
	k := float64(s.dpi()) / 96
	px := func(v float64) float32 { return float32(math.Round(v * k)) }
	f, c := w.placement.Frame, w.placement.Clip
	cx, cy := px(c.X), px(c.Y)
	fx, fy := px(f.X), px(f.Y)
	clipVisual(w.clipVis, px(c.X+c.W)-cx, px(c.Y+c.H)-cy)
	translate(w.clipVis, cx, cy)
	// The page lays out at its size, which its visual shows scaled to its
	// frame, if a transform of the content scales it.
	width, height := px(f.X+f.W)-fx, px(f.Y+f.H)-fy
	at, scale := [2]float32{fx, fy}, [2]float32{1, 1}
	if sx, sy := w.placement.Scale(); sx != 1 || sy != 1 {
		pw, ph := w.placement.Size()
		width, height = max(px(pw), 1), max(px(ph), 1)
		at = [2]float32{float32(f.X * k), float32(f.Y * k)}
		scale = [2]float32{float32(f.W*k) / width, float32(f.H*k) / height}
	}
	w.pageAt, w.pageScale = at, scale
	// The page draws from its visual's origin.
	place(w.webVis, at[0]-cx, at[1]-cy, scale[0], scale[1])
	var radii [4]float32
	for i, r := range w.placement.Radii {
		radii[i] = float32(r * k)
	}
	w.roundCorners(s.layer, width, height, radii)
	if w.controller == 0 {
		return
	}
	if w.shown {
		// Its bounds, in the surface's pixels, place what WebView2 shows
		// in windows of its own, as a select's options.
		putBounds(w.controller, rect{int32(fx), int32(fy), int32(fx + width), int32(fy + height)})
	}
	comCall(w.controller, ctlPutIsVisible, boolArg(w.shown && !w.host.hidden))
}

// roundCorners cuts the web view w's visual to the rectangle from 0, 0 to
// width, height pixels, its corners of radii from the top-left clockwise,
// or takes the cut away without radii.
func (w *window) roundCorners(l *webLayer, width, height float32, radii [4]float32) {
	if radii == [4]float32{} {
		if w.corners != 0 {
			comCall(w.webVis, dcompVisualSetClipObject, 0)
			release(w.corners)
			w.corners = 0
		}
		return
	}
	if w.corners == 0 {
		if failed(comCall(l.dcomp, dcompDevCreateRectangleClip, uintptr(unsafe.Pointer(&w.corners)))) {
			w.corners = 0
			return
		}
		comCall(w.webVis, dcompVisualSetClipObject, w.corners)
	}
	c := w.corners
	comCallFloat32(c, rectClipSetLeft, 0)
	comCallFloat32(c, rectClipSetTop, 0)
	comCallFloat32(c, rectClipSetRight, width)
	comCallFloat32(c, rectClipSetBottom, height)
	// The setters' order goes around by the bottom-left first.
	for i, r := range [4]float32{radii[0], radii[1], radii[3], radii[2]} {
		comCallFloat32(c, rectClipSetTopLeftRadius+4*i, r)   // x
		comCallFloat32(c, rectClipSetTopLeftRadius+4*i+2, r) // y
	}
}

// translate moves a visual by x, y pixels.
func translate(visual uintptr, x, y float32) { place(visual, x, y, 1, 1) }

// place scales a visual by sx, sy, then moves it by x, y pixels.
func place(visual uintptr, x, y, sx, sy float32) {
	m := [6]float32{sx, 0, 0, sy, x, y} // D2D_MATRIX_3X2_F
	comCall(visual, dcompVisualSetTransform, uintptr(unsafe.Pointer(&m)))
}

// clipVisual clips a visual to its rectangle from 0, 0 to w, h pixels.
func clipVisual(visual uintptr, w, h float32) {
	r := [4]float32{0, 0, w, h} // D2D_RECT_F: left, top, right, bottom
	comCall(visual, dcompVisualSetClip, uintptr(unsafe.Pointer(&r)))
}
