package ui

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// disabledKinds are the controls that open a popup or a menu: build makes
// one, which the test disables by itself ("self"), or by a container that
// is disabled before its children ("before") or after them ("after").
var disabledKinds = []struct {
	name  string
	build func(c *context, s *disabledState) *node
}{
	{"select", func(c *context, s *disabledState) *node {
		return coreSelect(c, &s.text, []string{"a", "b", "c"})
	}},
	{"dateinput", func(c *context, s *disabledState) *node {
		return coreDateInput(c, &s.date)
	}},
	{"colorwell", func(c *context, s *disabledState) *node {
		return coreColorWell(c, &s.color)
	}},
	{"popovermenubutton", func(c *context, s *disabledState) *node {
		return corePopoverMenuButton(c, "Menu", func(m *Menu) { m.Item("One"); m.Item("Two") })
	}},
	{"menubutton", func(c *context, s *disabledState) *node {
		return coreMenuButton(c, "Menu", func(m *Menu) { m.Item("One"); m.Item("Two") })
	}},
	{"tokenfield", func(c *context, s *disabledState) *node {
		s.suggests = true
		return coreTokenField(c, &s.tokens, []string{"alpha", "beta", "gamma"})
	}},
}

type disabledState struct {
	text   string
	date   time.Time
	color  Color
	tokens []string
	off    bool
	// suggests: the control shows suggestions in its popup.
	suggests bool
}

type disabledScene struct {
	tt     *Tester
	anchor *node
	s      *disabledState
}

func newDisabledScene(kind int, mode string, off bool) *disabledScene {
	d := &disabledScene{s: &disabledState{off: off, date: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)}}
	k := disabledKinds[kind]
	d.tt = coreNewTester(func(c *context) {
		switch mode {
		case "self":
			coreColumn(c).Padding(20).Children(func() { d.anchor = k.build(c, d.s).Disabled(d.s.off) })
		case "before":
			coreColumn(c).Padding(20).Disabled(d.s.off).Children(func() { d.anchor = k.build(c, d.s) })
		default:
			coreColumn(c).Padding(20).Children(func() { d.anchor = k.build(c, d.s) }).Disabled(d.s.off)
		}
	}, 400, 400)
	d.tt.Frame()
	d.tt.Frame()
	return d
}

// open reports whether a popup or a menu shows, whatever the anchor says.
func (d *disabledScene) open() bool {
	if len(d.tt.Menu()) > 0 {
		return true
	}
	for _, n := range d.tt.Nodes() {
		if n.Role == RolePopup || n.Role == RoleMenu {
			return true
		}
	}
	// The suggestions of a token field show in a popup of the input inside it.
	return d.s.suggests && d.tt.HasText("gamma")
}

func (d *disabledScene) click() {
	b := d.anchor.Bounds()
	d.tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	d.tt.Frame()
	d.tt.Frame()
}

var disabledModes = []string{"self", "before", "after"}

// A control disabled while its popup shows closes it.
func TestDisabledControlClosesItsOpenPopup(t *testing.T) {
	for i, k := range disabledKinds {
		if k.name == "menubutton" {
			continue // the system's menu is modal: the app cannot disable what the user is in
		}
		for _, mode := range disabledModes {
			t.Run(k.name+"/"+mode, func(t *testing.T) {
				d := newDisabledScene(i, mode, false)
				d.click()
				if k.name == "tokenfield" {
					// Its suggestions show as the user types.
					d.tt.Type("a")
					d.tt.Frame()
				}
				if !d.open() {
					t.Fatal("a click did not open it")
				}
				d.s.off = true
				d.tt.Frame()
				d.tt.Frame()
				if d.open() {
					t.Errorf("a disabled control shows its popup")
				}
			})
		}
	}
}

// A control that is disabled does not open on a click, or on keys.
func TestDisabledControlDoesNotOpen(t *testing.T) {
	keys := []struct {
		name string
		mods Modifiers
		key  Key
	}{{"Enter", 0, KeyEnter}, {"Space", 0, KeySpace}, {"Down", 0, KeyDown}, {"Alt+Down", Alt, KeyDown}}
	for i, k := range disabledKinds {
		for _, mode := range disabledModes {
			t.Run(k.name+"/"+mode+"/click", func(t *testing.T) {
				d := newDisabledScene(i, mode, true)
				d.click()
				if d.open() {
					t.Errorf("a click opened a disabled control")
				}
			})
			for _, key := range keys {
				t.Run(fmt.Sprint(k.name, "/", mode, "/", key.name), func(t *testing.T) {
					d := newDisabledScene(i, mode, true)
					d.tt.FocusNode(d.anchor.id)
					d.tt.Key(key.mods, key.key)
					d.tt.Frame()
					d.tt.Frame()
					if d.open() {
						t.Errorf("%s opened a disabled control", key.name)
					}
				})
			}
			t.Run(k.name+"/"+mode+"/access", func(t *testing.T) {
				d := newDisabledScene(i, mode, true)
				d.tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: d.anchor.id, Action: platform.AccessPress})
				d.tt.Frame()
				d.tt.Frame()
				if d.open() {
					t.Errorf("an accessibility press opened a disabled control")
				}
			})
		}
	}
}

// An app's own Popover anchored to a disabled element stays open, and its
// open state is the app's: "why is this disabled" is a legitimate popover.
func TestPopoverOfTheAppAnchoredToADisabledElementStaysOpen(t *testing.T) {
	for _, mode := range disabledModes {
		t.Run(mode, func(t *testing.T) {
			open, off := true, true
			tt := coreNewTester(func(c *context) {
				build := func() {
					b := coreButton(c, "Go")
					if mode == "self" {
						b.Disabled(off)
					}
					corePopover(c, b, &open, func() { coreText(c, "BECAUSE") })
				}
				switch mode {
				case "self":
					coreColumn(c).Padding(20).Children(build)
				case "before":
					coreColumn(c).Padding(20).Disabled(off).Children(build)
				default:
					coreColumn(c).Padding(20).Children(build).Disabled(off)
				}
			}, 400, 400)
			for range 3 {
				tt.Frame()
			}
			if !open || !tt.HasText("BECAUSE") {
				t.Errorf("the popover closed: open %v, shown %v", open, tt.HasText("BECAUSE"))
			}
		})
	}
}
