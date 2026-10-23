package widget

import "github.com/egoist/mygo/ui"

// CustomWidget is a control that draws itself with ui, made by Custom.
// Embed a *CustomWidget in a type of your own to make that type a Control,
// as a WinForms UserControl derives from Control: its setters change the
// state the build reads within Change.
type CustomWidget struct {
	common[CustomWidget]
	fn func(c *ui.Context) ui.Element
}

// Custom makes a control that build draws with ui every frame it is
// shown. build runs on the main thread while the package holds its lock:
// read the control's state there, but call no setter and no Change from
// it, which would wait for the lock forever. A nil build, or one that
// returns a zero Element, shows nothing.
func Custom(build func(c *ui.Context) ui.Element) *CustomWidget {
	w := &CustomWidget{fn: build}
	w.init(w)
	return w
}

func (w *CustomWidget) build(c *ui.Context) ui.Element {
	if w.fn == nil {
		return ui.Element{}
	}
	return w.fn(c)
}

// Change runs fn while no window builds, then redraws the windows on
// screen. Change the state a Custom control's build reads within it, from
// any goroutine but never from a build.
func Change(fn func()) { change(fn) }
