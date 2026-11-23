package widget

import (
	"slices"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// RibbonWidget is a command bar of tabs, each holding groups of large and
// small commands, as Office's and DevExpress's ribbon; put it first in a
// form's content.
type RibbonWidget struct {
	common[RibbonWidget]
	tabs []*ribbonTab
	// keys are tabs as ui takes them, a new slice when the tabs change: ui
	// keeps the last frame's.
	keys      []any
	tab       mirror[int]
	collapsed bool
	// expanded is set by a click on a tab that expanded the ribbon, so that
	// the double click it starts does not collapse it again; swallow is
	// set by a double click, whose release clicks the tab once more and
	// must not expand what it collapsed. Under mu, while building.
	expanded, swallow bool
	// changed and toggle are ui's callbacks, made once: a closure made in
	// each build is an allocation in each frame.
	changed, toggle func()
}

// ribbonTab is a tab of a ribbon, and ribbonGroup a group of a tab. A tab
// is its own key in ui's tabs: a pointer to a value that is not empty.
type ribbonTab struct {
	title  string
	groups []*ribbonGroup
}

type ribbonGroup struct {
	title string
	items []ribbonItem
}

type ribbonItem struct {
	w     Control
	large bool
}

// Ribbon creates a ribbon with no tab.
func Ribbon() *RibbonWidget {
	r := &RibbonWidget{}
	r.init(r)
	r.changed = onChange(&r.tab, nil)
	r.toggle = func() { r.change(func() { r.collapsed = !r.collapsed }) }
	return r
}

// AddTab adds a tab titled title after the others.
func (r *RibbonWidget) AddTab(title string) *RibbonWidget {
	r.change(func() { r.addTab(title) })
	return r
}

// addTab adds a tab. Under mu.
func (r *RibbonWidget) addTab(title string) {
	t := &ribbonTab{title: title}
	r.tabs = append(r.tabs, t)
	r.keys = append(slices.Clip(r.keys), t)
}

// AddGroup adds a group titled title to the last tab (to a new tab with
// no title before any).
func (r *RibbonWidget) AddGroup(title string) *RibbonWidget {
	r.change(func() { r.addGroup(title) })
	return r
}

// addGroup adds a group to the last tab. Under mu.
func (r *RibbonWidget) addGroup(title string) {
	if len(r.tabs) == 0 {
		r.addTab("")
	}
	t := r.tabs[len(r.tabs)-1]
	t.groups = append(t.groups, &ribbonGroup{title: title})
}

// AddLarge adds buttons to the last group (a new one with no title before
// any), each drawn large: its icon above its text.
func (r *RibbonWidget) AddLarge(buttons ...*ToolButtonWidget) *RibbonWidget {
	r.change(func() {
		for _, b := range buttons {
			if b != nil {
				r.add(b, true)
			}
		}
	})
	return r
}

// AddSmall adds controls to the last group, stacked three to a column: tool
// buttons, drop-down buttons, combo boxes, check boxes.
func (r *RibbonWidget) AddSmall(controls ...Control) *RibbonWidget {
	r.change(func() {
		for _, c := range controls {
			if c != nil {
				r.add(c, false)
			}
		}
	})
	return r
}

// add adds w to the last group, unless the ribbon holds it already, as
// Fields.Row refuses a control the form holds. Under mu.
func (r *RibbonWidget) add(w Control, large bool) {
	for _, t := range r.tabs {
		for _, g := range t.groups {
			for _, it := range g.items {
				if it.w == w {
					return
				}
			}
		}
	}
	t := len(r.tabs) - 1
	if t < 0 || len(r.tabs[t].groups) == 0 {
		r.addGroup("")
		t = len(r.tabs) - 1
	}
	g := r.tabs[t].groups[len(r.tabs[t].groups)-1]
	g.items = append(g.items, ribbonItem{w, large})
	adopt(&r.core, w)
}

// Tab returns the index of the tab shown; SetTab shows tab i.
func (r *RibbonWidget) Tab() int { mu.Lock(); defer mu.Unlock(); return r.tab.value }

func (r *RibbonWidget) SetTab(i int) *RibbonWidget {
	r.change(func() { r.tab.set(i) })
	return r
}

// SetCollapsed shows the tabs alone, or the tabs and their groups; a double
// click on a tab and Ctrl+F1 toggle it. Collapsed reports it.
func (r *RibbonWidget) SetCollapsed(on bool) *RibbonWidget {
	r.change(func() { r.collapsed = on })
	return r
}

func (r *RibbonWidget) Collapsed() bool { mu.Lock(); defer mu.Unlock(); return r.collapsed }

func (r *RibbonWidget) build(c *ui.Context) ui.Element {
	col := ui.Column(c).AlignSelf(ui.Stretch)
	if len(r.tabs) == 0 {
		return col
	}
	return col.Children(func() {
		p, _ := r.tab.bind()
		strip := r.strip(c, p)
		r.inner = strip // Focus (an access key) goes to the tabs, as a TabControl's
		c.OnWindowShortcut(ui.Ctrl, ui.KeyF1, r.toggle)
		// Tabs clamps *p while it builds, without OnChange: see it here.
		if *p != r.tab.value {
			r.tab.value = *p
		}
		if i := *p; !r.collapsed && i >= 0 && i < len(r.tabs) {
			r.groups(c, r.tabs[i])
		}
	})
}

// strip builds the tabs, as a TabControl's: ui's look, or the skin's
// TabHeader look over each tab. A click on a tab expands a collapsed
// ribbon, and a double click collapses it. Under mu.
func (r *RibbonWidget) strip(c *ui.Context, p *int) ui.Element {
	t := c.Theme()
	look := lookOf(skinhook.KindTabHeader)
	tabs := ui.TabsKeyedBase(c, p, r.keys)
	var clicked, double, pressed bool
	tabs.List.Gap(t.Space(1)).OnChange(r.changed).Children(func() {
		for i, tb := range r.tabs {
			var tab ui.Element
			if look != nil {
				tab = headerTab(c, tabs, i, i == *p, r.keys, look, r.disabled)
			} else {
				tab = documentTab(c, tabs, i, i == *p) // the shown one joins the groups row, Background as it
			}
			tab.Children(func() { ui.Text(c, tb.title).SingleLine() })
			if i == *p && r.wantFocus && !c.Measuring() {
				// Focus goes to the tab shown, not to the strip around it, which
				// render would focus.
				tab.Focus()
				r.wantFocus = false
			}
			double = tab.DoubleClicked() || double
			clicked = tab.Clicked() || clicked
			pressed = tab.Pressed() || pressed
		}
	})
	switch {
	case double:
		// The first click of it expanded a collapsed ribbon: it stays so.
		r.collapsed = r.collapsed || !r.expanded
		r.swallow = true
	case clicked && r.swallow:
		r.swallow = false // the release of the double click
	case clicked:
		r.expanded = r.collapsed
		r.collapsed = false
	case !pressed:
		r.swallow = false // the double click's press ended with no click
	}
	return tabs.List
}

// groups builds the groups of tab tb in a row filled with the skin's Chrome
// (Background without a skin, as the shown tab above it), a line under it,
// which clips what the window has no room for. Under mu.
func (r *RibbonWidget) groups(c *ui.Context, tb *ribbonTab) {
	t := c.Theme()
	fill := t.Background
	if tk, _ := skinOf(c); tk != nil {
		fill = tk.Chrome
	}
	ui.Row(c).Background(fill).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Clip().Children(func() {
		outer := axis
		axis = vertical
		defer func() { axis = outer }()
		for i, g := range tb.groups {
			r.group(c, g, i == len(tb.groups)-1)
		}
	})
}

// group builds g: its items in a row three small rows high, its title under
// them, and a line after it but the last. Under mu.
func (r *RibbonWidget) group(c *ui.Context, g *ribbonGroup, last bool) {
	t := c.Theme()
	row := t.Metrics.ControlHeight // ui scales Metrics with the text
	if row <= 0 {
		row = t.Rem(2.5)
	}
	gap := t.Space(0.5)
	high := 3*row + 2*gap
	col := ui.Column(c).Padding(t.Space(1), t.Space(1.5)).Gap(gap).AlignItems(ui.Center).Shrink(0)
	if g.title != "" {
		col.Label(g.title)
	}
	if !last {
		col.BorderWidth(0, 1, 0, 0).BorderColor(t.Border)
	}
	col.Children(func() {
		ui.Row(c).Height(high).Gap(t.Space(1)).AlignItems(ui.Start).Children(func() {
			for i := 0; i < len(g.items); {
				if g.items[i].large {
					r.large(c, g.items[i].w, high)
					i++
					continue
				}
				j := i + 1
				for j < len(g.items) && j-i < 3 && !g.items[j].large {
					j++
				}
				smalls := g.items[i:j]
				ui.Column(c).Gap(gap).Children(func() {
					for _, it := range smalls {
						if e := render(c, it.w); e != (ui.Element{}) {
							e.MinHeight(row)
							flatTool(it.w, e)
						}
					}
				})
				i = j
			}
		})
		if g.title != "" {
			ui.Text(c, g.title).TextColor(t.TextMuted).FontSize(t.Rem(0.85)).SingleLine()
		}
	})
}

// large builds w, a tool button, drawn large and high. Under mu.
func (r *RibbonWidget) large(c *ui.Context, w Control, high float32) {
	outer := toolLarge
	toolLarge = largeFace
	e := render(c, w)
	toolLarge = outer
	if e != (ui.Element{}) {
		e.Height(high)
		flatTool(w, e)
	}
}

// largeFace shows b's icon, 32 DIPs at the text scale, above its text,
// centred, the text in two lines at most no wider than twice the icon; it
// names e as iconFace does. Under mu, while b builds.
func largeFace(c *ui.Context, e ui.Element, b *ToolButtonWidget) {
	scale := c.Preferences().TextScale
	if scale <= 0 {
		scale = 1
	}
	side := 32 * scale
	name := b.text
	if name == "" {
		name = b.tooltip
	}
	e.Label(name).Children(func() {
		ui.Column(c).AlignItems(ui.Center).Gap(c.Theme().Space(1)).Children(func() {
			var icon ui.Element
			if b.icon != nil {
				icon = ui.Icon(c, b.icon)
			} else {
				icon = bitmapIcon(c, b.png)
			}
			icon.FontSize(side) // an icon is as high as its font size
			if b.text != "" {
				ui.Text(c, b.text).TextAlign(ui.Center).MaxLines(2).MaxWidth(2 * side)
			}
		})
	})
}

// flatTool gives a plain tool button of the ribbon no face until hovered,
// pressed or checked, as a tool strip's: outside ui.Toolbar a button has
// one. Under mu, after w builds.
func flatTool(w Control, e ui.Element) {
	if b, ok := w.(*ToolButtonWidget); ok && b.look == plainLook && !b.selected && !e.Hovered() && !e.Pressed() {
		e.Background(ui.Color{}).BorderColor(ui.Color{})
	}
}

func (r *RibbonWidget) children() []Control {
	var cs []Control
	for _, t := range r.tabs {
		for _, g := range t.groups {
			for _, it := range g.items {
				cs = append(cs, it.w)
			}
		}
	}
	return cs
}
