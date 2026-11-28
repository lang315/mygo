package widget

import "github.com/egoist/mygo/ui"

// FlyoutEdge is the edge of its owner a FlyoutPanel slides in from.
type FlyoutEdge uint8

// The edges: FlyoutRight, the default, is the end, the left right to left;
// FlyoutLeft is the start, the right right to left.
const (
	FlyoutRight FlyoutEdge = iota
	FlyoutLeft
	FlyoutTop
	FlyoutBottom
)

// FlyoutPanelWidget draws a control, its owner, and slides a panel of
// other controls in over it from an edge, as DevExpress's FlyoutPanel. While
// the panel shows, the owner still draws but takes no pointer; Escape, and a
// click outside the panel unless SetCloseOnOutsideClick(false), close it.
// The focus goes into the panel as it shows, and back where it was as it
// goes.
type FlyoutPanelWidget struct {
	common[FlyoutPanelWidget]
	owner, content Control
	edge           FlyoutEdge
	keepOpen       bool // SetCloseOnOutsideClick(false)
	open           bool
	focus          bool // ShowPopup asked: the panel takes the focus as it builds
	onClosed       func()
}

// FlyoutPanel makes a closed flyout panel placed where owner would be,
// drawing owner, whose panel holds content.
func FlyoutPanel(owner, content Control) *FlyoutPanelWidget {
	f := &FlyoutPanelWidget{owner: owner, content: content}
	f.init(f)
	adopt(&f.core, owner)
	adopt(&f.core, content)
	return f
}

// ShowPopup slides the panel in, in ui's motion, at once while the desktop
// asks for less motion. HidePopup slides it out. Neither runs OnClosed, as a
// setter runs no handler.
func (f *FlyoutPanelWidget) ShowPopup() *FlyoutPanelWidget {
	f.change(func() { f.open, f.focus = true, true })
	return f
}

func (f *FlyoutPanelWidget) HidePopup() *FlyoutPanelWidget {
	f.change(func() { f.open, f.focus = false, false })
	return f
}

// SetEdge sets the edge the panel comes from, FlyoutRight unless set.
func (f *FlyoutPanelWidget) SetEdge(e FlyoutEdge) *FlyoutPanelWidget {
	f.change(func() { f.edge = e })
	return f
}

// SetCloseOnOutsideClick sets whether a click outside the panel closes it,
// as it does unless set. The owner takes no click while the panel shows
// either way.
func (f *FlyoutPanelWidget) SetCloseOnOutsideClick(on bool) *FlyoutPanelWidget {
	f.change(func() { f.keepOpen = !on })
	return f
}

// IsOpen reports whether the panel shows, or slides in.
func (f *FlyoutPanelWidget) IsOpen() bool { mu.Lock(); defer mu.Unlock(); return f.open }

// OnClosed runs fn, on the event goroutine, after the user closed the
// panel: with Escape or a click outside it.
func (f *FlyoutPanelWidget) OnClosed(fn func()) *FlyoutPanelWidget {
	f.change(func() { f.onClosed = fn })
	return f
}

// flyoutKey is the key of the slide's value among the owner's animations.
type flyoutKey struct{}

func (f *FlyoutPanelWidget) build(c *ui.Context) ui.Element {
	e := render(c, f.owner)
	var to float32
	if f.open {
		to = 1
	}
	// Built every frame, closed too, so that the value starts at 0: 1 in,
	// 0 out of the owner's box. Reduced motion jumps.
	p := e.AnimateTo(flyoutKey{}, to, ui.Smooth)
	if p == 0 && !f.open || c.Measuring() || !e.Valid() {
		return e
	}
	r := e.Bounds() // the last frame's: where the owner is in the window
	rtl := c.Direction() == ui.RightToLeft
	ui.Overlay(c, func() {
		// Over the owner's box, clipping the panel on its way; as an
		// overlay's box, it takes the pointer from the owner.
		frame := ui.Box(c).Absolute().Left(r.X).Top(r.Y).Width(r.W).Height(r.H).Clip()
		frame.Children(func() {
			panel := ui.Column(c).Absolute().Role(ui.RoleGroup)
			comboPanel(c, panel)
			t := c.Theme()
			panel.Margin(0, 0, 0, 0).Radius(0).Padding(t.Space(3)).Gap(t.Space(2))
			out, x, y := (1-p)*100, float32(0), float32(0)
			switch f.edge {
			case FlyoutLeft:
				panel.Top(0).Bottom(0).Start(0)
				x = -out
			case FlyoutTop:
				panel.Top(0).Start(0).End(0)
				y = -out
			case FlyoutBottom:
				panel.Bottom(0).Start(0).End(0)
				y = out
			default:
				panel.Top(0).Bottom(0).End(0)
				x = out
			}
			if rtl {
				x = -x
			}
			panel.TranslatePercent(x, y)
			// The panel holds the focus it is given, Tab going on into the content.
			panel.Focusable().FocusRing(false)
			if f.focus {
				panel.Focus()
				f.focus = false
			}
			panel.Children(func() {
				outer := axis
				axis = vertical
				render(c, f.content)
				axis = outer
			})
			if !f.open {
				return
			}
			if panel.OverlayShortcut(0, ui.KeyEscape) || !f.keepOpen && panel.PressedOutside() {
				f.open = false
				if f.onClosed != nil {
					Post(f.onClosed)
				}
			}
		})
	})
	return e
}

func (f *FlyoutPanelWidget) fillsAcross() bool { return fillsAcross(f.owner) }

func (f *FlyoutPanelWidget) children() []Control { return []Control{f.owner, f.content} }
