package widget

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

// NavBarWidget is a side panel of groups, each a header that opens and
// closes its items: an icon and a text, one of them selected, as the
// navigation pane of an app (DevExpress's NavBarControl).
type NavBarWidget struct {
	common[NavBarWidget]
	groups   []*navGroup // pointers: ui writes a group's open state after the view returns
	keys     map[string]struct{}
	selected string
	onSelect func(key string)
	rows     []ui.Element // the items of the group being built, reused
}

// navGroup is a group of a NavBar: one with no title has no header and
// stays open.
type navGroup struct {
	title string
	open  mirror[bool]
	items []*navItem
}

// navItem is an item of a NavBar; click is its OnClick, made once.
type navItem struct {
	key, text string
	icon      *ui.SVG
	click     func()
}

// NavBar creates a NavBar with no group.
func NavBar() *NavBarWidget {
	n := &NavBarWidget{}
	n.init(n)
	return n
}

// AddGroup adds a group titled title, open, after the others.
func (n *NavBarWidget) AddGroup(title string) *NavBarWidget {
	n.change(func() { n.addGroup(title) })
	return n
}

// addGroup adds an open group. Under mu.
func (n *NavBarWidget) addGroup(title string) {
	g := &navGroup{title: title}
	g.open.set(true)
	n.groups = append(n.groups, g)
}

// AddItem adds an item to the last group added: key names it, text shows
// beside icon (nil for none). Before any group it adds one with no title,
// which has no header and stays open. A key given twice panics.
func (n *NavBarWidget) AddItem(key, text string, icon *ui.SVG) *NavBarWidget {
	n.change(func() {
		if _, ok := n.keys[key]; ok {
			panic(fmt.Sprintf("widget: NavBar: duplicate key %q: a key stands for one item", key))
		}
		if n.keys == nil {
			n.keys = map[string]struct{}{}
		}
		n.keys[key] = struct{}{}
		if len(n.groups) == 0 {
			n.addGroup("")
		}
		it := &navItem{key: key, text: text, icon: icon}
		it.click = func() { n.pick(key) }
		g := n.groups[len(n.groups)-1]
		g.items = append(g.items, it)
	})
	return n
}

// pick selects the item of key as the user clicked it, and runs OnSelect
// if that changed the selection.
func (n *NavBarWidget) pick(key string) {
	var fn func(string)
	n.change(func() {
		if n.selected == key {
			return
		}
		n.selected, fn = key, n.onSelect
	})
	if fn != nil {
		Post(func() { fn(key) })
	}
}

// Selected returns the key of the selected item, "" for none.
func (n *NavBarWidget) Selected() string { mu.Lock(); defer mu.Unlock(); return n.selected }

// SetSelected selects the item of key, firing no event; "" selects none,
// and a key that is not an item is ignored.
func (n *NavBarWidget) SetSelected(key string) *NavBarWidget {
	n.change(func() {
		if _, ok := n.keys[key]; ok || key == "" {
			n.selected = key
		}
	})
	return n
}

// OnSelect runs fn on the event goroutine with the key of the item the
// user selected, when it changes. A setter does not run it.
func (n *NavBarWidget) OnSelect(fn func(key string)) *NavBarWidget {
	n.change(func() { n.onSelect = fn })
	return n
}

// SetGroupOpen opens or closes the group of index i, in the order added.
func (n *NavBarWidget) SetGroupOpen(i int, open bool) *NavBarWidget {
	n.change(func() {
		if i >= 0 && i < len(n.groups) && n.groups[i].title != "" {
			n.groups[i].open.set(open)
		}
	})
	return n
}

// GroupOpen reports whether the group of index i is open.
func (n *NavBarWidget) GroupOpen(i int) bool {
	mu.Lock()
	defer mu.Unlock()
	if i < 0 || i >= len(n.groups) {
		return false
	}
	return n.groups[i].open.value // true for a group with no title, which SetGroupOpen leaves
}

func (n *NavBarWidget) build(c *ui.Context) ui.Element {
	t := c.Theme()
	acc := ui.AccordionBase(c)
	var first ui.Element // the first header, else the first item: what Focus and an access key focus
	acc.Root.Children(func() {
		for _, g := range n.groups {
			if g.title == "" {
				if e := n.items(c, g); first == (ui.Element{}) {
					first = e
				}
				continue
			}
			open, _ := g.open.bind()
			sec := acc.Item(open)
			h := sec.Trigger.AlignItems(ui.Center).Gap(t.Space(2)).Padding(t.Space(2), t.Space(3)).
				Background(t.Surface).BorderWidth(0, 0, 1, 0).BorderColor(t.Border)
			if first == (ui.Element{}) {
				first = h
			}
			h.Children(func() {
				ui.Text(c, g.title).Grow(1).FontWeight(600)
				ui.Icon(c, chevronDown).FontSize(t.Space(3)).TextColor(t.TextMuted).Rotate(180 * sec.Progress())
			})
			sec.Panel(func() { n.items(c, g) })
		}
	})
	n.inner = first
	return acc.Root
}

// items builds the items of g, and returns the first. Under mu, building.
func (n *NavBarWidget) items(c *ui.Context, g *navGroup) ui.Element {
	t := c.Theme()
	side := 16 * c.Preferences().TextScale
	n.rows = n.rows[:0]
	for _, it := range g.items {
		e := ui.ButtonBase(c).Justify(ui.Start).AlignItems(ui.Center).Gap(t.Space(2)).
			Padding(t.Space(1.5), t.Space(3)).PaddingStart(t.Space(5))
		fg := t.Text
		switch {
		case it.key == n.selected && n.selected != "": // "" is none, even for an item of key ""
			var bg ui.Color
			bg, fg = rowSelected(t)
			e.Background(bg).Current(ui.CurrentPage)
		case e.Hovered():
			e.Background(rowHover(t))
		}
		e.TextColor(fg).OnClick(it.click)
		e.Children(func() {
			// Every item keeps the icon's room, so that the texts line up.
			if it.icon != nil {
				ui.Icon(c, it.icon).FontSize(side).Size(side, side).TextColor(fg)
			} else {
				ui.Box(c).Size(side, side)
			}
			ui.Text(c, it.text)
		})
		n.rows = append(n.rows, e)
	}
	// Up and Down move between the items of the group, stopping at its ends.
	for i, e := range n.rows {
		switch {
		case e.Shortcut(0, ui.KeyDown):
			if i+1 < len(n.rows) {
				n.rows[i+1].Focus()
			}
		case e.Shortcut(0, ui.KeyUp):
			if i > 0 {
				n.rows[i-1].Focus()
			}
		}
	}
	if len(n.rows) == 0 {
		return ui.Element{}
	}
	return n.rows[0]
}

// rowSelected is the fill of a selected row and its text: the theme's
// RowSelected and RowSelectedText (Text when zero), or where it sets no
// RowSelected the accent and AccentText, as ui's lists and sidebars take.
func rowSelected(t *ui.Theme) (bg, fg ui.Color) {
	if t.RowSelected.A == 0 {
		return t.Accent, t.AccentText
	}
	fg = t.RowSelectedText
	if fg.A == 0 {
		fg = t.Text
	}
	return t.RowSelected, fg
}

// rowHover is the fill of a row under the pointer: RowHover, else
// SurfaceHover.
func rowHover(t *ui.Theme) ui.Color {
	if t.RowHover.A != 0 {
		return t.RowHover
	}
	return t.SurfaceHover
}
