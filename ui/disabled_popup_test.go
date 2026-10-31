package ui

import (
	"fmt"
	"testing"
)

var popupOptions = []string{"apple", "apricot", "banana", "cherry", "date"}

// disabledPopupScene is a column of rows of an input of kind, which the app
// disables by off, each twin with its own state.
type disabledPopupScene struct {
	apps []*struct {
		choice []string
		off    bool
	}
}

func (d *disabledPopupScene) set(f func(off *bool)) {
	for _, a := range d.apps {
		f(&a.off)
	}
}

func (d *disabledPopupScene) mk(kind string, n int) func() func(c *Context) {
	return func() func(c *Context) {
		a := &struct {
			choice []string
			off    bool
		}{choice: make([]string, n)}
		for i := range a.choice {
			a.choice[i] = popupOptions[i%len(popupOptions)]
		}
		d.apps = append(d.apps, a)
		s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 1 }}
		return func(c *Context) {
			Column(c).Fill().Disabled(a.off).Children(func() {
				List(c, s, n, func(i int) {
					Row(c).Padding(4).Gap(8).Children(func() {
						Textf(c, "row %d", i)
						switch kind {
						case "combo":
							Combobox(c, &a.choice[i], popupOptions).Width(180)
						case "auto":
							Autocomplete(c, &a.choice[i], popupOptions).Width(180)
						default:
							TextInput(c, &a.choice[i]).Width(180)
						}
					})
				}).Grow(1)
			})
		}
	}
}

// A focused input whose parent is disabled, and a click outside: the twins
// agree, the focus is nowhere and nothing is left open, whether the rows are
// kept or built.
func TestFocusedInputOfDisabledParentLosesFocusToAClickOutside(t *testing.T) {
	for _, kind := range []string{"combo", "auto", "textinput"} {
		t.Run(kind, func(t *testing.T) {
			var d disabledPopupScene
			p := newPair(t, 500, 400, d.mk(kind, 24))
			p.idle(3)
			p.both(func(tt *Tester) { tt.ClickAt(66, 284) })
			p.frame("focused")
			if p.kept.rt.focused == 0 {
				t.Fatal("the click did not focus an input")
			}
			d.set(func(off *bool) { *off = true })
			p.frame("parent disabled")
			p.both(func(tt *Tester) { tt.ClickAt(5, -8) })
			p.frame("click outside")
			p.frame("click outside +1")
			for _, x := range []*Tester{p.kept, p.built} {
				if x.rt.focused != 0 {
					t.Errorf("focus %d after a click outside, want none", x.rt.focused)
				}
			}
		})
	}
}

// An input that is disabled shows no popup, whatever its afterInput did.
func TestDisabledInputClosesItsPopup(t *testing.T) {
	var text string
	var off bool
	var parts *comboboxParts
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Disabled(off).Children(func() {
			parts = comboboxBase(c, &text, true)
			parts.Popup(func(panel *node) {
				for _, o := range popupOptions {
					parts.Item(o)
				}
			})
		})
	}, 300, 300)
	tt.Frame()
	tt.ClickAt(parts.Input.Bounds().X+5, parts.Input.Bounds().Y+5)
	tt.Frame()
	if !parts.Open() {
		t.Fatal("a click did not open the popup")
	}
	off = true
	tt.Frame()
	tt.Frame()
	if parts.Open() {
		t.Errorf("the popup of a disabled input is open")
	}
	if parts.Input.expanded {
		t.Errorf("a disabled input is expanded")
	}
}

// The focus goes back to the element that opened an overlay once it is gone,
// unless that element is disabled by then: the focus stays where it is, nowhere.
func TestRestoreFocusSkipsADisabledOpener(t *testing.T) {
	for _, disable := range []bool{false, true} {
		t.Run(fmt.Sprintf("disabled=%v", disable), func(t *testing.T) {
			var open, off bool
			var opener *node
			var inside string
			tt := coreNewTester(func(c *context) {
				coreColumn(c).Children(func() {
					coreBox(c).Disabled(off).Children(func() {
						opener = coreTextInputBase(c, new(string))
					})
					if open {
						coreOverlay(c, func() {
							coreBox(c).Padding(8).Children(func() { coreTextInputBase(c, &inside) })
						})
					}
				})
			}, 300, 300)
			tt.Frame()
			opener.Focus()
			tt.Frame()
			open = true
			tt.Frame()
			// The focus moves into the overlay.
			tt.rt.focused = 0
			for id, s := range tt.rt.states {
				if s.editor != nil && id != opener.id {
					tt.rt.focused = id
				}
			}
			if tt.rt.focused == 0 {
				t.Fatal("no input in the overlay")
			}
			tt.Frame()
			off = disable
			tt.Frame()
			open = false
			tt.Frame()
			tt.Frame()
			// What is left of the focus is an id of the element gone with the overlay.
			if got := tt.rt.focused; disable && tt.rt.states[got] != nil {
				t.Errorf("an element has the focus, want none")
			} else if !disable && got != opener.id {
				t.Errorf("focus %d, want the opener's %d", got, opener.id)
			}
		})
	}
}

// restoreScene: an input that opens an overlay with an input inside, the
// opener in a box the app disables by off.
func restoreScene(t *testing.T) (tt *Tester, open, off *bool, opener **node) {
	open, off, opener = new(bool), new(bool), new(*node)
	var inside string
	var second *node
	tt = coreNewTester(func(c *context) {
		coreColumn(c).Children(func() {
			coreBox(c).Disabled(*off).Children(func() {
				*opener = coreTextInputBase(c, new(string))
			})
			second = coreTextInputBase(c, new(string)) // a second focusable after the opener
			if *open {
				coreOverlay(c, func() {
					coreBox(c).Padding(8).Children(func() { coreTextInputBase(c, &inside) })
				})
			}
		})
	}, 300, 300)
	tt.Frame()
	(*opener).Focus()
	tt.Frame()
	*open = true
	tt.Frame()
	tt.rt.focused = 0
	for id, s := range tt.rt.states {
		if s.editor != nil && id != (*opener).id && id != second.id {
			tt.rt.focused = id
		}
	}
	if tt.rt.focused == 0 {
		t.Fatal("no input in the overlay")
	}
	tt.Frame()
	return
}

func TestRestoreFocusToAnOpenerEnabledAgain(t *testing.T) {
	t.Run("enabled in the frame the overlay closes", func(t *testing.T) {
		tt, open, off, opener := restoreScene(t)
		*off = true
		tt.Frame()
		*off, *open = false, false
		tt.Frame()
		tt.Frame()
		if tt.rt.focused != (*opener).id {
			t.Errorf("focus %d, want opener %d", tt.rt.focused, (*opener).id)
		}
	})
	t.Run("enabled one frame before the close", func(t *testing.T) {
		tt, open, off, opener := restoreScene(t)
		*off = true
		tt.Frame()
		*off = false
		tt.Frame()
		*open = false
		tt.Frame()
		tt.Frame()
		if tt.rt.focused != (*opener).id {
			t.Errorf("focus %d, want opener %d", tt.rt.focused, (*opener).id)
		}
	})
	t.Run("enabled one frame after the close", func(t *testing.T) {
		tt, open, off, opener := restoreScene(t)
		*off = true
		tt.Frame()
		*open = false
		tt.Frame()
		*off = false
		tt.Frame()
		tt.Frame()
		t.Logf("after close+enable: focused=%d opener=%d live=%v", tt.rt.focused, (*opener).id, tt.rt.states[tt.rt.focused] != nil)
		tt.Key(0, KeyTab)
		tt.Frame()
		t.Logf("after Tab: focused=%d opener=%d live=%v", tt.rt.focused, (*opener).id, tt.rt.states[tt.rt.focused] != nil)
		if tt.rt.focused != (*opener).id {
			t.Errorf("focus %d, want opener %d", tt.rt.focused, (*opener).id)
		}
	})
}

// Rule 1 with the box disabled after its children are built (as
// Fieldset(...).Disabled(true)): IsDisabled is false at build.
func TestInputDisabledAfterItsBuildClosesItsPopup(t *testing.T) {
	var text string
	var off bool
	var parts *comboboxParts
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Children(func() {
			parts = comboboxBase(c, &text, true)
			parts.Popup(func(panel *node) {
				for _, o := range popupOptions {
					item := parts.Item(o)
					item.Children(func() { coreText(c, o) })
				}
			})
		}).Disabled(off)
	}, 300, 300)
	tt.Frame()
	tt.ClickAt(parts.Input.Bounds().X+5, parts.Input.Bounds().Y+5)
	tt.Frame()
	if !parts.Open() {
		t.Fatal("a click did not open the popup")
	}
	off = true
	for range 4 {
		tt.Frame()
	}
	if parts.Open() {
		t.Errorf("the popup of an input disabled after build is open")
		tt.Click("banana")
		tt.Frame()
		t.Logf("text after clicking an option of the disabled input: %q", text)
	}
}

// The opener is in a row of a List with RowVersion, kept, not built, while
// the column around it is disabled and enabled again.
func TestRestoreFocusToTheOpenerOfAKeptRow(t *testing.T) {
	for _, order := range []string{"same frame", "enable before close", "disabled at close"} {
		t.Run(order, func(t *testing.T) {
			type app struct{ off, open bool }
			var apps []*app
			p := newPair(t, 500, 400, func() func(c *Context) {
				a := &app{}
				apps = append(apps, a)
				vals := make([]string, 24)
				var inside string
				s := &ListState{Key: func(i int) any { return i }, RowVersion: func(int) uint64 { return 1 }}
				return func(c *Context) {
					Column(c).Fill().Disabled(a.off).Children(func() {
						List(c, s, 24, func(i int) {
							Row(c).Padding(4).Gap(8).Children(func() {
								Textf(c, "row %d", i)
								TextInput(c, &vals[i]).Width(180)
							})
						}).Grow(1)
					})
					if a.open {
						Overlay(c, func() { TextInput(c, &inside) })
					}
				}
			})
			set := func(f func(a *app)) {
				for _, a := range apps {
					f(a)
				}
			}
			p.idle(3)
			p.both(func(tt *Tester) { tt.ClickAt(66, 284) })
			p.frame("focused")
			opener := [2]uint64{p.kept.rt.focused, p.built.rt.focused}
			if opener[0] == 0 {
				t.Fatal("no opener")
			}
			set(func(a *app) { a.open = true })
			p.frame("open")
			for _, tt := range []*Tester{p.kept, p.built} {
				for id, s := range tt.rt.states {
					if s.editor != nil && tt.rt.within(id, overlayID) {
						tt.rt.focused = id
					}
				}
			}
			if p.kept.rt.focused == opener[0] {
				t.Fatal("no input in the overlay")
			}
			p.frame("focus inside")
			set(func(a *app) { a.off = true })
			p.frame("disabled")
			if order == "same frame" {
				set(func(a *app) { a.off, a.open = false, false })
				p.frame("enabled+closed")
			} else if order == "disabled at close" {
				set(func(a *app) { a.open = false })
				p.frame("closed disabled")
				for i, tt := range []*Tester{p.kept, p.built} {
					if tt.rt.focused == opener[i] {
						t.Errorf("twin %d: a disabled opener took the focus", i)
					}
				}
				return
			} else {
				set(func(a *app) { a.off = false })
				p.frame("enabled")
				set(func(a *app) { a.open = false })
				p.frame("closed")
			}
			p.frame("+1")
			for i, tt := range []*Tester{p.kept, p.built} {
				if tt.rt.focused != opener[i] {
					t.Errorf("twin %d: focus %d, want opener %d", i, tt.rt.focused, opener[i])
				}
			}
		})
	}
}
