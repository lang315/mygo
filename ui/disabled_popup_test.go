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
