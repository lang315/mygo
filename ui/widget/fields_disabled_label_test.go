package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// A Fields row whose control is disabled draws its label, and its Describe
// text, as disabled, as the control's own text is. A disabled text is a node
// with the disabled state (its pixels are alike an enabled one's but for a
// skin's look), so the tests read the nodes, against a Label disabled by
// hand in the same test.

// dlLayouts build a form of the rows `row` adds, laid out each way: a grid,
// stacked rows, settings rows, and settings rows with a wide row.
var dlLayouts = map[string]func(row func(f *FieldsWidget) *FieldsWidget) *FieldsWidget{
	"grid":     func(row func(f *FieldsWidget) *FieldsWidget) *FieldsWidget { return row(Fields()) },
	"stacked":  func(row func(f *FieldsWidget) *FieldsWidget) *FieldsWidget { return row(Fields().Stacked()) },
	"settings": func(row func(f *FieldsWidget) *FieldsWidget) *FieldsWidget { return row(Fields().SettingsRows()) },
	"wide": func(row func(f *FieldsWidget) *FieldsWidget) *FieldsWidget {
		return row(Fields().SettingsRows()).Wide()
	},
}

// dlText reports whether the text s of tt is disabled.
func dlText(t *testing.T, tt *Tester, s string) bool {
	t.Helper()
	tt.Settle()
	return tt.Find(ByRole(ui.RoleText).Label(s)).Disabled()
}

// dlWant reports whether a Label with the text s, disabled when off, is a
// disabled node: what the label of a Fields row is to be.
func dlWant(t *testing.T, s string, off bool) bool {
	t.Helper()
	return dlText(t, NewTester(t, Label(s).SetDisabled(off), 100, 40), s)
}

// dlForm is a form of one row Alpha, described by Note, with the control w.
func dlForm(t *testing.T, lay string, w Control) *Tester {
	t.Helper()
	f := dlLayouts[lay](func(f *FieldsWidget) *FieldsWidget { return f.Row("Alpha", w).Describe("Note") })
	tt := NewTester(t, f, 400, 200)
	tt.Settle()
	return tt
}

// dlCheck fails the test unless the label and the description of tt are
// disabled as a Label disabled by hand is when off, and not as one that is not.
func dlCheck(t *testing.T, tt *Tester, off bool) {
	t.Helper()
	if !dlWant(t, "Alpha", true) || dlWant(t, "Alpha", false) {
		t.Fatal("a Label does not tell disabled from enabled")
	}
	if got := dlText(t, tt, "Alpha"); got != off {
		t.Errorf("the label is disabled: %v, want %v", got, off)
	}
	if got := dlText(t, tt, "Note"); got != off {
		t.Errorf("the description is disabled: %v, want %v", got, off)
	}
}

// Cases 1 and 5: the label and the description of a row with a disabled
// control are disabled, those of an enabled one are not.
func TestFieldsLabelDimsWithItsDisabledControl(t *testing.T) {
	for lay := range dlLayouts {
		t.Run(lay, func(t *testing.T) {
			dlCheck(t, dlForm(t, lay, TextBox().SetDisabled(true)), true)
			dlCheck(t, dlForm(t, lay, TextBox()), false)
		})
	}
}

// Case 2: enabling the control again restores the label on the next frame,
// and disabling it dims it again.
func TestFieldsLabelComesBackWithItsControl(t *testing.T) {
	for lay := range dlLayouts {
		t.Run(lay, func(t *testing.T) {
			tb := TextBox().SetDisabled(true)
			tt := dlForm(t, lay, tb)
			dlCheck(t, tt, true)
			tb.SetDisabled(false)
			tt.Frame()
			dlCheck(t, tt, false)
			tb.SetDisabled(true)
			tt.Frame()
			dlCheck(t, tt, true)
		})
	}
}

// A label disabled by hand stays so, whatever its control does.
func TestFieldsLabelDisabledByHandStays(t *testing.T) {
	tb := TextBox()
	f := Fields().Row("Alpha", tb)
	f.Label(0).SetDisabled(true)
	tt := NewTester(t, f, 400, 200)
	for _, off := range []bool{false, true, false} {
		tb.SetDisabled(off)
		tt.Frame()
		if !dlText(t, tt, "Alpha") {
			t.Fatalf("the control disabled %v, the label disabled by hand is enabled", off)
		}
	}
}

// Case 3: a disabled container dims its row's label.
func TestFieldsLabelDimsWithItsDisabledBox(t *testing.T) {
	for lay := range dlLayouts {
		t.Run(lay, func(t *testing.T) {
			box := VBox(0).Add(TextBox())
			tt := dlForm(t, lay, box.SetDisabled(true))
			dlCheck(t, tt, true)
			box.SetDisabled(false)
			tt.Frame()
			dlCheck(t, tt, false)
		})
	}
}

// Case 4: the label's access key does not focus a disabled control.
func TestFieldsKeyDoesNotFocusADisabledControl(t *testing.T) {
	withMac(t, false)
	for name, lay := range dlLayouts {
		t.Run(name, func(t *testing.T) {
			tb := TextBox().SetDisabled(true)
			f := lay(func(f *FieldsWidget) *FieldsWidget { return f.Row("&Kịch bản", tb) })
			tt := formOf(t, f)
			altKey(tt, ui.KeyK)
			if tt.Find(ByRole(ui.RoleTextField)).Focused() {
				t.Error("Alt+K focused the disabled control")
			}
			tb.SetDisabled(false)
			tt.Settle()
			altKey(tt, ui.KeyK)
			if !tt.Find(ByRole(ui.RoleTextField)).Focused() {
				t.Error("Alt+K did not focus the control once enabled")
			}
		})
	}
}
