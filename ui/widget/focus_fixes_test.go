package widget

import (
	"runtime"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

func focusPutsFocusInList(t *testing.T, lv *ListViewWidget[vitem, string]) bool {
	t.Helper()
	tt := tester(t, lv)
	if focusInList(tt) {
		t.Fatal("the list has the focus before Focus was asked for")
	}
	lv.Focus()
	tt.Frame()
	tt.Frame()
	return focusInList(tt)
}

func TestListViewFocusInDetailsFocusesTheTable(t *testing.T) {
	if !focusPutsFocusInList(t, newView()) {
		t.Error("Focus did not put the focus in the table")
	}
}

func TestListViewFocusInIconViewsFocusesTheGrid(t *testing.T) {
	for _, m := range allModes[1:] {
		t.Run(m.name, func(t *testing.T) {
			if !focusPutsFocusInList(t, newView().SetView(m.mode)) {
				t.Error("Focus did not put the focus in the grid")
			}
		})
	}
}

func TestListViewFocusWithAnEmptyStateFocusesTheList(t *testing.T) {
	calmOff()
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			if !focusPutsFocusInList(t, emptyView().SetEmptyState(redState()).SetView(m.mode)) {
				t.Error("Focus did not put the focus in the list")
			}
		})
	}
}

func TestListViewFocusWithNoRowsFocusesTheList(t *testing.T) {
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			if !focusPutsFocusInList(t, emptyView().SetView(m.mode)) {
				t.Error("Focus did not put the focus in the list")
			}
		})
	}
}

func TestListViewSwitchedWhileFocusedKeepsTheFocus(t *testing.T) {
	lv := newView()
	tt := tester(t, lv)
	lv.Focus()
	tt.Frame()
	tt.Frame()
	lv.SetView(ViewLargeIcon)
	tt.Frame()
	tt.Frame()
	if !focusInList(tt) {
		t.Error("the focus left the list when the view switched")
	}
}

func TestListViewFocusAfterSwitchingBackFocusesTheTable(t *testing.T) {
	lv := newView().SetView(ViewList)
	tt := tester(t, lv)
	lv.SetView(ViewDetails)
	tt.Frame()
	lv.Focus()
	tt.Frame()
	tt.Frame()
	if !focusInList(tt) {
		t.Error("Focus did not put the focus in the table")
	}
}

func TestListViewFocusDoesNotTakeBackTheFocusFromALaterFocus(t *testing.T) {
	lv := newView()
	other := TextBox()
	tt := tester(t, VBox(0).Add(lv).Add(other))
	lv.Focus()
	tt.Frame()
	tt.Frame()
	other.Focus()
	tt.Frame()
	tt.Frame()
	if focusInList(tt) {
		t.Error("the list took the focus back from the text box")
	}
}

func focusKind(tt *Tester) string {
	if focusInList(tt.Tester) {
		return "list"
	}
	for _, n := range tt.Nodes() {
		if n.Focused {
			return "other"
		}
	}
	return ""
}

func focusAfterEachAltF(tt *Tester, n int) []string {
	var got []string
	for range n {
		altKey(tt, ui.KeyF)
		got = append(got, focusKind(tt))
	}
	return got
}

func skipWhereOptionTypes(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "darwin" {
		t.Skip("on macOS Option+letter types a character in a text editor (ui/editor_navigation.go), so a form's access key never reaches the form from one")
	}
}

func TestSharedKeyOfAListAndAButtonWraps(t *testing.T) {
	calmOff()
	withMac(t, false)
	tt := formOf(t, Fields().Row("&Files", newListBox()).Row("&Find", Button("Go")))
	if got, want := focusAfterEachAltF(tt, 3), []string{"list", "other", "list"}; !slices.Equal(got, want) {
		t.Errorf("Alt+F focused %v, want %v", got, want)
	}
}

func TestSharedKeyOfAButtonAndAListWraps(t *testing.T) {
	calmOff()
	withMac(t, false)
	tt := formOf(t, Fields().Row("&Find", Button("Go")).Row("&Files", newListBox()))
	if got, want := focusAfterEachAltF(tt, 3), []string{"other", "list", "other"}; !slices.Equal(got, want) {
		t.Errorf("Alt+F focused %v, want %v", got, want)
	}
}

func TestSharedKeyFromAListToATextBoxWraps(t *testing.T) {
	skipWhereOptionTypes(t)
	calmOff()
	withMac(t, false)
	tt := formOf(t, Fields().Row("&Files", newListBox()).Row("&Find", TextBox()))
	if got, want := focusAfterEachAltF(tt, 3), []string{"list", "other", "list"}; !slices.Equal(got, want) {
		t.Errorf("Alt+F focused %v, want %v", got, want)
	}
}

func TestSharedKeyFromATextBoxToAListWraps(t *testing.T) {
	skipWhereOptionTypes(t)
	calmOff()
	withMac(t, false)
	tt := formOf(t, Fields().Row("&Find", TextBox()).Row("&Files", newListBox()))
	if got, want := focusAfterEachAltF(tt, 3), []string{"other", "list", "other"}; !slices.Equal(got, want) {
		t.Errorf("Alt+F focused %v, want %v", got, want)
	}
}

func TestSharedKeyMovesFromAListIntoEveryInputKind(t *testing.T) {
	calmOff()
	withMac(t, false)
	kinds := map[string]func() Control{
		"NumericUpDown":    func() Control { return NumericUpDown(0, 10, 1) },
		"ComboBox":         func() Control { return ComboBox(Option{"a", "Apple"}) },
		"EditableComboBox": func() Control { return ComboBox(Option{"a", "Apple"}).Editable() },
		"DateTimePicker":   func() Control { return DateTimePicker() },
		"DateOnly":         func() Control { return DateTimePicker().SetFormat(DateOnly) },
		"TimeOnly":         func() Control { return DateTimePicker().SetFormat(TimeOnly) },
	}
	for name, mk := range kinds {
		t.Run(name, func(t *testing.T) {
			tt := formOf(t, Fields().Row("&Files", newListBox()).Row("&Find", mk()))
			if got, want := focusAfterEachAltF(tt, 2), []string{"list", "other"}; !slices.Equal(got, want) {
				t.Errorf("Alt+F focused %v, want %v", got, want)
			}
		})
	}
}

func TestFocusOfAnInputKindTakesTheFocusFromAList(t *testing.T) {
	calmOff()
	withMac(t, false)
	kinds := map[string]func() Control{
		"EditableComboBox": func() Control { return ComboBox(Option{"a", "Apple"}).Editable() },
		"DateTimePicker":   func() Control { return DateTimePicker() },
	}
	for name, mk := range kinds {
		t.Run(name, func(t *testing.T) {
			lb, c := newListBox(), mk()
			tt := formOf(t, Fields().Row("&Files", lb).Row("&Find", c))
			lb.Focus()
			tt.Settle()
			if focusKind(tt) != "list" {
				t.Fatal("the list did not take the focus")
			}
			c.base().change(func() { c.base().wantFocus = true })
			tt.Settle()
			if got := focusKind(tt); got != "other" {
				t.Errorf("focus is %q after Focus, want it on the control's input", got)
			}
		})
	}
}

func TestSharedKeyMovesOutOfEveryInputKind(t *testing.T) {
	skipWhereOptionTypes(t)
	calmOff()
	withMac(t, false)
	kinds := map[string]func() Control{
		"NumericUpDown":    func() Control { return NumericUpDown(0, 10, 1) },
		"EditableComboBox": func() Control { return ComboBox(Option{"a", "Apple"}).Editable() },
		"DateTimePicker":   func() Control { return DateTimePicker() },
	}
	for name, mk := range kinds {
		t.Run(name, func(t *testing.T) {
			tt := formOf(t, Fields().Row("&Files", newListBox()).Row("&Find", mk()))
			if got, want := focusAfterEachAltF(tt, 3), []string{"list", "other", "list"}; !slices.Equal(got, want) {
				t.Errorf("Alt+F focused %v, want %v", got, want)
			}
		})
	}
}
