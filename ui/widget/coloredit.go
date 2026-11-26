package widget

import "github.com/egoist/mygo/ui"

// ColorEditWidget is a swatch of a color that opens a color picker below
// it, inline in a form, as DevExpress's ColorPickEdit: a click, Enter or
// Space opens it, and Escape or a click outside closes it. It is
// ui.ColorWell underneath; ColorDialog is the picker in a dialog.
type ColorEditWidget struct {
	edit func() // editedOnce's
	common[ColorEditWidget]
	color    mirror[ui.Color]
	onChange func()
}

// ColorEdit makes a color edit showing the zero Color, until SetColor.
func ColorEdit() *ColorEditWidget {
	e := &ColorEditWidget{}
	e.init(e)
	return e
}

// Color returns the color chosen.
func (e *ColorEditWidget) Color() ui.Color { mu.Lock(); defer mu.Unlock(); return e.color.value }

// SetColor shows c, without running OnChange.
func (e *ColorEditWidget) SetColor(c ui.Color) *ColorEditWidget {
	e.change(func() { e.color.set(c) })
	return e
}

// OnChange runs fn, on the event goroutine, after the user picked a color.
// A setter does not run it.
func (e *ColorEditWidget) OnChange(fn func()) *ColorEditWidget {
	e.change(func() { e.onChange = fn })
	return e
}

func (e *ColorEditWidget) build(c *ui.Context) ui.Element {
	p, changed := e.color.bind()
	if changed && e.onChange != nil {
		Post(e.onChange)
	}
	return ui.ColorWell(c, p).OnChange(editedOnce(&e.edit, &e.color, &e.bound, &e.onChange))
}
