//go:build linux && (amd64 || arm64)

package linux

import (
	"math"
	"slices"

	"github.com/ebitengine/purego"

	"github.com/egoist/mygo/internal/platform"
)

// A web view in a window of native UI (platform.Surface.NewWebView) is a
// window of this package without a GtkWindow of its own: win is its
// host's, and its WebKitWebView is a child of a GtkLayout under the
// surface's area, in a GtkOverlay that the first web view puts where the
// area was (the area is then an overlay of it, which GTK realizes again,
// with a new GL context). The area keeps an alpha channel from then on:
// its frames are transparent where the content shows a web view (holes),
// so that what the content paints after it shows over the page, and an
// input shape lets the pointer through to the page there. A drawing area
// (frames drawn in memory) has a GdkWindow of its own, which X11 shows
// over the page whatever its frames leave transparent: the same region
// shapes it, so that the page shows in the holes. Its signals find it by
// its id in Backend.webViews, apart from the windows.
//
// GTK 3 has no transforms of widgets: a page the content shows scaled has
// its view at the size it shows, and WebKit zooms the page by the scale
// (pageScale), so that it lays out in as many CSS pixels and only looks
// smaller or larger. The view draws cut to the corners of the content's
// hole, on whole pixels, as GTK draws WebKit's frames with OpenGL within
// the rectangles of a clip alone, and the region shaping the area
// follows them.

var (
	gtkLayoutNew                     func(h, v ptr) ptr
	gtkLayoutPut                     func(l, child ptr, x, y int32)
	gtkLayoutMove                    func(l, child ptr, x, y int32)
	gtkWidgetGetParent               func(w ptr) ptr
	gtkWidgetGetParentWindow         func(w ptr) ptr
	gdkWindowInputShapeCombineRegion func(w, region ptr, x, y int32)
	gdkWindowShapeCombineRegion      func(w, region ptr, x, y int32)
	cairoRegionCreateRectangle       func(r *gdkRectangle) ptr
	cairoRegionSubtractRectangle     func(region ptr, r *gdkRectangle) int32
	cairoRegionUnion                 func(region, other ptr) int32
	cairoRegionDestroy               func(region ptr)
	cairoRegionUnionRectangle        func(region ptr, r *gdkRectangle) int32
	gdkCairoRegion                   func(cr, region ptr)
	cairoClip                        func(cr ptr)
	gtkWidgetEvent                   func(w, event ptr) bool
	gtkContainerGetFocusChild        func(c ptr) ptr
	webViewsLoaded                   bool

	// cbWebViewFocus handles a web view's focus signal, cbWebViewScroll
	// the scroll-event of one that scrolls its container, and
	// cbWebViewDraw its draw signal, and cbWebViewKey the key-press-events
	// its page leaves unhandled.
	cbWebViewFocus, cbWebViewScroll, cbWebViewDraw, cbWebViewKey ptr
)

// widgetClassDraw is the offset of draw in GtkWidgetClass: a GObjectClass,
// activate_signal and 19 functions.
const widgetClassDraw = 288

func loadWebViews() {
	if webViewsLoaded {
		return
	}
	webViewsLoaded = true
	mustBind(libGTK, &gtkLayoutNew, "gtk_layout_new")
	mustBind(libGTK, &gtkLayoutPut, "gtk_layout_put")
	mustBind(libGTK, &gtkLayoutMove, "gtk_layout_move")
	mustBind(libGTK, &gtkWidgetGetParent, "gtk_widget_get_parent")
	mustBind(libGTK, &gtkWidgetGetParentWindow, "gtk_widget_get_parent_window")
	mustBind(libGDK, &gdkWindowInputShapeCombineRegion, "gdk_window_input_shape_combine_region")
	mustBind(libGDK, &gdkWindowShapeCombineRegion, "gdk_window_shape_combine_region")
	mustBind(libCairo, &cairoRegionCreateRectangle, "cairo_region_create_rectangle")
	mustBind(libCairo, &cairoRegionSubtractRectangle, "cairo_region_subtract_rectangle")
	mustBind(libCairo, &cairoRegionUnion, "cairo_region_union")
	mustBind(libCairo, &cairoRegionDestroy, "cairo_region_destroy")
	mustBind(libCairo, &cairoRegionUnionRectangle, "cairo_region_union_rectangle")
	mustBind(libGDK, &gdkCairoRegion, "gdk_cairo_region")
	mustBind(libCairo, &cairoClip, "cairo_clip")
	mustBind(libGTK, &gtkWidgetEvent, "gtk_widget_event")
	mustBind(libGTK, &gtkContainerGetFocusChild, "gtk_container_get_focus_child")
	// GTK moves the keyboard with Tab through focus signals, which
	// WebKit's Tab past the page's ends starts: the content takes the
	// keyboard back, and GTK moving it into a page by itself, as a window
	// shows, leaves it with the content.
	cbWebViewFocus = purego.NewCallback(func(widget ptr, direction int32, data ptr) bool {
		w := theBackend.window(data)
		if w == nil || w.host == nil {
			return false
		}
		const tabForward, tabBackward = 0, 1 // GtkDirectionType
		switch {
		case gtkWidgetHasFocus(w.web):
			if direction == tabForward || direction == tabBackward {
				w.tabOut(direction == tabBackward)
				return true
			}
			return false
		case gtkContainerGetFocusChild(w.web) != 0:
			return false // a dialog of WebKit's in the view has the keyboard
		}
		if s := w.host.surface; s != nil {
			gtkWidgetGrabFocus(s.area)
		}
		return true
	})
	// The wheel and the touchpad over a web view with ScrollToContainer
	// scroll the content around it, as over the content itself, and never
	// reach WebKit, whose handler of the signal runs last.
	cbWebViewScroll = purego.NewCallback(func(widget, event, data ptr) bool {
		w := theBackend.window(data)
		if w == nil || w.host == nil || w.host.closed || w.host.surface == nil {
			return false
		}
		// The event is in the web view's window, at its allocation in the
		// layout, which is where the area is.
		var a gdkRectangle
		gtkWidgetGetAllocation(w.web, &a)
		w.host.surface.send(scrollEvent(event, float64(a.X), float64(a.Y)))
		return true
	})
	// A rounded page draws inside its corners: WebKit's handler, which
	// GTK would run with the clip of none before it, draws clipped here.
	cbWebViewDraw = purego.NewCallback(func(widget, cr, data ptr) bool {
		w := theBackend.window(data)
		if w == nil || w.corners == [4]float64{} {
			return false
		}
		region := cairoRegionCreateRectangle(&gdkRectangle{Width: gtkWidgetGetAllocatedWidth(widget), Height: gtkWidgetGetAllocatedHeight(widget)})
		box := platform.RectF{W: float64(gtkWidgetGetAllocatedWidth(widget)), H: float64(gtkWidgetGetAllocatedHeight(widget))}
		for _, r := range cornerPixels(box, w.corners, w.corners) {
			cairoRegionSubtractRectangle(region, &r)
		}
		gdkCairoRegion(cr, region)
		cairoRegionDestroy(region)
		cairoClip(cr)
		// The instance's class, then its draw.
		draw := *slot(*slot(widget, 0), widgetClassDraw)
		purego.SyscallN(uintptr(draw), uintptr(widget), uintptr(cr))
		return true
	})
	// A key the page leaves unhandled, which WebKit sends again once its
	// web process said so, and which its class handler then passes on:
	// this handler runs after it, for those keys alone. The content takes
	// it as if the web view's element had the focus. Tab moves the
	// keyboard (cbWebViewFocus), a modifier key on its own is no key of
	// the content's, and the shortcut of a menu item with a role, as Edit
	// › Copy's, goes on to the menu (cbMenuKey). GdkEventKey: state 24,
	// keyval 28.
	cbWebViewKey = purego.NewCallback(func(widget, event, data ptr) bool {
		w := theBackend.window(data)
		if w == nil || w.host == nil || w.host.closed || w.host.surface == nil {
			return false
		}
		keyval := field[uint32](event, 28)
		if roleKeys[w.host.accel][[2]uint32{gdkKeyvalToLower(keyval), field[uint32](event, 24) & modifierMask}] {
			return false
		}
		k, physical, mods := keyvalKey(keyval), eventPhysicalKey(event), gdkMods(field[uint32](event, 24))
		if k == platform.KeyTab || modifierKeyval(keyval) != 0 || k == platform.KeyUnknown && !platform.KeyByPosition(k, physical, mods) {
			return false
		}
		w.host.surface.send(platform.SurfaceEvent{Kind: platform.WebViewKey, Key: k, PhysicalKey: physical, Mods: mods, WebView: w})
		return true
	})
}

func (s *surface) NewWebView(o *platform.WindowOptions, h platform.WindowHandler) (platform.WebView, error) {
	if err := webKit(); err != nil {
		return nil, err
	}
	loadWebViews()
	host := s.w
	b := host.b
	if s.layers == 0 {
		s.makeLayers()
	}
	b.nextID++
	v := &window{b: b, id: b.nextID, h: h, opts: o, win: host.win, host: host}
	b.webViews[v.id] = v
	v.createWebView()
	connect(v.web, "focus", cbWebViewFocus, ptr(v.id))
	connect(v.web, "draw", cbWebViewDraw, ptr(v.id))
	connectAfter(v.web, "key-press-event", cbWebViewKey, ptr(v.id))
	if o.ScrollToContainer {
		connect(v.web, "scroll-event", cbWebViewScroll, ptr(v.id))
	}
	gtkWidgetSetNoShowAll(v.web, true) // hidden until placed
	gtkLayoutPut(s.web, v.web, 0, 0)
	b.byWebView[v.web] = v
	host.webViews = append(host.webViews, v)
	return v, nil
}

// makeLayers puts the surface's area over a layout holding the web views,
// in an overlay where the area was.
func (s *surface) makeLayers() {
	area := s.area
	parent := gtkWidgetGetParent(area)
	focused := gtkWidgetHasFocus(area)
	s.layers = newOverlay()
	s.web = gtkLayoutNew(0, 0)
	gtkContainerAdd(s.layers, s.web)
	gObjectRef(area)
	if parent != 0 {
		gtkContainerRemove(parent, area)
	}
	if s.gl {
		// Holes show the web views through the frames.
		gtkGLAreaSetHasAlpha(area, true)
	}
	gtkOverlayAddOverlay(s.layers, area)
	gObjectUnref(area)
	switch {
	case parent == 0:
	case parent == s.w.box:
		gtkBoxPackStart(parent, s.layers, true, true, 0)
		if s.w.menubar != 0 {
			gtkBoxReorderChild(parent, s.layers, 1) // under the menu bar
		}
	default: // the overlay of a hidden title bar's controls
		gtkContainerAdd(parent, s.layers)
	}
	gtkWidgetShowAll(s.layers)
	if focused {
		gtkWidgetGrabFocus(area)
	}
}

// cornerPixels returns, by rows, the pixels that corners of radii rx, ry
// (from the top-left clockwise) cut off at all from f.
func cornerPixels(f platform.RectF, rx, ry [4]float64) []gdkRectangle {
	var out []gdkRectangle
	for i := range 4 {
		rx, ry := rx[i], ry[i]
		if rx <= 0 || ry <= 0 {
			continue
		}
		right, bottom := i == 1 || i == 2, i >= 2
		cx, cy := f.X+rx, f.Y+ry // the center of the corner's curve
		if right {
			cx = f.X + f.W - rx
		}
		if bottom {
			cy = f.Y + f.H - ry
		}
		top := f.Y
		if bottom {
			top = cy
		}
		for y := math.Floor(top); y < top+ry; y++ {
			// The curve is nearest the side at the row's edge furthest
			// from its center.
			dy := cy - max(y, top)
			if bottom {
				dy = min(y+1, top+ry) - cy
			}
			half := rx * math.Sqrt(max(1-dy*dy/(ry*ry), 0))
			x0, x1 := math.Floor(f.X), math.Ceil(cx-half)
			if right {
				x0, x1 = math.Floor(cx+half), math.Ceil(f.X+f.W)
			}
			if x1 > x0 {
				out = append(out, gdkRectangle{X: int32(x0), Y: int32(y), Width: int32(x1 - x0), Height: 1})
			}
		}
	}
	return out
}

// webViewAt returns the web view that takes the pointer at x, y in the
// surface.
func (s *surface) webViewAt(x, y float64) *window {
	for i := len(s.placed) - 1; i >= 0; i-- {
		if p := &s.placed[i]; p.At(x, y) {
			if v, ok := p.WebView.(*window); ok && !v.closed {
				return v
			}
		}
	}
	return nil
}

// top returns the window that shows w: its host for a web view.
func (w *window) top() *window {
	if w.host != nil {
		return w.host
	}
	return w
}

// webViewPressed tells the content of the window that a button went down
// on the web view w, at x, y in its window, which takes the press.
func (w *window) webViewPressed(x, y float64, button uint32, mods platform.Modifiers) {
	s := w.host.surface
	if s == nil || w.host.closed {
		return
	}
	var a gdkRectangle
	gtkWidgetGetAllocation(w.web, &a)
	b, ok := map[uint32]int{1: 0, 2: 2, 3: 1}[button]
	if !ok {
		return
	}
	s.send(platform.SurfaceEvent{Kind: platform.WebViewPress, X: float64(a.X) + x, Y: float64(a.Y) + y, Button: b, Mods: mods})
}

// focusedWebView returns the web view of w that has the keyboard, if one
// does.
func (w *window) focusedWebView() *window {
	for _, v := range w.webViews {
		if !v.closed && gtkWidgetHasFocus(v.web) {
			return v
		}
	}
	return nil
}

// TabInto gives the web view the keyboard. WebKitGTK focuses no element
// of the page as the view takes it: once the page let go of the one it
// had, the Tab moving the focus here, passed on, focuses its first, or
// its last going back.
func (w *window) TabInto(back bool) {
	s := w.host.surface
	if w.closed || s == nil {
		return
	}
	w.Eval("document.activeElement && document.activeElement.blur()")
	gtkWidgetGrabFocus(w.web)
	if ev := s.pressing; ev != 0 {
		switch field[uint32](ev, 28) {
		case 0xff09, 0xfe20, 0xff89: // GDK_KEY_Tab, ISO_Left_Tab, KP_Tab
			gtkWidgetEvent(w.web, ev)
		}
	}
}

// tabOut gives the content the keyboard back as Tab leaves the page of the
// web view w past its last element, or Shift+Tab past its first.
func (w *window) tabOut(back bool) {
	s := w.host.surface
	if s == nil || w.host.closed {
		return
	}
	gtkWidgetGrabFocus(s.area)
	ev := platform.SurfaceEvent{Kind: platform.WebViewTabOut, Key: platform.KeyTab, WebView: w}
	if back {
		ev.Mods = platform.ModShift
	}
	s.send(ev)
}

// closeWebView closes a web view: by Close, or as its window closes.
func (w *window) closeWebView() {
	if w.closed {
		return
	}
	w.closed = true
	b := w.b
	delete(b.webViews, w.id)
	delete(b.byWebView, w.web)
	if gtkWidgetHasFocus(w.web) && w.host.surface != nil {
		gtkWidgetGrabFocus(w.host.surface.area)
	}
	webkitUserContentManagerUnregisterHandler(w.ucm, cs("mygo"))
	webkitUserContentManagerRemoveAllScripts(w.ucm)
	gtkWidgetDestroy(w.web)
	gObjectUnref(w.ucm)
	if w.press.event != 0 {
		gdkEventFree(w.press.event)
		w.press.event = 0
	}
	h := w.host
	h.webViews = slices.DeleteFunc(h.webViews, func(v *window) bool { return v == w })
	if s := h.surface; s != nil {
		s.placed = slices.DeleteFunc(s.placed, func(p platform.WebViewPlacement) bool { return p.WebView == platform.WebView(w) })
		if !h.closed {
			s.shape()
		}
	}
}

// closeWebViews closes the web views of a window that closes.
func (w *window) closeWebViews() {
	for len(w.webViews) > 0 {
		w.webViews[len(w.webViews)-1].closeWebView()
	}
}
