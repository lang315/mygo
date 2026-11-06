package widget

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
)

// wbar is a control w by h DIPs, red from edge to edge, with text at its top
// left: a fixed size, whatever the font.
func wbar(text string, w, h float32) *CustomWidget {
	return Custom(func(c *ui.Context) ui.Element {
		return ui.Column(c).Width(w).Height(h).Background(ui.RGB(255, 0, 0)).Children(func() { ui.Text(c, text) })
	})
}

// rules returns the y of each divider drawn across [1, x1): a line of one
// color, not the page's and not red, as wide as that, 1 high.
func rules(tt *ui.Tester, x1 int) (ys []int) {
	img, bg := tt.Image(), page(tt)
	b := img.Bounds()
	prev := false
	for y := b.Min.Y; y < b.Max.Y; y++ {
		first := img.RGBAAt(1, y)
		line := first != bg && !(first.R > 200 && first.G < 60)
		for x := 2; line && x < x1; x++ {
			line = img.RGBAAt(x, y) == first
		}
		if line && !prev {
			ys = append(ys, y)
		}
		prev = line
	}
	return
}

func settingsForm() (*FieldsWidget, [3]*CustomWidget) {
	cs := [3]*CustomWidget{wbar("c1", 60, 24), wbar("c2", 120, 24), wbar("c3", 90, 24)}
	return Fields().SettingsRows().Row("Name", cs[0]).Row("Email address", cs[1]).Row("Age", cs[2]), cs
}

// Controls of different widths share their right edge, at the form's.
func TestSettingsRowsControlsShareARightEdge(t *testing.T) {
	f, _ := settingsForm()
	tt := NewTester(t, f, 400, 300)
	edge := find(t, tt.Tester, "c1").X + 60
	for text, w := range map[string]float32{"c2": 120, "c3": 90} {
		if got := find(t, tt.Tester, text).X + w; !approx(got, edge) {
			t.Errorf("%s ends at %v, c1 at %v", text, got, edge)
		}
	}
	if edge < 390 || edge > 401 {
		t.Errorf("right edge at %v in a 400 wide window", edge)
	}
	// The labels stay at the left, in one column.
	if n, e := find(t, tt.Tester, "Name"), find(t, tt.Tester, "Email address"); n.X != e.X || n.X > 2 {
		t.Errorf("labels at x %v and %v", n.X, e.X)
	}
}

// A control that asks for its own place keeps it.
func TestSettingsRowsControlAlignWins(t *testing.T) {
	left := wbar("c1", 60, 24).Align(Start)
	f := Fields().SettingsRows().Row("Name", left).Row("Age", wbar("c2", 120, 24))
	tt := NewTester(t, f, 400, 300)
	// The column is as wide as the wider control, c2: c1 starts where it does.
	if a, b := find(t, tt.Tester, "c1").X, find(t, tt.Tester, "c2").X; !approx(a, b) || b > 290 {
		t.Errorf("c1 at %v, c2 at %v: want both at the column's left, 280", a, b)
	}
}

// A long label takes the width the control leaves and wraps.
func TestSettingsRowsLongLabelWraps(t *testing.T) {
	one := find(t, tester(t, Label("Name")), "Name").H
	f := Fields().SettingsRows().Row(longLine, wbar("c1", 60, 24)).Row("Name", wbar("c2", 60, 24))
	tt := NewTester(t, f, 300, 400)
	l := find(t, tt.Tester, longLine)
	if l.H < 2*one {
		t.Errorf("label %v is one line (%v high)", l, one)
	}
}

// The label stops before the control, whatever its longest word: the label
// column is no narrower than its content.
func TestSettingsRowsLabelEndsBeforeTheControl(t *testing.T) {
	tt := NewTester(t, Fields().SettingsRows().Row("Name shown to others", TextBox()), 200, 300)
	l := find(t, tt.Tester, "Name shown to others")
	if c := tt.Find(ByRole(ui.RoleTextField)).Bounds(); l.X+l.W > c.X+0.5 {
		t.Errorf("the label %v runs under the control %v", l, c)
	}
}

// A real control next to a wrapped label keeps the height it has alone and
// sits at the label's middle.
func TestSettingsRowsControlKeepsItsHeightNextToAWrappedLabel(t *testing.T) {
	alone := NewTester(t, Fields().SettingsRows().Row("Name", TextBox()), 400, 300).Find(ByRole(ui.RoleTextField)).Bounds()
	tt := NewTester(t, Fields().SettingsRows().Row(longLine, TextBox()), 400, 300)
	l, c := find(t, tt.Tester, longLine), tt.Find(ByRole(ui.RoleTextField)).Bounds()
	if l.H <= alone.H {
		t.Fatalf("label %v is not taller than the control alone %v: make the label longer", l, alone)
	}
	if !approx(c.H, alone.H) {
		t.Errorf("control %v next to the wrapped label, %v alone", c, alone)
	}
	if lm, cm := l.Y+l.H/2, c.Y+c.H/2; !approx(lm, cm) {
		t.Errorf("label's middle at %v, control's at %v", lm, cm)
	}
}

// A hidden control takes its label and its divider away, in the first, the
// middle and the last row; the dividers are between the rows that show, never
// doubled nor trailing; shown again, all come back.
func TestSettingsRowsDividersFollowTheVisibleRows(t *testing.T) {
	for hide := range 3 {
		t.Run(fmt.Sprint("hide ", hide), func(t *testing.T) {
			f, cs := settingsForm()
			tt := NewTester(t, f, 400, 300)
			left := int(find(t, tt.Tester, "c2").X) - 4
			if got := len(rules(tt.Tester, left)); got != 2 {
				t.Fatalf("%d dividers with 3 rows, want 2", got)
			}
			cs[hide].SetVisible(false)
			tt.Frame()
			names := []string{"Name", "Email address", "Age"}
			if tt.HasText(names[hide]) {
				t.Errorf("the label of the hidden control shows: %q", tt.Texts())
			}
			ys := rules(tt.Tester, left)
			if len(ys) != 1 {
				t.Fatalf("%d dividers with 2 rows, want 1 (at %v)", len(ys), ys)
			}
			// Between the two rows left.
			var shown []string
			for i, n := range names {
				if i != hide {
					shown = append(shown, n)
				}
			}
			a, b := find(t, tt.Tester, shown[0]), find(t, tt.Tester, shown[1])
			if y := float32(ys[0]); y < a.Y+a.H || y > b.Y {
				t.Errorf("divider at %v is not between %v and %v", y, a, b)
			}
			cs[hide].SetVisible(true)
			tt.Frame()
			if got := len(rules(tt.Tester, left)); got != 2 || !tt.HasText(names[hide]) {
				t.Errorf("shown again: %d dividers, texts %q", got, tt.Texts())
			}
		})
	}
	// Only one row left: no divider at all.
	f, cs := settingsForm()
	tt := NewTester(t, f, 400, 300)
	cs[0].SetVisible(false)
	cs[2].SetVisible(false)
	tt.Frame()
	if ys := rules(tt.Tester, int(find(t, tt.Tester, "c2").X)-4); len(ys) != 0 {
		t.Errorf("a lone row has dividers at %v", ys)
	}
}

// A label still names its control, and its mnemonic focuses it.
func TestSettingsRowsNameTheirControlsAndTakeMnemonics(t *testing.T) {
	withMac(t, false)
	f := NewForm("m", 400, 200)
	f.SetContent(Fields().SettingsRows().Row("&Name", TextBox()).Row("E&mail", TextBox()))
	tt := TestForm(t, f)
	tt.Settle()
	for _, n := range []string{"Name", "Email"} {
		if !tt.Has(ByRole(ui.RoleTextField).Label(n)) {
			t.Errorf("no text field named %q among %q", n, fieldNodes(tt.Tester))
		}
	}
	tt.Key(ui.Alt, ui.KeyM)
	tt.Settle()
	if !tt.Find(ByRole(ui.RoleTextField).Nth(1)).Focused() {
		t.Error("Alt+M did not focus the second text box")
	}
}

// A dialog of height 0 counts the wrapped label: it is higher than with a
// short label, by a line at least.
func TestSettingsRowsFitCountsWrappedLabels(t *testing.T) {
	mk := func(label string) Control {
		return Fields().SettingsRows().Row(label, wbar("c1", 60, 24))
	}
	short, long := fitOfForm(300, mk("Name")), fitOfForm(300, mk(longLine))
	if long < short+15 {
		t.Errorf("fit %d with a long label, %d with a short one", long, short)
	}
}

// Of Stacked and SettingsRows, the last call wins.
func TestSettingsRowsAndStackedLastCallWins(t *testing.T) {
	a := Fields().SettingsRows().Stacked().Row("Name", wbar("c1", 60, 24))
	b := Fields().Stacked().SettingsRows().Row("Name", wbar("c1", 60, 24))
	ta, tb := NewTester(t, a, 400, 300), NewTester(t, b, 400, 300)
	if l, c := find(t, ta.Tester, "Name"), find(t, ta.Tester, "c1"); c.Y <= l.Y {
		t.Errorf("Stacked last: control %v is not under label %v", c, l)
	}
	if l, c := find(t, tb.Tester, "Name"), find(t, tb.Tester, "c1"); c.Y > l.Y+l.H {
		t.Errorf("SettingsRows last: control %v is under label %v", c, l)
	}
}
