package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// sectionForm is three sections of rows whose controls are 24 DIPs high: the
// long labels are in the first section, the short ones in the others.
func sectionForm() (*FieldsWidget, [4]*CustomWidget) {
	cs := [4]*CustomWidget{bar("c1", 24), bar("c2", 24), bar("c3", 24), bar("c4", 24)}
	f := Fields().
		Section("Alpha").Row("A long label here", cs[0]).Row("Another long one", cs[1]).
		Section("Beta").Row("Id", cs[2]).
		Section("Gamma").Row("Ok", cs[3])
	return f, cs
}

func sectionTester(t *testing.T, f *FieldsWidget) *Tester {
	t.Helper()
	tt := NewTester(t, f, 800, 600)
	tt.Frame()
	return tt
}

// Rows of every section share one label column: the controls start at the
// same x, as far right as the widest label of any section needs.
func TestFieldsSectionsShareOneLabelColumn(t *testing.T) {
	f, _ := sectionForm()
	tt := sectionTester(t, f)
	c1, c3, c4 := find(t, tt.Tester, "c1"), find(t, tt.Tester, "c3"), find(t, tt.Tester, "c4")
	if !approx(c1.X, c3.X) || !approx(c1.X, c4.X) {
		t.Fatalf("controls at x %v %v %v: want one x", c1.X, c3.X, c4.X)
	}
	wide := find(t, tt.Tester, "Another long one")
	if want := wide.X + wide.W + 12; !approx(c3.X, want) {
		t.Fatalf("controls at x %v, want %v: the widest label of all sections plus 12", c3.X, want)
	}
}

// A heading starts at the label column and spans both columns.
func TestFieldsSectionHeadingSpansTheForm(t *testing.T) {
	for _, settings := range []bool{false, true} {
		f, _ := sectionForm()
		if settings {
			f.SettingsRows()
		}
		tt := sectionTester(t, f)
		h, l := find(t, tt.Tester, "Beta"), find(t, tt.Tester, "Id")
		if h.X != l.X {
			t.Errorf("settings %v: heading at x %v, the labels at %v", settings, h.X, l.X)
		}
		if h.X+h.W < 790 {
			t.Errorf("settings %v: heading %v ends before the form's right edge", settings, h)
		}
	}
}

// The first heading sits where the first row would, and a later one has more
// room above it than a row has.
func TestFieldsSectionGapAboveSection(t *testing.T) {
	f, _ := sectionForm()
	tt := sectionTester(t, f)
	base := sectionTester(t, Fields().Row("x", bar("c1", 24)))
	if a, b := find(t, tt.Tester, "Alpha"), find(t, base.Tester, "c1"); !approx(a.Y, b.Y) {
		t.Fatalf("first heading at y %v, the first row would be at %v: no extra gap above the first", a.Y, b.Y)
	}
	c1, c2 := find(t, tt.Tester, "c1"), find(t, tt.Tester, "c2")
	rowGap := c2.Y - c1.Y - 24
	beta := find(t, tt.Tester, "Beta")
	if sec := beta.Y - (c2.Y + 24); sec <= rowGap+1 {
		t.Fatalf("gap above section %v, between rows %v: want the section's larger", sec, rowGap)
	}
}

// A section whose rows are all hidden takes its heading and its gap away;
// showing a row brings both back.
func TestFieldsSectionHiddenWithItsRows(t *testing.T) {
	f, cs := sectionForm()
	tt := sectionTester(t, f)
	gamma := find(t, tt.Tester, "Gamma")
	gap := func() float32 { return find(t, tt.Tester, "Gamma").Y - (find(t, tt.Tester, "c2").Y + 24) }
	before := gap()

	cs[2].SetVisible(false)
	tt.Frame()
	if tt.HasText("Beta") {
		t.Fatalf("heading of a hidden section shows: %q", tt.Texts())
	}
	if after := gap(); after >= before-20 || after <= 1 {
		t.Fatalf("gap above Gamma %v with Beta hidden, %v with it shown: want one section gap left, not two", after, before)
	}
	if g := find(t, tt.Tester, "Gamma"); g.Y >= gamma.Y {
		t.Fatalf("Gamma at y %v, was %v: the hidden section's room remains", g.Y, gamma.Y)
	}

	cs[2].SetVisible(true)
	tt.Frame()
	if !tt.HasText("Beta") {
		t.Fatalf("heading of a section with a shown row is gone: %q", tt.Texts())
	}
	if g := find(t, tt.Tester, "Gamma"); !approx(g.Y, gamma.Y) {
		t.Fatalf("Gamma at y %v, was %v", g.Y, gamma.Y)
	}

	// Hiding the first section makes the next one first: no gap above it.
	cs[0].SetVisible(false)
	cs[1].SetVisible(false)
	tt.Frame()
	if tt.HasText("Alpha") {
		t.Fatalf("heading of a hidden section shows: %q", tt.Texts())
	}
	base := sectionTester(t, Fields().Row("x", bar("c1", 24)))
	if b, c := find(t, tt.Tester, "Beta"), find(t, base.Tester, "c1"); !approx(b.Y, c.Y) {
		t.Fatalf("Beta at y %v, the first row would be at %v: a gap above the first visible section", b.Y, c.Y)
	}
}

// A heading takes the place of the divider before its section: never one
// above a heading, and none between the heading and its first row.
func TestSettingsRowsDividersNextToHeadings(t *testing.T) {
	cs := [3]*CustomWidget{wbar("c1", 60, 24), wbar("c2", 60, 24), wbar("c3", 60, 24)}
	f := Fields().SettingsRows().
		Section("Alpha").Row("One", cs[0]).Row("Two", cs[1]).
		Section("Beta").Row("Three", cs[2])
	tt := NewTester(t, f, 400, 300)
	tt.Frame()
	left := int(find(t, tt.Tester, "c3").X) - 4
	if ys := rules(tt.Tester, left); len(ys) != 1 {
		t.Fatalf("%d dividers (at %v), want 1: the one between the rows of Alpha", len(ys), rules(tt.Tester, left))
	}
}

// A heading is a heading to assistive technology, which jumps between them,
// and not a stop of the keyboard focus.
func TestFieldsSectionHeadingIsAHeading(t *testing.T) {
	for _, mode := range []string{"default", "stacked", "settings"} {
		f, _ := sectionForm()
		switch mode {
		case "stacked":
			f.Stacked()
		case "settings":
			f.SettingsRows()
		}
		tt := sectionTester(t, f)
		for _, name := range []string{"Alpha", "Beta", "Gamma"} {
			a, ok := tt.AccessByRole(ui.RoleHeading, name)
			if !ok {
				t.Fatalf("%s: no heading %q", mode, name)
			}
			if a.Focusable {
				t.Errorf("%s: heading %q takes the focus", mode, name)
			}
		}
	}
}

// Stacked shows the headings too, with room above a later section, none above
// the first, and none for a section whose rows are all hidden.
func TestFieldsSectionStacked(t *testing.T) {
	f, cs := sectionForm()
	tt := sectionTester(t, f.Stacked())
	base := sectionTester(t, Fields().Stacked().Row("x", bar("c1", 24)))
	if a, b := find(t, tt.Tester, "Alpha"), find(t, base.Tester, "x"); !approx(a.Y, b.Y) {
		t.Fatalf("first heading at y %v, the first label would be at %v", a.Y, b.Y)
	}
	c1, c2 := find(t, tt.Tester, "c1"), find(t, tt.Tester, "c2")
	rowGap := find(t, tt.Tester, "Another long one").Y - (c1.Y + 24)
	if sec := find(t, tt.Tester, "Beta").Y - (c2.Y + 24); sec <= rowGap+1 {
		t.Fatalf("gap above section %v, between rows %v: want the section's larger", sec, rowGap)
	}
	gamma := find(t, tt.Tester, "Gamma")
	cs[2].SetVisible(false)
	tt.Frame()
	if tt.HasText("Beta") {
		t.Fatalf("heading of a hidden section shows: %q", tt.Texts())
	}
	if g := find(t, tt.Tester, "Gamma"); g.Y >= gamma.Y {
		t.Fatalf("Gamma at y %v, was %v: the hidden section's room remains", g.Y, gamma.Y)
	}
}
