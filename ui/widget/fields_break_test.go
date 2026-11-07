package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

var breakModes = []string{"default", "stacked", "settings"}

func inMode(f *FieldsWidget, mode string) *FieldsWidget {
	switch mode {
	case "stacked":
		f.Stacked()
	case "settings":
		f.SettingsRows()
	}
	return f
}

// top is where a row starts, as far as the gap above it goes: its label
// when stacked, else its control, which is as high as the row.
func top(t *testing.T, tt *Tester, mode, label, control string) float32 {
	t.Helper()
	if mode == "stacked" {
		return find(t, tt.Tester, label).Y
	}
	return find(t, tt.Tester, control).Y
}

// A group break is a gap, not a heading: it has no node in the tree.
func TestFieldsEmptySectionHasNoNode(t *testing.T) {
	for _, mode := range breakModes {
		f := inMode(Fields().Row("A", bar("c1", 24)).Section("").Row("B", bar("c2", 24)), mode)
		tt := sectionTester(t, f)
		if a, ok := tt.AccessByRole(ui.RoleHeading, ""); ok {
			t.Errorf("%s: Section(\"\") has a heading node %+v", mode, a)
		}
	}
}

// The gap above a group is larger than one between rows, and as large as the
// one above a titled section's heading.
func TestFieldsEmptySectionGap(t *testing.T) {
	for _, mode := range breakModes {
		build := func(title string) (*Tester, float32, float32) {
			f := inMode(Fields().Row("A", bar("c1", 24)).Row("B", bar("c2", 24)).
				Section(title).Row("C", bar("c3", 24)), mode)
			tt := sectionTester(t, f)
			prev := find(t, tt.Tester, "c2")
			row := top(t, tt, mode, "B", "c2") - (find(t, tt.Tester, "c1").Y + 24)
			if title == "" {
				return tt, top(t, tt, mode, "C", "c3") - (prev.Y + 24), row
			}
			return tt, find(t, tt.Tester, title).Y - (prev.Y + 24), row
		}
		_, group, row := build("")
		_, titled, _ := build("Heading")
		if group <= row+1 {
			t.Errorf("%s: gap between groups %v, between rows %v: want larger", mode, group, row)
		}
		if !approx(group, titled) {
			t.Errorf("%s: gap between groups %v, above a titled section %v: want equal", mode, group, titled)
		}
	}
}

// The label column is shared across a break.
func TestFieldsEmptySectionKeepsLabelColumn(t *testing.T) {
	f := Fields().Row("A long label here", bar("c1", 24)).Section("").Row("Id", bar("c2", 24))
	tt := sectionTester(t, f)
	if a, b := find(t, tt.Tester, "c1"), find(t, tt.Tester, "c2"); !approx(a.X, b.X) {
		t.Fatalf("controls at x %v and %v: want one column", a.X, b.X)
	}
}

// A break first adds no gap; one whose rows are all hidden takes its gap away.
func TestFieldsEmptySectionCollapses(t *testing.T) {
	for _, mode := range breakModes {
		cs := [4]*CustomWidget{bar("c1", 24), bar("c2", 24), bar("c3", 24), bar("c4", 24)}
		f := inMode(Fields().Section("").Row("A", cs[0]).Row("B", cs[1]).
			Section("").Row("C", cs[2]).Section("").Row("D", cs[3]), mode)
		tt := sectionTester(t, f)
		base := sectionTester(t, inMode(Fields().Row("A", bar("c1", 24)), mode))
		if a, b := top(t, tt, mode, "A", "c1"), top(t, base, mode, "A", "c1"); !approx(a, b) {
			t.Fatalf("%s: first row at y %v, alone at %v: a break first adds a gap", mode, a, b)
		}
		gap := func(prev, label, control string) float32 {
			return top(t, tt, mode, label, control) - (find(t, tt.Tester, prev).Y + 24)
		}
		one := gap("c2", "C", "c3")

		cs[2].SetVisible(false)
		tt.Frame()
		if g := gap("c2", "D", "c4"); !approx(g, one) {
			t.Fatalf("%s: gap above D %v with C hidden, %v with C shown: want one break left", mode, g, one)
		}

		cs[0].SetVisible(false)
		cs[1].SetVisible(false)
		tt.Frame()
		if a, b := top(t, tt, mode, "D", "c4"), top(t, base, mode, "A", "c1"); !approx(a, b) {
			t.Fatalf("%s: D at y %v, a first row at %v: a break above the first shown row adds a gap", mode, a, b)
		}
	}
}
