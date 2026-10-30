package ui

import (
	"testing"
	"time"
	"unsafe"

	"github.com/egoist/mygo/i18n"
)

// ctxScene: a List with RowVersion whose rows show what show says of the
// window, and a count of the builds of rows.
func ctxScene(show func(c *Context, i int)) func() func(c *Context) {
	return func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			Column(c).Fill().Children(func() {
				List(c, s, 30, func(i int) {
					Row(c).Padding(4).Children(func() { show(c, i) })
				}).Grow(1)
			})
		}
	}
}

// A row that shows a value of the window is built when the value changes,
// and kept while it does not.
func TestRetainRowsShowingTheWindow(t *testing.T) {
	for _, tc := range []struct {
		name string
		show func(c *Context, i int)
		do   func(tt *Tester)
	}{
		{"Size", func(c *Context, i int) { w, h := c.Size(); Textf(c, "row %d in %v x %v", i, w, h) }, func(tt *Tester) { tt.SetSize(320, 400) }},
		{"Zoom", func(c *Context, i int) { Textf(c, "row %d at %v", i, c.Zoom()) }, func(tt *Tester) { tt.SetZoom(1.5) }},
		{"Preferences", func(c *Context, i int) {
			p := c.Preferences()
			if p.HighContrast {
				Textf(c, "row %d: more contrast", i)
			} else {
				Textf(c, "row %d", i)
			}
		}, func(tt *Tester) { tt.SetPreferences(Preferences{HighContrast: true, TextScale: 1}) }},
		{"Locale", func(c *Context, i int) { Textf(c, "row %d in %q", i, c.Locale()) }, func(tt *Tester) { tt.SetLocale("de") }},
		{"Region", func(c *Context, i int) { Textf(c, "row %d: %v", i, c.Region().FirstWeekday) }, func(tt *Tester) {
			tt.SetRegion(i18n.Region{Locale: "en-GB", FirstWeekday: time.Monday, HourCycle: 24})
		}},
		{"Vibrancy", func(c *Context, i int) { Textf(c, "row %d vibrancy %v", i, c.Vibrancy()) }, func(tt *Tester) { tt.SetVibrancy(true) }},
		{"TitleBar", func(c *Context, i int) { Textf(c, "row %d bar %v", i, c.TitleBar().Height) }, func(tt *Tester) { tt.SetTitleBar(TitleBar{Height: 30}) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newPair(t, 300, 400, ctxScene(tc.show))
			p.idle(3)
			built, kept := p.kept.rt.rowsBuilt, p.kept.rt.rowsKept
			p.frame("idle")
			if b, k := p.kept.rt.rowsBuilt-built, p.kept.rt.rowsKept-kept; b != 0 || k == 0 {
				t.Errorf("idle: %d rows built, %d kept; want none built (the value did not change)", b, k)
			}
			p.both(tc.do)
			p.frame("changed")
			p.frame("changed 2")
		})
	}
}

// Rows that call Shortcut or Now are built in every frame: Shortcut registers
// the shortcut at each build, and the time is a different one at each.
func TestRetainRowsCallingShortcutAndNowAreBuilt(t *testing.T) {
	for _, tc := range []struct {
		name string
		show func(c *Context, i int)
	}{
		{"Shortcut", func(c *Context, i int) {
			if c.Shortcut(Cmd, KeyK) {
				Textf(c, "row %d: Cmd+K", i)
			} else {
				Textf(c, "row %d", i)
			}
		}},
		{"Now", func(c *Context, i int) { Textf(c, "row %d at %v", i, c.Now().Second()) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newPair(t, 300, 400, ctxScene(tc.show))
			p.idle(3)
			built := p.kept.rt.rowsBuilt
			p.frame("idle")
			if p.kept.rt.rowsBuilt == built {
				t.Errorf("no row was built in an idle frame")
			}
			p.both(func(tt *Tester) { tt.Key(Cmd, KeyK) })
			p.compare("shortcut")
		})
	}
}

// A sidebar List in a layout route marks the row of the path shown, read in the
// row through c.Router(); navigation keeps the layout, so the list stays.
func TestRetainRowsReadingTheRouter(t *testing.T) {
	var routers []*Router
	paths := []string{"/s/a", "/s/b", "/s/c"}
	p := newPair(t, 400, 300, func() func(c *Context) {
		r := NewRouter("/s/a")
		routers = append(routers, r)
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		return func(c *Context) {
			r.View(c, func(rt *Route) {
				rt.Match("/s/{rest...}")
				Row(c).Grow(1).Children(func() {
					Column(c).Width(140).Children(func() {
						List(c, s, len(paths), func(i int) {
							if c.Router().Path() == paths[i] {
								Textf(c, "> %s", paths[i])
							} else {
								Textf(c, "  %s", paths[i])
							}
						}).Grow(1)
					})
					rt.View(c, func(rt *Route) { Textf(c, "page %s", rt.Path()) })
				})
			})
		}
	})
	p.idle(3)
	built, kept := p.kept.rt.rowsBuilt, p.kept.rt.rowsKept
	p.frame("idle")
	if b, k := p.kept.rt.rowsBuilt-built, p.kept.rt.rowsKept-kept; b != 0 || k == 0 {
		t.Errorf("idle: %d rows built, %d kept; want none built", b, k)
	}
	for _, r := range routers {
		r.Push("/s/b")
	}
	p.both(func(tt *Tester) { tt.Frame() })
	p.compare("pushed")
	p.frame("pushed 2")
}

// The window's zoom changes in the last pass of a frame: a row built before it
// is linked only while the zoom is the one it read, not the one of the end.
func TestRetainRowsReadingAValueThatChangesInTheLastPass(t *testing.T) {
	gen := 0
	p := newPair(t, 300, 400, func() func(c *Context) {
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 0 }}
		seen, left := 0, 0
		return func(c *Context) {
			if seen != gen {
				seen, left = gen, 3
			}
			Column(c).Fill().Children(func() {
				List(c, s, 10, func(i int) {
					Textf(c, "row %d at %.2f", i, c.Zoom())
				}).Grow(1)
			})
			if left > 0 {
				left--
				c.SetZoom(c.Zoom() + 0.25)
			}
		}
	})
	p.idle(2)
	gen = 1
	p.both(func(tt *Tester) { tt.Frame() })
	p.both(func(tt *Tester) { tt.Frame() })
	p.both(func(tt *Tester) { tt.Frame() })
	p.compare("after zooming three times in one frame")
	p.frame("settled")
}

// The layout walks and copies the rows of a frame: what a row read through the
// Context goes in a side slice, not in the row.
func TestListRowStaysSmall(t *testing.T) {
	if n := unsafe.Sizeof(listRow{}); n > 56 {
		t.Errorf("listRow is %d bytes, want at most 56: keep what rows read out of it (listFrame.reads)", n)
	}
}
