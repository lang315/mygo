// Package widget builds user interfaces from widgets that live on: make a
// widget once, keep it, and change it from any goroutine; the windows
// showing it redraw. It is ui, the immediate-mode toolkit, wrapped in the
// style of GTK wrappers such as gota:
//
//	var name *widget.TextBoxWidget
//	box := widget.VBox(8).Add(
//		widget.Label("Name").Bold(),
//		widget.TextBox().SetPlaceholder("Ann").AssignTo(&name),
//		widget.Button("Greet").OnClick(func() {
//			greeting.SetText("Hello, " + name.Text())
//		}),
//	)
//
// Handlers run one at a time on a goroutine of the package, never on the
// main thread, so they may block, as on a dialog.
package widget

import "github.com/egoist/mygo/ui"

// Control is a part of a user interface: a label, a box of controls, a
// table. The package's constructors make them, and Custom makes one of
// your own.
type Control interface {
	base() *core
	build(c *ui.Context) ui.Element
}

// core is the state every widget has.
type core struct {
	id          int
	hidden      bool
	disabled    bool
	wantFocus   bool // Focus asked for the keyboard focus
	focusIn     int  // the view that last built it focused, noView for none
	builtIn     int  // the view that last built it
	tooltip     string
	ownStyle    Style   // what the control's own methods set
	applied     []Style // Apply's styles
	style       Style   // applied merged, ownStyle last: what frames resolve
	styled      bool    // style sets anything
	textKey     any     // textKey(id), boxed once
	idKey       any     // id, boxed once, for keeper.key
	place       layout
	bound       binding // Bind's, for the controls that can be bound
	onFocusOut  func()
	click       func()
	menu        []MenuItem
	post        func(c *ui.Context, e ui.Element) // after the styles apply, set by build
	group       *radioGroup                       // the radio buttons added to the widget, a container
	dnd         dragDrop                          // drag sources and drop targets
	inertParent bool                              // el was disabled by a container, at the end of the last build
	el          ui.Element                        // the element of the last build, valid until it ends
	errMsg      string                            // ErrorProvider's
	errBy       *ErrorProviderWidget
	validating  func() bool                      // OnValidating's
	over        []func(p *ui.Painter, r ui.Rect) // drawn over the element, set by build
}

// noView is core.focusIn for a widget no view has seen focused.
const noView = -1

// building is the id of the view building: a window's, or content's.
// Only views touch it, under mu.
var building int

func (c *common[W]) init(self *W) {
	c.self, c.id, c.focusIn = self, newID(), noView
	c.textKey = textKey(c.id)
	c.idKey = c.id
}

// restyle merges the control's styles, its own last. Under mu.
func (b *core) restyle() {
	var s Style
	for _, a := range b.applied {
		s = s.Merge(a)
	}
	b.style = s.Merge(b.ownStyle)
	b.place = b.style.layout()
	// A control that only places itself has nothing to restyle each frame.
	rest := b.style
	rest.set &^= pLayout
	b.styled = !rest.empty()
}

// SetVisible shows or hides the widget. A hidden widget keeps its values,
// caret, selection and scroll position; an open popup closes, and hiding
// it takes its keyboard focus.
func (c *common[W]) SetVisible(v bool) *W {
	c.change(func() {
		c.hidden = !v
		if !v && c.focusIn != noView {
			c.focusIn = noView
			if c.onFocusOut != nil {
				Post(c.onFocusOut)
			}
		}
	})
	return c.self
}

// OnClick runs fn when the widget is clicked: gota's EventBox, on any
// widget. Buttons have their own OnClick.
func (c *common[W]) OnClick(fn func()) *W { c.change(func() { c.click = fn }); return c.self }

// ContextMenu shows items in a menu when the widget is right-clicked; no
// items remove it. Set it again when the items change.
func (c *common[W]) ContextMenu(items ...MenuItem) *W {
	c.change(func() { c.menu = items })
	return c.self
}

// Focus moves the keyboard focus to the widget when it next draws.
func (c *common[W]) Focus() *W { c.change(func() { c.wantFocus = true }); return c.self }

func (b *core) base() *core { return b }

// common gives the widget type W the methods every widget has. They
// return W, so calls chain.
type common[W any] struct {
	core
	self *W
}

// AssignTo stores the widget in *p, to keep a widget made inside a chain.
func (c *common[W]) AssignTo(p **W) *W { *p = c.self; return c.self }

// Visible reports whether the widget is shown.
func (c *common[W]) Visible() bool { mu.Lock(); defer mu.Unlock(); return !c.hidden }

// SetDisabled disables the widget, or enables it again.
func (c *common[W]) SetDisabled(d bool) *W { c.change(func() { c.disabled = d }); return c.self }

// SetTooltip sets the text shown while the pointer rests on the widget.
func (c *common[W]) SetTooltip(s string) *W { c.change(func() { c.tooltip = s }); return c.self }

// OnFocusOut runs fn when the widget loses the keyboard focus.
func (c *common[W]) OnFocusOut(fn func()) *W { c.change(func() { c.onFocusOut = fn }); return c.self }

// direction is how a box lays out its children.
type direction uint8

const (
	vertical direction = iota
	horizontal
)

// axis is the direction of the box being built, which the alignment and
// expansion of its children follow. Only views touch it, under mu.
var axis direction

// render builds w keyed globally by its id, so its ui state follows it
// wherever it moves in the window (keeper.key), and styles it. It builds
// nothing for a hidden widget. Under mu.
func render(c *ui.Context, w Control) ui.Element {
	b := w.base()
	if b.hidden {
		keeping.hold(b)
		b.dnd.endHovers()
		return ui.Element{}
	}
	keeping.key(c, b)
	e := w.build(c)
	b.el = e
	if f := buildingForm; f != nil {
		f.built = append(f.built, builtEl{w, e})
	}
	if fillsAcross(w) {
		e.AlignSelf(ui.Stretch) // before the widget's own alignment, which wins
	}
	b.place.apply(e)
	if b.styled {
		p := b.style.effective(c.Theme().Dark, e.Hovered(), e.Focused(), e.Pressed(), b.disabled)
		p.apply(c, e, b.textKey)
	}
	if b.post != nil {
		b.post(c, e)
		b.post = nil
	}
	if c.Inspecting() && !c.Measuring() {
		b.leaveNote(c, w, e)
	}
	b.dnd.apply(c, e)
	if b.disabled {
		e.Disabled(true)
	}
	if b.tooltip != "" && b.errMsg == "" {
		e.Tooltip(b.tooltip)
	}
	if b.errMsg != "" {
		// The value is invalid for assistive technology, and the message is
		// the tooltip too, ahead of the control's own.
		tip := b.errMsg
		if b.tooltip != "" {
			tip += "\n" + b.tooltip
			e.Description(b.tooltip) // the error comes first in the description
		}
		e.Error(b.errMsg)
		e.Tooltip(tip)
		rtl := c.Direction() == ui.RightToLeft
		danger := c.Theme().Danger
		b.over = append(b.over, func(p *ui.Painter, r ui.Rect) { paintErrorMark(p, r, rtl, danger) })
	}
	if len(b.over) > 0 {
		over := b.over
		e.DrawOver(func(p *ui.Painter, r ui.Rect) {
			for _, fn := range over {
				fn(p, r)
			}
		})
		b.over = nil
	}
	if h := b.click; h != nil {
		e.OnClick(func() { Post(h) })
	}
	if menu := b.menu; len(menu) > 0 {
		e.ContextMenu(func(m *ui.Menu) { buildMenu(m, menu) })
	}
	if b.wantFocus && !c.Measuring() { // the request is for the real window
		e.Focus()
		b.wantFocus = false
	}
	if b.onFocusOut != nil {
		switch f := e.Focused(); {
		case f:
			b.focusIn = building
		case b.focusIn == building:
			b.focusIn = noView
			Post(b.onFocusOut)
		}
	}
	return e
}

// content is a view showing w alone, filling the window.
func content(w Control) func(c *ui.Context) {
	id := newID()
	var k keeper
	return func(c *ui.Context) {
		mu.Lock()
		defer mu.Unlock()
		viewing.Store(true)
		defer viewing.Store(false)
		axis, building, buildingForm = vertical, id, nil
		k.begin()
		ui.Column(c).Fill().Children(func() { render(c, w) })
		k.end(c)
	}
}

// View returns a ui view showing w alone, for a ui.Tester or a window of
// your own: mygo.WindowOptions{Content: ui.View(widget.View(w))}. A setter
// does not redraw a window that shows a View, as only a Form's windows are
// tracked: Invalidate it yourself.
func View(w Control) func(c *ui.Context) { return content(w) }

// acrossFiller is a control that may stretch across the box that holds it.
type acrossFiller interface{ fillsAcross() bool }

// fillsAcross reports whether w stretches across the box that holds it: the
// controls that scroll or virtualize their rows (viewports: a list, table,
// tree, multi-line text, split, tab control, scroll panel) always do, as a
// row would otherwise center them at the height of all their content; the
// boxes and group boxes do when what they hold does, so that a box of labels
// stays centred and a box with a list passes the height down to it. The
// widget's own Align and VAlign still win (render). Under mu.
func fillsAcross(w Control) bool {
	f, ok := w.(acrossFiller)
	return ok && f.fillsAcross()
}
