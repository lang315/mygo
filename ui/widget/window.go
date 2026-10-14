package widget

import (
	"slices"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Window is a top-level window showing a widget. Make it with NewWindow,
// set it up, then Show it.
type Window struct {
	id            int // its views' id, for focus tracking (Task 3)
	title         string
	width, height int
	resizable     bool
	maximized     bool
	icon          []byte
	root          Widget
	header        *HeaderBarWidget
	onClose       func() bool
	menu          []MenuItem
	parent        *Window       // a dialog's; nil for a top-level window
	modal         bool          // blocks parent while shown
	deletable     bool          // the user may close it
	win           *mygo.Window  // nil until shown, and again once closed
	showing       bool          // shown, or being made, and not closed yet
	closing       bool          // Close is closing it: OnClose is not asked
	asking        bool          // OnClose is asking the user
	done          chan struct{} // closed when the shown window closes
}

// quit quits the app; tests replace it.
var quit = mygo.App.Quit

var (
	watchQuit sync.Once
	// quitAsked is set while the app runs its quit sequence, which asks
	// every window to close. Only the main thread touches it.
	quitAsked bool
)

func onBeforeQuit(e *mygo.QuitEvent) {
	quitAsked = true
	// The quit sequence runs on the main thread, and WhenReady queues
	// there once ready: the flag clears right after the sequence, whether
	// it quit or a window kept it from quitting.
	mygo.App.WhenReady(func() { quitAsked = false })
}

// NewWindow makes a window, not shown yet.
func NewWindow(title string, width, height int) *Window {
	return &Window{id: newID(), title: title, width: width, height: height, resizable: true, deletable: true}
}

// Run runs an app with one window titled name: fn sets the window up on
// the event goroutine, then the window shows. Run returns when the app
// quits; call it from main.
func Run(name string, fn func(w *Window)) error {
	mygo.App.SetName(name)
	mygo.App.WhenReady(func() {
		w := NewWindow(name, 1000, 700)
		Post(func() { fn(w); w.Show() })
	})
	return mygo.App.Run()
}

// do runs fn with the native window, if shown.
func (w *Window) do(fn func(n *mygo.Window)) {
	mu.Lock()
	n := w.win
	mu.Unlock()
	if n != nil {
		fn(n)
	}
}

func (w *Window) SetTitle(s string) *Window {
	change(func() { w.title = s })
	w.do(func(n *mygo.Window) { n.SetTitle(s) })
	return w
}

func (w *Window) SetSize(width, height int) *Window {
	change(func() { w.width, w.height = width, height })
	w.do(func(n *mygo.Window) { n.SetSize(width, height) })
	return w
}

// SetIcon sets the window's icon from a PNG. macOS shows no window icons:
// there it does nothing (set the Dock's with mygo.App.Dock.SetIcon).
func (w *Window) SetIcon(png []byte) *Window {
	change(func() { w.icon = png })
	w.do(func(n *mygo.Window) { n.SetIcon(png) })
	return w
}

func (w *Window) SetResizable(on bool) *Window {
	change(func() { w.resizable = on })
	w.do(func(n *mygo.Window) { n.SetResizable(on) })
	return w
}

// Maximize maximizes the window, or makes it maximized when shown.
func (w *Window) Maximize() *Window {
	change(func() { w.maximized = true })
	w.do((*mygo.Window).Maximize)
	return w
}

func (w *Window) Unmaximize() *Window {
	change(func() { w.maximized = false })
	w.do((*mygo.Window).Unmaximize)
	return w
}

// SetMenuBar sets the window's menu bar, on Linux and Windows. On macOS
// the menu bar is the app's, set with mygo.App.SetMenu; there it does
// nothing.
func (w *Window) SetMenuBar(items ...MenuItem) *Window {
	change(func() { w.menu = items })
	w.do(func(n *mygo.Window) { n.SetMenu(mygo.NewMenu(nativeMenu(items))) })
	return w
}

// SetContent shows root in the window, replacing what it showed.
func (w *Window) SetContent(root Widget) *Window { change(func() { w.root = root }); return w }

// UseHeaderBar draws the title bar with ui: the title, subtitle and
// widgets packed at its ends. Call it before Show.
func (w *Window) UseHeaderBar(subtitle string) *HeaderBarWidget {
	h := &HeaderBarWidget{subtitle: subtitle, win: w}
	h.init(h)
	change(func() { w.header = h })
	return h
}

// HeaderBar returns the header bar, nil without UseHeaderBar.
func (w *Window) HeaderBar() *HeaderBarWidget { mu.Lock(); defer mu.Unlock(); return w.header }

// OnClose runs fn when the user asks to close the window, on the event
// goroutine; fn returning false keeps the window open. nil removes it.
func (w *Window) OnClose(fn func() bool) *Window { change(func() { w.onClose = fn }); return w }

// Native returns the mygo window, nil before Show and after the window
// closed.
func (w *Window) Native() *mygo.Window { mu.Lock(); defer mu.Unlock(); return w.win }

// Show makes the window and shows it. It does nothing while the window is
// shown; a window that closed shows again.
func (w *Window) Show() {
	mu.Lock()
	if w.showing {
		mu.Unlock()
		return
	}
	w.showing, w.closing = true, false
	done := make(chan struct{})
	w.done = done
	opts := mygo.WindowOptions{
		Title: w.title, Width: w.width, Height: w.height,
		Maximized: w.maximized,
		Content:   ui.View(w.view),
	}
	if w.header != nil {
		opts.TitleBarStyle = mygo.TitleBarHidden
	}
	if p := w.parent; p != nil && p.win != nil {
		opts.Parent, opts.Modal = p.win, w.modal
	}
	icon, resizable, menu := w.icon, w.resizable, w.menu
	mu.Unlock()
	watchQuit.Do(func() { mygo.App.OnBeforeQuit(onBeforeQuit) })
	n := mygo.NewWindow(opts)
	if icon != nil {
		n.SetIcon(icon)
	}
	if !resizable {
		n.SetResizable(false)
	}
	if menu != nil {
		n.SetMenu(mygo.NewMenu(nativeMenu(menu)))
	}
	n.OnClose(w.closeAsked)
	n.OnClosed(func() {
		mu.Lock()
		delete(shown, n)
		w.win, w.showing, w.asking = nil, false, false
		mu.Unlock()
		close(done)
	})
	change(func() {
		select {
		case <-done: // it closed already
		default:
			w.win = n
			shown[n] = struct{}{}
		}
	})
}

// closeAsked answers a request to close the window, on the main thread.
func (w *Window) closeAsked(e *mygo.CloseEvent) {
	mu.Lock()
	h, closing, asking, deletable := w.onClose, w.closing, w.asking, w.deletable
	if h != nil && !closing && deletable {
		w.asking = true
	}
	mu.Unlock()
	if closing {
		return
	}
	if !deletable {
		e.PreventDefault()
		return
	}
	if h == nil {
		return
	}
	// The user's answer comes from the event goroutine: keep the window,
	// ask, and close it if the answer is yes. Keeping it stops a quit too,
	// so a yes then quits again. A request while asking is dropped.
	e.PreventDefault()
	if asking {
		return
	}
	quitting := quitAsked
	Post(func() {
		yes := h()
		mu.Lock()
		w.asking = false
		mu.Unlock()
		if yes {
			w.closeAnyway()
			if quitting {
				quit()
			}
		}
	})
}

// Close closes the window without asking OnClose.
func (w *Window) Close() { w.closeAnyway() }

func (w *Window) closeAnyway() {
	mu.Lock()
	n := w.win
	w.closing = true
	mu.Unlock()
	if n != nil {
		n.Close()
		mu.Lock()
		delete(shown, n)
		mu.Unlock()
	}
}

func (w *Window) view(c *ui.Context) {
	mu.Lock()
	defer mu.Unlock()
	axis, building = vertical, w.id
	ui.Column(c).Fill().Children(func() {
		if w.header != nil {
			render(c, w.header)
		}
		if w.root != nil {
			render(c, w.root).Grow(1)
		}
	})
}

// HeaderBarWidget is a title bar drawn with ui, holding widgets at its
// ends. The platform keeps its window buttons.
type HeaderBarWidget struct {
	common[HeaderBarWidget]
	win        *Window
	subtitle   string
	start, end []Widget
}

func (h *HeaderBarWidget) SetTitle(s string) *HeaderBarWidget { h.win.SetTitle(s); return h }
func (h *HeaderBarWidget) SetSubtitle(s string) *HeaderBarWidget {
	change(func() { h.subtitle = s })
	return h
}
func (h *HeaderBarWidget) PackStart(ws ...Widget) *HeaderBarWidget {
	change(func() { h.start = h.add(h.start, ws) })
	return h
}
func (h *HeaderBarWidget) PackEnd(ws ...Widget) *HeaderBarWidget {
	change(func() { h.end = h.add(h.end, ws) })
	return h
}

func (h *HeaderBarWidget) build(c *ui.Context) ui.Element {
	bar, t := c.TitleBar(), c.Theme()
	return ui.Row(c).Height(max(bar.Height, 32)).Padding(0, bar.Right+t.Space(2), 0, bar.Left+t.Space(2)).
		Gap(t.Space(2)).DragWindow().Children(func() {
		outer := axis
		axis = horizontal
		defer func() { axis = outer }()
		for _, w := range h.start {
			render(c, w)
		}
		ui.Spacer(c)
		ui.Column(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, h.win.title).Bold()
			if h.subtitle != "" {
				ui.Text(c, h.subtitle).TextColor(t.TextMuted)
			}
		})
		ui.Spacer(c)
		for i := len(h.end) - 1; i >= 0; i-- {
			render(c, h.end[i])
		}
	})
}

// add appends the widgets not yet in the bar to side. Under mu.
func (h *HeaderBarWidget) add(side, ws []Widget) []Widget {
	for _, w := range ws {
		if !slices.Contains(h.start, w) && !slices.Contains(h.end, w) && !slices.Contains(side, w) {
			side = append(side, w)
		}
	}
	return side
}

// native is the mygo window of w, nil for a nil or unshown w.
func (w *Window) native() *mygo.Window {
	if w == nil {
		return nil
	}
	return w.Native()
}
