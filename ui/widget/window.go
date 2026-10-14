package widget

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Window is a top-level window showing a widget. Make it with NewWindow,
// set it up, then Show it.
type Window struct {
	title         string
	width, height int
	resizable     bool
	icon          []byte
	root          Widget
	header        *HeaderBarWidget
	onClose       func() bool
	win           *mygo.Window // nil until Show
}

// NewWindow makes a window, not shown yet.
func NewWindow(title string, width, height int) *Window {
	return &Window{title: title, width: width, height: height, resizable: true}
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

func (w *Window) Maximize() *Window   { w.do((*mygo.Window).Maximize); return w }
func (w *Window) Unmaximize() *Window { w.do((*mygo.Window).Unmaximize); return w }

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

// Native returns the mygo window, nil before Show.
func (w *Window) Native() *mygo.Window { mu.Lock(); defer mu.Unlock(); return w.win }

// Show makes the window and shows it. Once shown, Show does nothing.
func (w *Window) Show() {
	mu.Lock()
	if w.win != nil {
		mu.Unlock()
		return
	}
	opts := mygo.WindowOptions{
		Title: w.title, Width: w.width, Height: w.height,
		Content: ui.View(w.view),
	}
	if w.header != nil {
		opts.TitleBarStyle = mygo.TitleBarHidden
	}
	icon, resizable := w.icon, w.resizable
	mu.Unlock()
	n := mygo.NewWindow(opts)
	if icon != nil {
		n.SetIcon(icon)
	}
	if !resizable {
		n.SetResizable(false)
	}
	n.OnClose(func(e *mygo.CloseEvent) {
		mu.Lock()
		h := w.onClose
		mu.Unlock()
		if h == nil {
			return
		}
		// The user's answer comes from the event goroutine: keep the
		// window, ask, and close it if the answer is yes.
		e.PreventDefault()
		Post(func() {
			if h() {
				w.closeAnyway()
			}
		})
	})
	n.OnClosed(func() {
		mu.Lock()
		delete(shown, n)
		mu.Unlock()
	})
	change(func() { w.win = n; shown[n] = struct{}{} })
}

// Close closes the window without asking OnClose.
func (w *Window) Close() { w.closeAnyway() }

func (w *Window) closeAnyway() {
	mu.Lock()
	n := w.win
	w.onClose = nil
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
	axis = vertical
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
	change(func() { h.start = append(h.start, ws...) })
	return h
}
func (h *HeaderBarWidget) PackEnd(ws ...Widget) *HeaderBarWidget {
	change(func() { h.end = append(h.end, ws...) })
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
