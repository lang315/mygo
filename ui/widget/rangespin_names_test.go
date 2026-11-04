package widget

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

func spinNames(tt *ui.Tester) (fields, buttons []string) {
	for _, n := range tt.Nodes() {
		switch n.Role {
		case ui.RoleTextField:
			fields = append(fields, n.Label)
		case ui.RoleButton:
			buttons = append(buttons, n.Label)
		}
	}
	return
}

// node returns the bounds of the node whose accessibility label in Nodes()
// is exactly label: Find matches only drawn text, never such labels.
func node(t *testing.T, tt *ui.Tester, label string) ui.Rect {
	t.Helper()
	for _, n := range tt.Nodes() {
		if n.Label == label {
			return n.Bounds
		}
	}
	t.Fatalf("no node labeled %q", label)
	return ui.Rect{}
}

// clickNode clicks the node labeled label.
func clickNode(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	b := node(t, tt, label)
	tt.ClickAt(b.X+b.W/2, b.Y+b.H/2)
	tt.Frame()
}

func TestRangeSpinFieldsAreToldApartByAssistiveTechnology(t *testing.T) {
	r := RangeSpin(1, 100, 5, 100, 1, "to", "s")
	tt := tester(t, r)
	fields, buttons := spinNames(tt)
	if !slices.Equal(fields, []string{"minimum", "maximum"}) {
		t.Errorf("fields %q", fields)
	}
	if want := []string{"minimum Increase", "minimum Decrease", "maximum Increase", "maximum Decrease"}; !slices.Equal(buttons, want) {
		t.Errorf("buttons %q, want %q", buttons, want)
	}
	// Told apart, a button steps its own number.
	for _, n := range tt.Nodes() {
		if n.Label == "maximum Increase" {
			tt.ClickAt(n.Bounds.X+n.Bounds.W/2, n.Bounds.Y+n.Bounds.H/2)
		}
	}
	tt.Frame()
	if lo, hi := r.Ints(); lo != 1 || hi != 6 {
		t.Fatalf("values %d, %d after maximum Increase", lo, hi)
	}
}

func TestRangeSpinSetAccessibleNames(t *testing.T) {
	r := RangeSpin(1, 100, 5, 100, 1, "to", "s").SetAccessibleNames("Delay from", "Delay to")
	tt := tester(t, r)
	fields, _ := spinNames(tt)
	if !slices.Equal(fields, []string{"Delay from", "Delay to"}) {
		t.Errorf("fields %q", fields)
	}
}

func TestNumericUpDownAccessibleName(t *testing.T) {
	n := NumericUpDown(0, 10, 1)
	tt := tester(t, n)
	if fields, buttons := spinNames(tt); !slices.Equal(fields, []string{""}) || !slices.Equal(buttons, []string{"Increase", "Decrease"}) {
		t.Fatalf("unnamed: fields %q, buttons %q", fields, buttons)
	}
	n.SetAccessibleName("Copies")
	tt.Frame()
	fields, buttons := spinNames(tt)
	if !slices.Equal(fields, []string{"Copies"}) || !slices.Equal(buttons, []string{"Copies Increase", "Copies Decrease"}) {
		t.Fatalf("named: fields %q, buttons %q", fields, buttons)
	}
}

// SideButtons gives both numbers NumericUpDown.SideButtons' look: − and +
// beside each box, which still step their own number.
func TestRangeSpinSideButtons(t *testing.T) {
	r := RangeSpin(1, 100, 5, 100, 1, "to", "").SideButtons()
	tt := tester(t, r)
	_, buttons := spinNames(tt)
	if want := []string{"minimum Decrease", "minimum Increase", "maximum Decrease", "maximum Increase"}; !slices.Equal(buttons, want) {
		t.Fatalf("buttons %q, want %q", buttons, want)
	}
	up, down := node(t, tt, "maximum Increase"), node(t, tt, "maximum Decrease")
	if up.Y != down.Y || up.X <= down.X {
		t.Fatalf("maximum Increase %v is not after maximum Decrease %v", up, down)
	}
	if !tt.HasText("+") {
		t.Fatalf("no + in %q", tt.Texts())
	}
	clickNode(t, tt, "maximum Increase")
	clickNode(t, tt, "minimum Increase")
	if lo, hi := r.Ints(); lo != 2 || hi != 6 {
		t.Fatalf("values %d, %d after the two Increase buttons", lo, hi)
	}
}
