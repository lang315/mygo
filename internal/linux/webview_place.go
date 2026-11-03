//go:build linux && (amd64 || arm64) && !mygo_nowebview

package linux

// The placement of web views in a window of native UI: PlaceWebViews and
// what shapes the surface and the pages by it. A build with nativeOnly
// (mygo_nowebview) has none of it (webview_none.go).

import (
	"cmp"
	"math"
	"slices"

	"github.com/egoist/mygo/internal/platform"
)

// PlaceWebViews shows the window's web views where the content shows
// them, in its paint order under the surface, and hides the others; the
// area's shapes show them and let the pointer through to them.
func (s *surface) PlaceWebViews(views []platform.WebViewPlacement) {
	host := s.w
	if host.closed || s.layers == 0 {
		return
	}
	s.placed = append(s.placed[:0], views...)
	shown := make([]*window, 0, len(views))
	for _, p := range views {
		v, ok := p.WebView.(*window)
		if !ok || v.host != host || v.closed {
			continue
		}
		f := p.Frame
		x, y := int32(math.Round(f.X)), int32(math.Round(f.Y))
		width, height := int32(math.Round(f.X+f.W))-x, int32(math.Round(f.Y+f.H))-y
		gtkLayoutMove(s.web, v.web, x, y)
		gtkWidgetSetSizeRequest(v.web, width, height)
		v.present(&p, width)
		if !gtkWidgetGetVisible(v.web) {
			gtkWidgetShow(v.web)
		}
		shown = append(shown, v)
	}
	for _, v := range host.webViews {
		if !slices.Contains(shown, v) && gtkWidgetGetVisible(v.web) {
			if gtkWidgetHasFocus(v.web) {
				// A hidden view keeps no keyboard: it goes back to the
				// content.
				gtkWidgetGrabFocus(s.area)
			}
			gtkWidgetHide(v.web)
		}
	}
	s.shape()
}

// present zooms the page of the web view w by the scale its placement p
// shows it at, width wide, and cuts its drawing to p's corners.
func (w *window) present(p *platform.WebViewPlacement, width int32) {
	scale := 1.0
	if sx, _ := p.Scale(); math.Abs(sx-1) > 1e-6 && width > 0 {
		// WebKit lays the page out at the view's width over its zoom: at
		// the page's own, whatever whole pixels the view has.
		pw, _ := p.Size()
		scale = float64(width) / pw
	}
	if old := cmp.Or(w.pageScale, 1); scale != old {
		webkitWebViewSetZoomLevel(w.web, webkitWebViewGetZoomLevel(w.web)/old*scale)
		w.pageScale = scale
	}
	var corners [4]float64
	for i, r := range p.Radii {
		corners[i] = r * scale
	}
	if corners != w.corners {
		w.corners = corners
		gtkWidgetQueueDraw(w.web)
	}
}

// shape lets the pointer through the area where the web views take it:
// in their clips but where the content painted over them. A drawing area
// (frames drawn in memory) is shaped by the same region, as only its
// shape lets the pages show through its own GdkWindow (#693). A GtkGLArea
// has no window of its own: it draws in its parent's, with an alpha
// channel whose holes show the pages, and is left unshaped. The shapes
// follow the area's size (cbSurfaceSize) and new windows
// (cbSurfaceRealize), and go once no web view is placed.
func (s *surface) shape() {
	var all gdkRectangle
	gtkWidgetGetAllocation(s.area, &all)
	region := shapeRegion(all.Width, all.Height, s.placed) // none: no shape
	// The overlay's window of the area.
	if win := gtkWidgetGetParentWindow(s.area); win != 0 && win != gtkWidgetGetWindow(s.w.win) {
		gdkWindowInputShapeCombineRegion(win, region, 0, 0)
	}
	if gtkWidgetGetHasWindow(s.area) {
		if win := gtkWidgetGetWindow(s.area); win != 0 {
			gdkWindowShapeCombineRegion(win, region, 0, 0)
		}
	}
	if region != 0 {
		cairoRegionDestroy(region)
	}
}

// shapeRegion returns the region, in an area width by height, that shows
// the content and takes the pointer with web views placed there: all of
// it but their clips, save what the content painted over them; 0 without
// web views. Clips leave out the pixels they cover only in part, covers
// put back those they cover in part.
func shapeRegion(width, height int32, placed []platform.WebViewPlacement) ptr {
	if len(placed) == 0 {
		return 0
	}
	region := cairoRegionCreateRectangle(&gdkRectangle{Width: width, Height: height})
	for _, p := range placed {
		clip := gdkRect(p.Clip)
		cairoRegionSubtractRectangle(region, clip)
		// Save what its corners cut off.
		for _, r := range cornerRows(&p) {
			if r = intersectRect(r, *clip); r.Width > 0 && r.Height > 0 {
				cairoRegionUnionRectangle(region, &r)
			}
		}
		// Unless what the content painted over it: put back by a second
		// region, as cairo has no union of a difference.
		for _, c := range p.Covers {
			cover := cairoRegionCreateRectangle(gdkRectOut(c))
			cairoRegionUnion(region, cover)
			cairoRegionDestroy(cover)
		}
	}
	return region
}

// cornerRows returns, by rows, the pixels of the surface that the corners
// of a placed page cut off at all, in DIPs.
func cornerRows(p *platform.WebViewPlacement) []gdkRectangle {
	if !p.Rounded() {
		return nil
	}
	sx, sy := p.Scale()
	var rx, ry [4]float64
	for i, r := range p.Radii {
		rx[i], ry[i] = r*sx, r*sy
	}
	return cornerPixels(p.Frame, rx, ry)
}

// intersectRect returns the pixels a and b both have.
func intersectRect(a, b gdkRectangle) gdkRectangle {
	x0, y0 := max(a.X, b.X), max(a.Y, b.Y)
	x1, y1 := min(a.X+a.Width, b.X+b.Width), min(a.Y+a.Height, b.Y+b.Height)
	return gdkRectangle{X: x0, Y: y0, Width: max(x1-x0, 0), Height: max(y1-y0, 0)}
}

// gdkRect returns the pixels of r, in DIPs, that it covers wholly.
func gdkRect(r platform.RectF) *gdkRectangle {
	x, y := int32(math.Ceil(r.X)), int32(math.Ceil(r.Y))
	return &gdkRectangle{X: x, Y: y, Width: max(int32(math.Floor(r.X+r.W))-x, 0), Height: max(int32(math.Floor(r.Y+r.H))-y, 0)}
}

// gdkRectOut returns the pixels of r, in DIPs, that it covers at all.
func gdkRectOut(r platform.RectF) *gdkRectangle {
	x, y := int32(math.Floor(r.X)), int32(math.Floor(r.Y))
	return &gdkRectangle{X: x, Y: y, Width: max(int32(math.Ceil(r.X+r.W))-x, 0), Height: max(int32(math.Ceil(r.Y+r.H))-y, 0)}
}
