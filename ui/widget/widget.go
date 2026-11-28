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

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

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
	hides       int // views that did not build it, and shows of its form, for a popup of its own to close
	disabled    bool
	selected    bool // checked or on, set by build: the Style's Selected applies
	wantFocus   bool // Focus asked for the keyboard focus
	focusIn     int  // the view that last built it focused, noView for none
	builtIn     int  // the view that last built it
	tooltip     string
	desc        string  // the description of its Fields row, the tooltip after it (Describe), set by the form as it builds
	accName     string  // SetAccessibleName's
	ownStyle    Style   // what the control's own methods set
	applied     []Style // Apply's styles
	style       Style   // applied merged, ownStyle last: what frames resolve
	styled      bool    // style sets anything
	skinKind    uint8   // 1 + the skinhook.Kind of the skin's look the control takes, 0 for none (lookAs)
	status      Status  // the status a Badge or an InfoBar shows, which the status roles resolve to (styling)
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
	inner       ui.Element                        // set by a build that returns a box around its control: the control, which render focuses, names and describes
	errMsg      string                            // ErrorProvider's
	errBy       *ErrorProviderWidget
	validating  func() bool                      // OnValidating's
	rules       *ruleList                        // Rule's
	over        []func(p *ui.Painter, r ui.Rect) // drawn over the element, set by build

	// superTip is SetSuperTip's, nil for none, and buildTip builds it, set
	// with it: an app that sets none does not link the builder.
	superTip *superTipState
	buildTip func(c *ui.Context, ctl ui.Element, b *core)
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
	c.change(func() { c.setVisible(v) })
	return c.self
}

// setVisible is SetVisible, and how Shows of a ComboBox shows and hides.
// Under mu.
func (b *core) setVisible(v bool) {
	b.hidden = !v
	if !v && b.focusIn != noView {
		b.focusIn = noView
		if b.onFocusOut != nil {
			Post(b.onFocusOut)
		}
	}
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

// lookAs makes the control take the skin's look of kind k.
func (b *core) lookAs(k skinhook.Kind) { b.skinKind = uint8(k) + 1 }

// lookStyle returns the skin's look of the control's kind, nil for none.
func (b *core) lookStyle() *Style {
	if b.skinKind == 0 {
		return nil
	}
	return lookOf(skinhook.Kind(b.skinKind - 1))
}

// lookOf returns the skin's look of kind k, nil for none. It allocates
// nothing: the skin boxed it once.
func lookOf(k skinhook.Kind) *Style {
	s, _ := skinhook.Look(k).(*Style)
	return s
}

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

// SetAccessibleName names the widget for assistive technology and a Tester
// (ByLabel), in place of the name it takes from its text, its tooltip or the
// label of its Fields row: an icon-only button, say. "" gives that name back.
func (c *common[W]) SetAccessibleName(s string) *W {
	c.change(func() { c.accName, nameControl = s, nameControlOf })
	return c.self
}

// nameControl names the control a widget built, set by SetAccessibleName: an
// app that names none does not link the search for the control. Under mu.
var nameControl func(w Control, e ui.Element, name string)

// nameControlOf names the control e holds, or e, a container's group.
func nameControlOf(w Control, e ui.Element, name string) {
	if _, ok := w.(interface{ children() []Control }); ok {
		e.Label(name) // a container names its group
	} else {
		e.AccessibleName(name) // the control, not a box around it
	}
}

// OnFocusOut runs fn when the widget loses the keyboard focus, and when
// its form closes with the focus in it (before a dialog's Run returns;
// the app quitting does not wait for it).
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
	b.selected = false // a build that can be selected sets it: a ToolButton only while a toggle
	outer := styling
	styling = b.status // the status roles resolve to the control's own, in its build and its styles
	from := 0
	if f := buildingForm; f != nil {
		from = len(f.built)
	}
	e := w.build(c)
	// ctl is the control in e, which takes the focus and is named and
	// described: e itself, but where the build put a box around it.
	ctl := e
	if in := b.inner; in != (ui.Element{}) {
		ctl, b.inner = in, ui.Element{}
	}
	b.el = ctl
	if f := buildingForm; f != nil {
		f.built = append(f.built, builtEl{w, ctl, from})
	}
	if fillsAcross(w) {
		e.AlignSelf(ui.Stretch) // before the widget's own alignment, which wins
	}
	b.place.apply(e)
	if b.accName != "" {
		nameControl(w, ctl, b.accName)
	}
	if look := b.lookStyle(); b.styled || look != nil {
		dark, hover, focused, pressed := c.Theme().Dark, e.Hovered(), ctl.Focused(), e.Pressed()
		var p props
		if look != nil {
			// The skin's look under the control's own styles, property by
			// property: merging the Styles would allocate their variants.
			p = look.effective(dark, hover, b.selected, focused, pressed, b.disabled)
			if b.styled {
				own := b.style.effective(dark, hover, b.selected, focused, pressed, b.disabled)
				p.over(&own)
			}
		} else {
			p = b.style.effective(dark, hover, b.selected, focused, pressed, b.disabled)
		}
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
	if b.desc != "" {
		// Before the tooltip, which describes the control only where nothing
		// else does: desc holds it already.
		ctl.Description(b.desc)
	}
	tooltip := b.tooltip
	if b.buildTip != nil {
		tooltip = "" // the SuperTip takes its place
		b.buildTip(c, ctl, b)
	}
	if tooltip != "" && b.errMsg == "" {
		ctl.Tooltip(tooltip)
	}
	if b.errMsg != "" {
		// The value is invalid for assistive technology, and the message is
		// the tooltip too, ahead of the control's own.
		tip := b.errMsg
		if tooltip != "" {
			tip += "\n" + tooltip
			if b.desc == "" {
				ctl.Description(tooltip) // the error comes first in the description
			}
		}
		ctl.Error(b.errMsg)
		ctl.Tooltip(tip)
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
		ctl.Focus()
		b.wantFocus = false
	}
	if b.onFocusOut != nil || b.rules != nil {
		switch f := ctl.Focused(); {
		case f:
			b.focusIn = building
		case b.focusIn == building:
			b.focusIn = noView
			if b.onFocusOut != nil {
				buildingForm.postFocusOut(b.onFocusOut)
			}
			if b.rules != nil {
				Post(b.rules.leave) // after OnFocusOut: SetValidateOnLeave's providers check
			}
		}
	}
	styling = outer
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
