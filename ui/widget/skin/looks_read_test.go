package skin

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// Tailwind's 600s (and two others), as tw gives them: the sentinel looks'
// colors, none of them a color Calm paints.
var (
	red600     = ui.Hex("#dc2626")
	orange600  = ui.Hex("#ea580c")
	amber600   = ui.Hex("#d97706")
	yellow600  = ui.Hex("#ca8a04")
	lime600    = ui.Hex("#65a30d")
	green600   = ui.Hex("#16a34a")
	emerald600 = ui.Hex("#059669")
	teal600    = ui.Hex("#0d9488")
	cyan600    = ui.Hex("#0891b2")
	sky600     = ui.Hex("#0284c7")
	pink600    = ui.Hex("#db2777")
	rose600    = ui.Hex("#e11d48")
	indigo600  = ui.Hex("#4f46e5")
	violet600  = ui.Hex("#7c3aed")
	purple600  = ui.Hex("#9333ea")
	fuchsia600 = ui.Hex("#c026d3")
	red800     = ui.Hex("#991b1b")
)

// sentinelLooks is a look for every kind, each its own color; the open tab
// is pink.
var sentinelLooks = Looks{
	Button: "bg-red-600", ToolButton: "bg-orange-600", Field: "bg-amber-600", ComboBox: "bg-yellow-600",
	CheckBox: "bg-lime-600", RadioButton: "bg-green-600", GroupBox: "bg-emerald-600", Card: "bg-teal-600",
	TabStrip: "bg-cyan-600", TabHeader: "bg-sky-600 selected:bg-pink-600", GridHeader: "bg-rose-600",
	ToolStrip: "bg-indigo-600", StatusStrip: "bg-violet-600", MenuBar: "bg-purple-600", HeaderBar: "bg-fuchsia-600",
}

func looksSentinel() *Skin { return Calm.WithLooks(sentinelLooks) }

// countColor counts the pixels of r that are c exactly.
func countColor(tt *widget.Tester, r ui.Rect, c ui.Color) int {
	img := tt.Image()
	n := 0
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			if colorAt(img, x, y) == c {
				n++
			}
		}
	}
	return n
}

type lookRow struct{ ID, Name string }

func lookGrid(n int, builds *atomic.Int32) *widget.DataGridViewWidget[lookRow, string] {
	var rows []lookRow
	for i := range n {
		rows = append(rows, lookRow{fmt.Sprint(i), fmt.Sprint("Row ", i)})
	}
	return widget.DataGridView(func(r *lookRow) string { return r.ID },
		widget.Column[lookRow]{Title: "Name", Text: func(r *lookRow) string {
			if builds != nil {
				builds.Add(1)
			}
			return r.Name
		}},
	).SetRows(rows).SetSize(0, 360)
}

// leastLook is the pixels of its color a look must paint: more than a
// border's or a fringe's.
const leastLook = 60

// nodeRect returns the bounds of the first node of role r that pick
// accepts, failing the test without one.
func nodeRect(t *testing.T, tt *widget.Tester, r ui.Role, pick func(ui.Rect) bool) ui.Rect {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Role == r && (pick == nil || pick(n.Bounds)) {
			return n.Bounds
		}
	}
	t.Fatalf("no %v node", r)
	return ui.Rect{}
}

// roleBox is the box typed into of a control whose node of role r it is.
func roleBox(r ui.Role) func(*testing.T, *widget.Tester) []ui.Rect {
	return func(t *testing.T, tt *widget.Tester) []ui.Rect { return []ui.Rect{nodeRect(t, tt, r, nil)} }
}

// domainBox is a DomainUpDown's field: the control left of its stepper.
func domainBox(t *testing.T, tt *widget.Tester) []ui.Rect {
	s := nodeRect(t, tt, ui.RoleStepper, nil)
	return []ui.Rect{{X: 0, Y: s.Y, W: s.X, H: s.H}}
}

// dateTimeBoxes are a DateTimePicker's two fields: the date's button and
// the time's group after it.
func dateTimeBoxes(t *testing.T, tt *widget.Tester) []ui.Rect {
	date := nodeRect(t, tt, ui.RolePopUpButton, nil)
	tm := nodeRect(t, tt, ui.RoleGroup, func(r ui.Rect) bool { return r.X > date.X+date.W && r.H >= date.H-1 })
	return []ui.Rect{date, tm}
}

// Each kind's control paints its look's color, light and dark: the seven
// inputs typed into the Field's, the open tab the TabHeader's selected:
// color and the others its own, the column header the GridHeader's.
func TestLooksReachEveryKind(t *testing.T) {
	whole := ui.Rect{W: 600, H: 400}
	cases := []struct {
		name  string
		make  func() widget.Control
		color ui.Color
		// boxes, when set, are where the look must cover half the area.
		boxes func(*testing.T, *widget.Tester) []ui.Rect
	}{
		{"Button", func() widget.Control { return widget.Button("Go") }, red600, roleBox(ui.RoleButton)},
		{"ToolButton", func() widget.Control { return widget.ToolStrip(widget.ToolButton("Tool")) }, orange600, roleBox(ui.RoleButton)},
		{"TextBox", func() widget.Control { return widget.TextBox() }, amber600, roleBox(ui.RoleTextField)},
		{"MultilineTextBox", func() widget.Control { return widget.MultilineTextBox().SetSize(200, 80) }, amber600, roleBox(ui.RoleTextField)},
		{"MaskedTextBox", func() widget.Control { return widget.MaskedTextBox("000-0000") }, amber600, roleBox(ui.RoleTextField)},
		{"NumericUpDown", func() widget.Control { return widget.NumericUpDown(0, 10, 1) }, amber600, roleBox(ui.RoleTextField)},
		{"DomainUpDown", func() widget.Control { return widget.DomainUpDown("One", "Two") }, amber600, domainBox},
		{"DateTimePicker", func() widget.Control { return widget.DateTimePicker() }, amber600, dateTimeBoxes},
		{"EditableComboBox", func() widget.Control {
			return widget.ComboBox(widget.Option{Value: "a", Text: "Alpha"}).Editable()
		}, amber600, roleBox(ui.RoleComboBox)},
		{"ComboBox", func() widget.Control {
			return widget.ComboBox(widget.Option{Value: "a", Text: "Alpha"}).SetValue("a")
		}, yellow600, roleBox(ui.RolePopUpButton)},
		{"CheckBox", func() widget.Control { return widget.CheckBox("Check") }, lime600, nil},
		{"RadioButton", func() widget.Control { return widget.RadioButton("Radio") }, green600, nil},
		{"GroupBox", func() widget.Control { return widget.GroupBox("Group").Add(widget.Label("inside")) }, emerald600, nil},
		{"Card", func() widget.Control { return widget.Card().SetTitle(widget.Label("Card")) }, teal600, nil},
		{"TabStrip", func() widget.Control {
			return widget.TabControl().AddPage("One", widget.Label("one")).AddPage("Two", widget.Label("two"))
		}, cyan600, nil},
		{"GridHeader", func() widget.Control { return lookGrid(3, nil) }, rose600, roleBox(ui.RoleColumnHeader)},
		{"ListViewDetails", func() widget.Control {
			return widget.ListView(func(r *lookRow) string { return r.ID },
				widget.Column[lookRow]{Title: "Name", Text: func(r *lookRow) string { return r.Name }}).
				SetView(widget.ViewDetails).SetRows([]lookRow{{"1", "Ada"}}).SetSize(0, 200)
		}, rose600, roleBox(ui.RoleColumnHeader)},
		{"ToolStrip", func() widget.Control { return widget.ToolStrip(widget.ToolButton("Tool")) }, indigo600, nil},
		{"StatusStrip", func() widget.Control { return widget.StatusStrip(widget.StatusLabel("Ready")) }, violet600, nil},
		{"MenuBar", func() widget.Control {
			return widget.MenuBar(widget.MenuItem{Text: "&File", Children: []widget.MenuItem{{Text: "Open"}}})
		}, purple600, nil},
	}
	for _, c := range cases {
		for _, dark := range []bool{false, true} {
			t.Run(c.name+"-"+appearance(dark), func(t *testing.T) {
				tt := showSkin(t, widget.VBox(8).Add(c.make()), 600, 400, looksSentinel(), dark, 1)
				if n := countColor(tt, whole, c.color); n < leastLook {
					t.Errorf("%d pixels of the look's %v, want at least %d", n, c.color, leastLook)
				}
				if c.boxes != nil {
					for _, r := range c.boxes(t, tt) {
						if n, want := countColor(tt, r, c.color), int(r.W*r.H)/2; n < want {
							t.Errorf("the box %v has %d pixels of the look's %v, want at least half, %d", r, n, c.color, want)
						}
					}
				}
				Use(Calm)
				tt.Frame()
				if n := countColor(tt, whole, c.color); n != 0 {
					t.Errorf("under Calm, %d pixels of the look's %v: the test sees no change", n, c.color)
				}
			})
		}
	}
	for _, dark := range []bool{false, true} {
		t.Run("HeaderBar-"+appearance(dark), func(t *testing.T) {
			t.Cleanup(func() { Use(nil) })
			f := widget.NewForm("Shell", 600, 400).SetContent(widget.Label("content"))
			f.UseHeaderBar("")
			tt := widget.TestForm(t, f)
			Use(looksSentinel())
			tt.SetDark(dark)
			tt.Frame()
			if n := countColor(tt, whole, fuchsia600); n < leastLook {
				t.Errorf("%d pixels of the HeaderBar look's color, want at least %d", n, leastLook)
			}
		})
		t.Run("TabHeader-"+appearance(dark), func(t *testing.T) {
			tc := widget.TabControl().AddPage("One", widget.Label("one")).AddPage("Two", widget.Label("two"))
			tt := showSkin(t, tc, 600, 400, looksSentinel(), dark, 1)
			one, two := nodeOf(t, tt, ui.RoleTab, "One"), nodeOf(t, tt, ui.RoleTab, "Two")
			if n := countColor(tt, one, pink600); n < leastLook {
				t.Errorf("the open tab has %d pixels of the selected: color, want at least %d", n, leastLook)
			}
			if n := countColor(tt, two, sky600); n < leastLook {
				t.Errorf("the other tab has %d pixels of the look's color, want at least %d", n, leastLook)
			}
			if n := countColor(tt, two, pink600); n != 0 {
				t.Errorf("the tab not open has %d pixels of the selected: color, want 0", n)
			}
			if n := countColor(tt, one, sky600); n != 0 {
				t.Errorf("the open tab has %d pixels of the base color, want 0", n)
			}
		})
		t.Run("GridHeaderCell-"+appearance(dark), func(t *testing.T) {
			tt := showSkin(t, lookGrid(3, nil), 600, 400, looksSentinel(), dark, 1)
			head := nodeOf(t, tt, ui.RoleColumnHeader, "Name")
			if n, want := countColor(tt, head, rose600), int(head.W*head.H)/2; n < want {
				t.Errorf("the column header cell %v has %d pixels of the look's color, want at least half, %d", head, n, want)
			}
		})
	}
}

// The control's own methods win over its Apply'd styles, which win over its
// look; StripStyle wins over the TabStrip look, and the ToolStrip look over
// the skin's own Chrome.
func TestLooksPrecedence(t *testing.T) {
	s := Calm.WithLooks(Looks{Button: "bg-red-600", TabStrip: "bg-cyan-600", ToolStrip: "bg-indigo-600"})
	for _, dark := range []bool{false, true} {
		t.Run(appearance(dark), func(t *testing.T) {
			accent := Calm.Tokens(dark).Accent.Base
			applied := widget.Button("Applied").Apply(widget.NewStyle().Background(widget.RoleAccent))
			own := widget.Button("Own").Apply(widget.NewStyle().Background(widget.RoleAccent)).Background(ui.Hex("#123456"))
			strip := widget.TabControl().AddPage("One", widget.Label("one")).
				StripStyle(widget.NewStyle().Background(widget.Hex("#345678")))
			tools := widget.ToolStrip(widget.ToolButton("Tool"))
			tt := showSkin(t, widget.VBox(8).Add(applied, own, strip, tools), 600, 400, s, dark, 1)
			at := func(r ui.Rect) ui.Color { return colorAt(tt.Image(), int(r.X)+3, int(r.Y+r.H/2)) }
			if got := at(tt.Find(widget.ByLabel("Applied")).Bounds()); got != accent {
				t.Errorf("an Apply'd accent under a red look paints %v, want the accent %v", got, accent)
			}
			if got := at(tt.Find(widget.ByLabel("Own")).Bounds()); got != ui.Hex("#123456") {
				t.Errorf("the button's own Background paints %v, want #123456", got)
			}
			tab := nodeOf(t, tt, ui.RoleTab, "One")
			if got := colorAt(tt.Image(), 500, int(tab.Y+tab.H/2)); got != ui.Hex("#345678") {
				t.Errorf("the strip past its tab paints %v, want StripStyle's #345678", got)
			}
			bar := nodeOf(t, tt, ui.RoleToolbar, "")
			if got := colorAt(tt.Image(), int(bar.X+bar.W)-4, int(bar.Y+bar.H/2)); got != indigo600 {
				t.Errorf("the tool strip paints %v, want the look's %v over the skin's Chrome %v", got, indigo600, Calm.Tokens(dark).Chrome)
			}
		})
	}
}

// A look's selected: applies while the control is checked, and only then.
func TestLookSelected(t *testing.T) {
	s := Calm.WithLooks(Looks{CheckBox: "selected:bg-green-600"})
	for _, dark := range []bool{false, true} {
		t.Run(appearance(dark), func(t *testing.T) {
			box := widget.CheckBox("Check")
			tt := showSkin(t, widget.VBox(8).Add(box), 400, 200, s, dark, 1)
			whole := ui.Rect{W: 400, H: 200}
			if n := countColor(tt, whole, green600); n != 0 {
				t.Errorf("unchecked, %d pixels of the selected: green, want 0", n)
			}
			box.SetChecked(true)
			tt.Frame()
			if n := countColor(tt, whole, green600); n < leastLook {
				t.Errorf("checked, %d pixels of the selected: green, want at least %d", n, leastLook)
			}
			box.SetChecked(false)
			tt.Frame()
			if n := countColor(tt, whole, green600); n != 0 {
				t.Errorf("unchecked again, %d pixels of the selected: green, want 0", n)
			}
		})
	}
}

// A DataGridView whose rows are kept from frame to frame paints the new
// skin's Chrome in its header in the frame after Use, through a GridHeader
// look of bg-chrome.
func TestGridHeaderLookFollowsUse(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	skinWith := func(name string, chrome ui.Color) *Skin {
		l, d := Calm.Tokens(false), Calm.Tokens(true)
		l.Chrome, d.Chrome = chrome, chrome
		return New(name, l, d).WithLooks(Looks{GridHeader: "bg-chrome"})
	}
	a, b := skinWith("A", ui.Hex("#a10001")), skinWith("B", ui.Hex("#0001b1"))
	var builds atomic.Int32
	grid := lookGrid(50, &builds)
	note := widget.Label("one")
	tt := widget.NewTester(t, widget.VBox(8).Add(note, grid), 600, 400)
	Use(a)
	tt.Settle()
	builds.Store(0)
	note.SetText("two")
	tt.Frame()
	if n := builds.Load(); n != 0 {
		t.Fatalf("a frame that changed only a label built %d cells: the grid keeps no rows", n)
	}
	head := nodeOf(t, tt, ui.RoleColumnHeader, "Name")
	if n, want := countColor(tt, head, ui.Hex("#a10001")), int(head.W*head.H)/2; n < want {
		t.Fatalf("under A the header has %d pixels of A's Chrome, want at least %d", n, want)
	}
	Use(b)
	tt.Frame()
	head = nodeOf(t, tt, ui.RoleColumnHeader, "Name")
	if n, want := countColor(tt, head, ui.Hex("#0001b1")), int(head.W*head.H)/2; n < want {
		t.Errorf("one frame after Use(B) the header has %d pixels of B's Chrome, want at least %d", n, want)
	}
	if n := countColor(tt, head, ui.Hex("#a10001")); n != 0 {
		t.Errorf("one frame after Use(B) the header still has %d pixels of A's Chrome", n)
	}
}

// With a TabHeader look, the tab with the keyboard focus keeps it when the
// page before it goes, as without.
func TestTabHeaderLookKeepsFocus(t *testing.T) {
	for _, s := range []*Skin{looksSentinel(), Calm} {
		t.Run(fmt.Sprintf("looks-%v", s.Looks() != Looks{}), func(t *testing.T) {
			tc := widget.TabControl().AddPage("Alpha", widget.Label("alpha")).
				AddPage("Beta", widget.Label("beta")).AddPage("Gamma", widget.Label("gamma"))
			tt := showSkin(t, tc, 600, 300, s, false, 1)
			if err := tt.Click("Gamma"); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			tc.RemovePage(0)
			tt.Frame()
			tt.Frame()
			var focused string
			for _, n := range tt.Nodes() {
				if n.Role == ui.RoleTab && n.Focused {
					focused = n.Label
				}
			}
			if focused != "Gamma" {
				t.Errorf("focus on %q after the first page went, want Gamma", focused)
			}
			if tc.Page() != 1 {
				t.Errorf("page %d shown, want 1, Gamma's", tc.Page())
			}
		})
	}
}

// With a TabHeader look, the tab the keyboard focuses shows the focus ring,
// in the theme's Focus, as without.
func TestTabHeaderLookRing(t *testing.T) {
	tc := widget.TabControl().AddPage("One", widget.Label("one")).AddPage("Two", widget.Label("two"))
	tt := showSkin(t, tc, 600, 300, looksSentinel(), false, 1)
	focus := Calm.Tokens(false).Accent.Base
	tab := nodeOf(t, tt, ui.RoleTab, "One")
	around := ui.Rect{X: tab.X - 6, Y: tab.Y - 6, W: tab.W + 12, H: tab.H + 12}
	if n := countColor(tt, around, focus); n != 0 {
		t.Fatalf("%d pixels of the focus ring's %v before the focus", n, focus)
	}
	tt.Key(0, ui.KeyTab)
	tt.Settle()
	if n := countColor(tt, around, focus); n < 40 {
		t.Errorf("%d pixels of the focus ring's %v around the focused tab, want at least 40", n, focus)
	}
}

// Under Calm, which has no looks, the button, the grid's header and the tab
// strip paint what they painted before looks: Calm's Surface, Background and
// Chrome.
func TestNoLooksNoChange(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(appearance(dark), func(t *testing.T) {
			face, ground, chrome := ui.Hex("#ffffff"), ui.Hex("#ffffff"), ui.Hex("#f4f5f7")
			if dark {
				face, ground, chrome = ui.Hex("#262a30"), ui.Hex("#15171a"), ui.Hex("#1d1f23")
			}
			tc := widget.TabControl().AddPage("One", widget.Label("one"))
			tt := showSkin(t, widget.VBox(8).Add(widget.Button("Go"), lookGrid(3, nil), tc), 600, 500, Calm, dark, 1)
			b := tt.Find(widget.ByLabel("Go")).Bounds()
			if got := colorAt(tt.Image(), int(b.X)+4, int(b.Y+b.H/2)); got != face {
				t.Errorf("the button paints %v, want %v", got, face)
			}
			head := nodeOf(t, tt, ui.RoleColumnHeader, "Name")
			if got := colorAt(tt.Image(), int(head.X+head.W)-4, int(head.Y+head.H/2)); got != ground {
				t.Errorf("the column header paints %v, want %v", got, ground)
			}
			tab := nodeOf(t, tt, ui.RoleTab, "One")
			if got := colorAt(tt.Image(), 500, int(tab.Y+tab.H/2)); got != chrome {
				t.Errorf("the tab strip paints %v, want %v", got, chrome)
			}
		})
	}
}

// Looks with dark: resolve in the dark appearance, and their sizes are DIPs
// or theme units as tw gives them, at text scale 1.5 too.
func TestLooksDarkAndScale(t *testing.T) {
	s := Calm.WithLooks(Looks{Button: "bg-red-600 dark:bg-red-800 h-12"})
	for _, dark := range []bool{false, true} {
		for _, scale := range []float32{1, 1.5} {
			t.Run(fmt.Sprintf("%s-%g", appearance(dark), scale), func(t *testing.T) {
				tt := showSkin(t, widget.VBox(8).Add(widget.Button("Go")), 400, 200, s, dark, scale)
				b := tt.Find(widget.ByLabel("Go")).Bounds()
				want := red600
				if dark {
					want = red800
				}
				if got := colorAt(tt.Image(), int(b.X)+4, int(b.Y+b.H/2)); got != want {
					t.Errorf("the button paints %v, want %v", got, want)
				}
				if b.H < 47.5 || b.H > 48.5 {
					t.Errorf("the button is %v high, want h-12's 48", b.H)
				}
			})
		}
	}
}

// A frame of labels, buttons and text boxes allocates nothing with every
// look on, as without a skin: plain, and with a style of the app's over the
// looks.
func TestLooksAllocs(t *testing.T) {
	t.Cleanup(func() { Use(nil) })
	look := widget.NewStyle().Padding(1).Radius(1).Background(widget.RoleSurface).
		Hover(widget.NewStyle().Background(widget.RoleSurfaceHover))
	for _, styled := range []bool{false, true} {
		t.Run(fmt.Sprintf("styled-%v", styled), func(t *testing.T) {
			box := widget.VBox(2)
			for i := range 100 {
				var w widget.Control
				switch i % 3 {
				case 0:
					w = widget.Label(fmt.Sprintf("Label %d", i))
				case 1:
					b := widget.Button(fmt.Sprintf("Button %d", i))
					if styled {
						b.Apply(look)
					}
					w = b
				default:
					e := widget.TextBox().SetText(fmt.Sprintf("Text %d", i))
					if styled {
						e.Apply(look)
					}
					w = e
				}
				box.Add(w)
			}
			tt := widget.NewTester(t, widget.ScrollPanel(box), 1280, 800)
			Use(looksSentinel())
			for range 10 {
				tt.Frame()
			}
			if n := allocrun.Least(30, tt.Frame); n != 0 {
				t.Errorf("%v allocations in a frame with every look on, want 0", n)
			}
		})
	}
}
