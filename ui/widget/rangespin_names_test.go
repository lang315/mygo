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

func TestRangeSpinFieldsAreToldApartByAssistiveTechnology(t *testing.T) {
	r := RangeSpin(1, 100, 5, 100, 1, "to", "s")
	tt := tester(t, r)
	fields, buttons := spinNames(tt)
	if !slices.Equal(fields, []string{"minimum", "maximum"}) {
		t.Errorf("fields %q", fields)
	}
	if want := []string{"minimum Decrease", "minimum Increase", "maximum Decrease", "maximum Increase"}; !slices.Equal(buttons, want) {
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
	if fields, buttons := spinNames(tt); !slices.Equal(fields, []string{""}) || !slices.Equal(buttons, []string{"Decrease", "Increase"}) {
		t.Fatalf("unnamed: fields %q, buttons %q", fields, buttons)
	}
	n.SetAccessibleName("Copies")
	tt.Frame()
	fields, buttons := spinNames(tt)
	if !slices.Equal(fields, []string{"Copies"}) || !slices.Equal(buttons, []string{"Copies Decrease", "Copies Increase"}) {
		t.Fatalf("named: fields %q, buttons %q", fields, buttons)
	}
}
