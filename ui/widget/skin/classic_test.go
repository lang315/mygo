package skin

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
	"github.com/egoist/mygo/ui/widget/tw"
)

// Classic passes Check with its own palettes.
func TestClassicCheck(t *testing.T) {
	if err := Classic.Check(); err != nil {
		t.Errorf("Classic.Check(): %v", err)
	}
	if err := Classic.WithPalette(ClassicSlate).Check(); err != nil {
		t.Errorf("Classic.WithPalette(ClassicSlate).Check(): %v", err)
	}
	if f := Classic.textFloor(); f != 4.5 {
		t.Errorf("Classic's text floor is %v, want 4.5", f)
	}
}

// Classic is Compact by default: after Use(Classic) a Button is its
// ControlHeight less 6, or its content where that is more, and a grid row
// its RowHeight less 4.
func TestClassicDensity(t *testing.T) {
	if Classic.Density() != Compact {
		t.Fatalf("Classic.Density() is %v, want Compact", Classic.Density())
	}
	t.Cleanup(func() { Use(nil) })
	tk := Classic.Tokens(false)
	l, d := Classic.Tokens(false), Classic.Tokens(true)
	l.ControlHeight, d.ControlHeight = 1, 1
	tiny := New("tiny", l, d).WithLooks(Classic.Looks())
	tt := widget.NewTester(t, sizeForm(), 600, 700)
	Use(tiny)
	tt.Frame()
	content := widgetSizes(t, tt)["Button"]
	if content <= 1 {
		t.Fatalf("a Button under ControlHeight 1 is %v high: no content measured", content)
	}
	Use(Classic)
	tt.Frame()
	got := widgetSizes(t, tt)
	if want := max(tk.ControlHeight-6, content); !same(got["Button"], want) {
		t.Errorf("a Button is %v, want %v (ControlHeight %v less 6, content %v)", got["Button"], want, tk.ControlHeight, content)
	}
	if want := tk.RowHeight - 4; !same(got["DataGridView row"], want) {
		t.Errorf("a grid row is %v, want %v", got["DataGridView row"], want)
	}
}

// Classic's palettes are its own: Indigo, its default, and Slate.
func TestClassicPalettes(t *testing.T) {
	var names []string
	for _, p := range Classic.Palettes() {
		names = append(names, p.Name())
	}
	if !slices.Equal(names, []string{"Indigo", "Slate"}) {
		t.Errorf("Classic.Palettes() are %q, want Indigo and Slate", names)
	}
	if Classic.Palette() != ClassicIndigo {
		t.Errorf("Classic.Palette() is %v, want ClassicIndigo", Classic.Palette().Name())
	}
	if Classic.Palettes()[0] != ClassicIndigo || Classic.Palettes()[1] != ClassicSlate {
		t.Errorf("Classic.Palettes() are not ClassicIndigo and ClassicSlate")
	}
	if Classic.Tokens(false).Accent.Base == Calm.WithPalette(Indigo).Tokens(false).Accent.Base {
		t.Errorf("ClassicIndigo is Calm's Indigo")
	}
}

// Classic's shape, light and dark: the open tab joins its page, with no
// border-colored pixel along its bottom edge; a closed tab is filled with
// Chrome; a grid header cell is filled with Chrome and ends in a
// BorderStrong line; a Button is outlined with corners of 2.
func TestClassicShape(t *testing.T) {
	type row struct{ ID, Name, Note string }
	for _, dark := range []bool{false, true} {
		t.Run(appearance(dark), func(t *testing.T) {
			tk := Classic.Tokens(dark)
			tabs := widget.TabControl().AddPage("One", widget.Label("one")).AddPage("Two", widget.Label("two")).SetSize(0, 120)
			grid := widget.DataGridView(func(r *row) string { return r.ID },
				widget.Column[row]{Title: "Name", Text: func(r *row) string { return r.Name }},
				widget.Column[row]{Title: "Note", Text: func(r *row) string { return r.Note }},
			).SetRows([]row{{"1", "Ada", "a"}}).SetSize(0, 100)
			tt := showSkin(t, widget.VBox(8).Add(tabs, grid, widget.Button("OK")), 600, 400, Classic, dark, 1)
			img := tt.Image()
			near := func(c, want ui.Color) bool { return dist(c, want) <= 3 }

			one, two := nodeOf(t, tt, ui.RoleTab, "One"), nodeOf(t, tt, ui.RoleTab, "Two")
			y := int(one.Y+one.H) - 1
			for x := int(one.X) + 2; x < int(one.X+one.W)-2; x++ {
				if c := colorAt(img, x, y); near(c, tk.Border) || near(c, tk.BorderStrong) {
					t.Errorf("the open tab's bottom edge has a border pixel %v at x=%d: it does not join its page", c, x)
					break
				}
			}
			// Its fill runs into the page with no line: a column in its padding
			// left of the text, from mid-tab to 6 rows into the page, is
			// Background all the way.
			for y := int(one.Y + one.H/2); y < int(one.Y+one.H)+6; y++ {
				if c := colorAt(img, int(one.X)+4, y); !near(c, tk.Background) {
					t.Errorf("at y=%d (the tab ends at %v) the column under the open tab is %v, want Background %v", y, one.Y+one.H, c, tk.Background)
					break
				}
			}
			if c := colorAt(img, int(two.X+two.W/2), int(two.Y+two.H)-1); !near(c, tk.BorderStrong) {
				t.Errorf("the closed tab's bottom edge is %v, want BorderStrong %v", c, tk.BorderStrong)
			}
			if c := colorAt(img, int(two.X)+5, int(two.Y+two.H/2)); !near(c, tk.Chrome) {
				t.Errorf("the closed tab is %v, want Chrome %v", c, tk.Chrome)
			}

			head := nodeOf(t, tt, ui.RoleColumnHeader, "Name")
			if c := colorAt(img, int(head.X)+4, int(head.Y+head.H/2)); !near(c, tk.Chrome) {
				t.Errorf("the grid header cell is %v, want Chrome %v", c, tk.Chrome)
			}
			if c := colorAt(img, int(head.X+head.W)-1, int(head.Y+head.H/2)); !near(c, tk.BorderStrong) {
				t.Errorf("the grid header cell's right edge is %v, want BorderStrong %v", c, tk.BorderStrong)
			}

			b := tt.Find(widget.ByLabel("OK")).Bounds()
			by := int(b.Y)
			if c := colorAt(img, int(b.X+b.W/2), by); !near(c, tk.ButtonBorder) {
				t.Errorf("the Button's top edge is %v, want ButtonBorder %v", c, tk.ButtonBorder)
			}
			// Corners of 2: the top row is the border from x = 2 on (Calm's 6
			// leaves x = 3 the ground), and the corner pixel itself is rounded off.
			if c := colorAt(img, int(b.X)+3, by); !near(c, tk.ButtonBorder) {
				t.Errorf("the Button's top row at x+3 is %v, want ButtonBorder %v: its corner is over 2", c, tk.ButtonBorder)
			}
			if c := colorAt(img, int(b.X), by); near(c, tk.ButtonBorder) {
				t.Errorf("the Button's corner pixel is the border %v: it is square", c)
			}
		})
	}
}

// Every class of Classic's looks is one tw knows.
func TestClassicLooksParse(t *testing.T) {
	l := Classic.Looks()
	for name, cl := range map[string]string{"Button": l.Button, "ToolButton": l.ToolButton, "GroupBox": l.GroupBox,
		"TabStrip": l.TabStrip, "TabHeader": l.TabHeader, "GridHeader": l.GridHeader} {
		if cl == "" {
			t.Errorf("%s has no look", name)
		}
		if _, err := tw.Parse(cl); err != nil {
			t.Errorf("%s %q: %v", name, cl, err)
		}
	}
	if l.CheckBox != "" || l.RadioButton != "" {
		t.Errorf("CheckBox %q and RadioButton %q: a look of theirs boxes their text", l.CheckBox, l.RadioButton)
	}
}

// Classic's values: radii 2 and 3, check boxes 14, no card shadow, and its
// light grays the plan's.
func TestClassicValues(t *testing.T) {
	for _, dark := range []bool{false, true} {
		tk := Classic.Tokens(dark)
		if tk.RadiusControl != 2 || tk.RadiusCard != 3 || tk.CheckSize != 14 {
			t.Errorf("dark=%v: RadiusControl %v, RadiusCard %v, CheckSize %v; want 2, 3, 14", dark, tk.RadiusControl, tk.RadiusCard, tk.CheckSize)
		}
		if tk.CardShadow != (Shadow{}) || tk.PopupShadow == (Shadow{}) {
			t.Errorf("dark=%v: CardShadow %+v, PopupShadow %+v; want none and one", dark, tk.CardShadow, tk.PopupShadow)
		}
	}
	l := Classic.Tokens(false)
	for _, c := range []struct {
		name string
		got  ui.Color
		want string
	}{{"Chrome", l.Chrome, "#eef0f3"}, {"Background", l.Background, "#ffffff"}, {"Border", l.Border, "#d0d4da"},
		{"BorderStrong", l.BorderStrong, "#aab0b9"}, {"ControlBorder", l.ControlBorder, "#7d8590"}} {
		if c.got != ui.Hex(c.want) {
			t.Errorf("light %s is %v, want %s", c.name, hexOf(c.got), c.want)
		}
	}
	if Classic.Name() != "Classic" {
		t.Errorf("Classic.Name() is %q", Classic.Name())
	}
}
