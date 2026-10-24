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
	id         int
	hidden     bool
	disabled   bool
	wantFocus  bool // Focus asked for the keyboard focus
	focusIn    int  // the view that last built it focused, noView for none
	builtIn    int  // the view that last built it
	tooltip    string
	ownStyle   Style   // what the control's own methods set
	applied    []Style // Apply's styles
	style      Style   // applied merged, ownStyle last: what frames resolve
	styled     bool    // style sets anything
	textKey    any     // textKey(id), boxed once
	place      layout
	bound      binding // Bind's, for the controls that can be bound
	onFocusOut func()
	click      func()
	menu       []MenuItem
}

// noView is core.focusIn for a widget no view has seen focused.
const noView = -1

// building is the id of the view building: a window's, or content's.
// Only views touch it, under mu.
var building int

func (c *common[W]) init(self *W) {
	c.self, c.id, c.focusIn = self, newID(), noView
	c.textKey = textKey(c.id)
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
		return ui.Element{}
	}
	keeping.key(c, b)
	e := w.build(c)
	b.place.apply(e)
	if b.styled {
		p := b.style.effective(c.Theme().Dark, e.Hovered(), e.Focused(), e.Pressed(), b.disabled)
		p.apply(c, e, b.textKey)
	}
	if b.disabled {
		e.Disabled(true)
	}
	if b.tooltip != "" {
		e.Tooltip(b.tooltip)
	}
	if h := b.click; h != nil {
		e.OnClick(func() { Post(h) })
	}
	if menu := b.menu; len(menu) > 0 {
		e.ContextMenu(func(m *ui.Menu) { buildMenu(m, menu) })
	}
	if b.wantFocus {
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
		axis, building = vertical, id
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
