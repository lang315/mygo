package widget

import (
	"slices"

	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget/internal/skinhook"
)

// SetClosable gives each tab an x that closes its page, as a middle click
// on the tab and Ctrl+F4 in the control (the page shown) do; Ctrl+Tab and
// Ctrl+Shift+Tab then show the next and the previous page. OnPageClosing
// may keep a page. SetClosable(false) takes them away.
func (n *TabControlWidget) SetClosable(on bool) *TabControlWidget {
	n.change(func() {
		switch {
		case !on:
			n.closable = nil
		case n.closable == nil:
			n.closable = closableTabs(n)
		}
	})
	return n
}

// OnPageClosing runs fn on the event goroutine when the user asks to close
// page, before it closes: fn returning false keeps it, as Form.OnClose's
// does. nil removes it, and pages close at once.
func (n *TabControlWidget) OnPageClosing(fn func(page Control) bool) *TabControlWidget {
	n.change(func() { n.onClosing = fn })
	return n
}

// askClose asks, on the event goroutine, to close the page of the tab k:
// OnPageClosing may keep it. It goes by k, not by index, so that pages
// added or removed meanwhile change nothing; a page gone by then is not
// asked about. When the page shown closes, OnSwitch runs.
func (n *TabControlWidget) askClose(k *pageKey) {
	Post(func() {
		mu.Lock()
		i, fn := slices.Index(n.keys, k), n.onClosing
		var page Control
		if i >= 0 {
			page = n.pages[i]
		}
		mu.Unlock()
		if i < 0 || fn != nil && !fn(page) {
			return
		}
		var switched func()
		n.change(func() {
			if i := slices.Index(n.keys, k); i >= 0 {
				if i == n.shownIndex() {
					switched = n.onSwitch
				}
				n.removePage(i)
			}
		})
		if switched != nil {
			switched()
		}
	})
}

// closableTabs returns the closable build of n (TabControlWidget.closable):
// the strip of tabs, each with an x, and the keys on col, around the strip
// and the page. It makes its functions once, and its keys only when the
// pages change, so that a frame makes none.
func closableTabs(n *TabControlWidget) func(c *ui.Context, col ui.Element, p *int) ui.Element {
	// held is the tab the middle button pressed on the last frame: a press
	// closes on its first frame alone. keys are n.keys as ui takes them, a
	// new slice when they change: ui keeps the last frame's. Under mu.
	var (
		held *pageKey
		keys []any
	)
	changed := func() {
		if n.page.commit() {
			mu.Lock()
			fn := n.onSwitch
			mu.Unlock()
			if fn != nil {
				Post(fn)
			}
		}
	}
	closeShown := func() {
		mu.Lock()
		var k *pageKey
		if len(n.keys) > 0 {
			k = n.keys[n.shownIndex()]
		}
		mu.Unlock()
		if k != nil {
			n.askClose(k)
		}
	}
	step := func(d int) func() {
		return func() {
			var switched func()
			n.change(func() {
				if l := len(n.pages); l > 1 {
					n.page.set((n.shownIndex() + d + l) % l)
					switched = n.onSwitch
				}
			})
			if switched != nil {
				Post(switched)
			}
		}
	}
	next, prev := step(1), step(-1)
	return func(c *ui.Context, col ui.Element, p *int) ui.Element {
		fresh := len(keys) != len(n.keys)
		for i := 0; !fresh && i < len(keys); i++ {
			fresh = keys[i] != n.keys[i]
		}
		if fresh {
			keys = make([]any, len(n.keys))
			for i, k := range n.keys {
				keys[i] = k // a tab keeps the focus of its page, whatever its index
			}
		}
		t := c.Theme()
		// A disabled control's keys do nothing: nothing in it has the focus.
		col.OnShortcut(ui.Ctrl, ui.KeyF4, closeShown).OnShortcut(ui.Ctrl, ui.KeyTab, next).OnShortcut(ui.Ctrl|ui.Shift, ui.KeyTab, prev)
		look := lookOf(skinhook.KindTabHeader)
		side := 12 * c.Preferences().TextScale
		tabs := ui.TabsKeyedBase(c, p, keys)
		var pressing *pageKey
		tabs.List.Gap(t.Space(1)).Children(func() {
			for i, label := range n.labels {
				k, on := n.keys[i], i == *p
				var tab ui.Element
				if look != nil {
					tab = n.headerTab(c, tabs, i, on, keys, look)
				} else {
					tab = documentTab(c, tabs, i, on)
				}
				if b, ok := tab.PressedButton(); ok && b == 2 {
					pressing = k
					if held != k {
						n.askClose(k)
					}
				}
				// Named by its label alone, as without the x, which Ctrl+F4 stands for.
				tab.Label(label).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
					ui.Text(c, label).SingleLine()
					x := ui.ButtonBase(c).Label(l10n.Close()).Padding(t.Space(0.5)).Radius(t.Radius)
					if x.Hovered() {
						x.Background(t.Text.Alpha(0.12))
					}
					x.Children(func() { ui.Icon(c, closeIcon).FontSize(side) })
					if x.Clicked() {
						n.askClose(k)
					}
				})
			}
		})
		held = pressing
		return tabs.List.OnChange(changed)
	}
}

// documentTab makes tab i of tabs as a document's, on while open: the open
// one filled as the page under it, with a line on its top and sides, the
// others unfilled until hovered. It uses no accent, which a window that is
// not active fades. The caller gives it its children. Under mu.
func documentTab(c *ui.Context, tabs ui.TabsParts, i int, on bool) ui.Element {
	t := c.Theme()
	tab := tabs.Tab(i).Padding(t.Space(2), t.Space(2), t.Space(2), t.Space(3)).Radius(t.Radius, t.Radius, 0, 0)
	if h := t.Metrics.TabHeight; h > 0 {
		tab.MinHeight(h)
	}
	switch {
	case on:
		tab.Background(t.Background).BorderWidth(1, 1, 0, 1).BorderColor(t.Border).TextColor(t.Text).FontWeight(600)
	case tab.Hovered():
		tab.Background(t.SurfaceHover).TextColor(t.Text)
	default:
		tab.TextColor(t.TextMuted)
	}
	return tab
}
