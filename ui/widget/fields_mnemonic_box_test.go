package widget

import (
	"testing"

	"github.com/egoist/mygo/internal/allocrun"
	"github.com/egoist/mygo/ui"
)

// altKey presses Alt+k in tt and lets the focus request it posts land.
func altKey(tt *Tester, k ui.Key) {
	tt.Settle()
	tt.Key(ui.Alt, k)
	tt.Settle()
	<-posted()
	tt.Settle()
}

// combo is a ComboBox with an option: one with none draws no node.
func combo() *ComboBoxWidget { return ComboBox(Option{"a", "Apple"}) }

func comboPlus() *BoxWidget {
	return HBox(0).Pack(combo(), true, true).Add(Button("+"))
}

// fieldsKinds are the ways a Fields lays out its rows.
var fieldsKinds = map[string]func(f *FieldsWidget) *FieldsWidget{
	"grid":     func(f *FieldsWidget) *FieldsWidget { return f },
	"stacked":  func(f *FieldsWidget) *FieldsWidget { return f.Stacked() },
	"settings": func(f *FieldsWidget) *FieldsWidget { return f.SettingsRows() },
	"wide":     func(f *FieldsWidget) *FieldsWidget { return f.Wide() },
}

func formOf(t *testing.T, c Control) *Tester {
	f := NewForm("m", 500, 300)
	f.SetContent(c)
	tt := TestForm(t, f)
	tt.Settle()
	return tt
}

// Case 1.
func TestRowKeyOfAContainerFocusesTheFirstControlInIt(t *testing.T) {
	withMac(t, false)
	for name, lay := range fieldsKinds {
		t.Run(name, func(t *testing.T) {
			tt := formOf(t, lay(Fields()).Row("&Kịch bản", comboPlus()))
			altKey(tt, ui.KeyK)
			if !tt.Find(ByRole(ui.RolePopUpButton)).Focused() {
				t.Error("Alt+K did not focus the combo")
			}
			if tt.Find(ByLabel("+")).Focused() {
				t.Error("Alt+K focused the button")
			}
		})
	}
}

// Case 2.
func TestRowKeyFocusesTheFirstInTabOrderNotTheCombo(t *testing.T) {
	withMac(t, false)
	for name, lay := range fieldsKinds {
		t.Run(name, func(t *testing.T) {
			box := HBox(0).Add(Button("+")).Pack(combo(), true, true)
			tt := formOf(t, lay(Fields()).Row("&Kịch bản", box))
			altKey(tt, ui.KeyK)
			if !tt.Find(ByLabel("+")).Focused() {
				t.Error("Alt+K did not focus the first control, the button")
			}
			if tt.Find(ByRole(ui.RolePopUpButton)).Focused() {
				t.Error("Alt+K focused the combo behind it")
			}
		})
	}
}

// Case 3: disabled and hidden first controls are passed.
func TestRowKeySkipsControlsThatCannotTakeTheFocus(t *testing.T) {
	withMac(t, false)
	for name, mk := range map[string]func() *BoxWidget{
		"disabled": func() *BoxWidget { return HBox(0).Add(Button("+").SetDisabled(true)).Add(Button("go")) },
		"hidden":   func() *BoxWidget { return HBox(0).Add(Button("+").SetVisible(false)).Add(Button("go")) },
	} {
		t.Run(name, func(t *testing.T) {
			tt := formOf(t, Fields().Row("&Kịch bản", mk()))
			altKey(tt, ui.KeyK)
			if !tt.Find(ByLabel("go")).Focused() {
				t.Error("Alt+K did not focus the next control that can")
			}
		})
	}
}

// Case 4.
func TestRowKeyReachesIntoNestedContainers(t *testing.T) {
	withMac(t, false)
	box := VBox(0).Add(HBox(0).Add(TextBox()), Button("x"))
	tt := formOf(t, Fields().Row("&K", box))
	altKey(tt, ui.KeyK)
	if !tt.Find(ByRole(ui.RoleTextField)).Focused() {
		t.Error("Alt+K did not focus the text box in the inner box")
	}
	if tt.Find(ByLabel("x")).Focused() {
		t.Error("Alt+K focused the button")
	}
}

// Case 5.
func TestRowKeyOfAContainerWithNothingToFocusDoesNothing(t *testing.T) {
	withMac(t, false)
	box := HBox(0).Add(Label("a"), Label("b"))
	tt := formOf(t, VBox(0).Add(Fields().Row("&K", box), TextBox()))
	tt.Find(ByRole(ui.RoleTextField)).Click()
	tt.Settle()
	altKey(tt, ui.KeyK)
	if !tt.Find(ByRole(ui.RoleTextField)).Focused() {
		t.Error("the focus left where it was")
	}
}

// Case 6: a row whose control takes the focus itself.
func TestRowKeyOfAControlThatFocusesItselfIsUnchanged(t *testing.T) {
	withMac(t, false)
	for name, c := range map[string]struct {
		w    Control
		role ui.Role
	}{
		"textbox": {TextBox(), ui.RoleTextField},
		"combo":   {combo(), ui.RolePopUpButton},
	} {
		t.Run(name, func(t *testing.T) {
			tt := formOf(t, Fields().Row("&K", c.w).Row("Other", TextBox()))
			altKey(tt, ui.KeyK)
			if !tt.Find(ByRole(c.role).Nth(0)).Focused() {
				t.Error("Alt+K did not focus the control")
			}
		})
	}
}

// Case 7: both rows ask for K.
func TestTwoRowsSharingAKeyMoveFromTheInnerControlToTheNextRow(t *testing.T) {
	withMac(t, false)
	tt := formOf(t, Fields().Row("&Kind", comboPlus()).Row("&Key", TextBox()))
	altKey(tt, ui.KeyK)
	if !tt.Find(ByRole(ui.RolePopUpButton)).Focused() {
		t.Fatal("the first Alt+K did not focus the combo")
	}
	altKey(tt, ui.KeyK)
	if !tt.Find(ByRole(ui.RoleTextField)).Focused() {
		t.Error("the second Alt+K did not move to the second row's text box")
	}
}

// Case 8.
func TestRowLabelNamesTheComboAndNoOtherNode(t *testing.T) {
	withMac(t, false)
	tt := formOf(t, Fields().Row("&Kịch bản", comboPlus()))
	if !tt.Has(ByRole(ui.RolePopUpButton).Label("Kịch bản")) {
		t.Error("the combo is not named Kịch bản")
	}
	if n := len(tt.FindAll(ByLabel("Kịch bản"))); n != 2 {
		t.Errorf("%d nodes carry the label, want the label and the combo", n)
	}
}

// Case 9.
func TestLabelBeforeABoxFocusesItsFirstControl(t *testing.T) {
	withMac(t, false)
	tt := formOf(t, VBox(0).Add(Label("&Tên").SetUseMnemonic(true), HBox(0).Add(TextBox(), Button("+"))))
	altKey(tt, ui.KeyT)
	if !tt.Find(ByRole(ui.RoleTextField)).Focused() {
		t.Error("Alt+T did not focus the text box")
	}
}

// Case 10: a container row costs a frame nothing a row of a control does not.
func TestRowKeyOfAContainerAddsNoAllocationsToAFrame(t *testing.T) {
	withMac(t, false)
	frame := func(label string, c Control) float64 {
		f := NewForm("m", 500, 300)
		f.SetContent(Fields().Row(label, c))
		tt := TestForm(t, f)
		for range 10 {
			tt.Frame()
		}
		return allocrun.Least(30, tt.Frame)
	}
	leaf := frame("&K", TextBox()) - frame("K", TextBox())
	box := frame("&K", comboPlus()) - frame("K", comboPlus())
	if box > leaf {
		t.Errorf("a key on a container costs %.1f allocations a frame, on a control %.1f", box, leaf)
	}
}
