package widget

import (
	"runtime"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// P2g, A: ListView.Focus puts the keyboard focus in the list, in every view.

// focusedList focuses lv in a Tester and reports whether the focus is in its
// list, table or grid.
func focusedList(t *testing.T, lv *ListViewWidget[vitem, string]) bool {
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

// Case 1: Details with rows.
func TestListViewFocusInDetailsFocusesTheTable(t *testing.T) {
	if !focusedList(t, newView()) {
		t.Error("Focus did not put the focus in the table")
	}
}

// Case 2: the three icon views.
func TestListViewFocusInIconViewsFocusesTheGrid(t *testing.T) {
	for _, m := range allModes[1:] {
		t.Run(m.name, func(t *testing.T) {
			if !focusedList(t, newView().SetView(m.mode)) {
				t.Error("Focus did not put the focus in the grid")
			}
		})
	}
}

// Case 3: the same with an empty state and no rows, in every view.
func TestListViewFocusWithAnEmptyStateFocusesTheList(t *testing.T) {
	calmOff()
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			if !focusedList(t, emptyView().SetEmptyState(redState()).SetView(m.mode)) {
				t.Error("Focus did not put the focus in the list")
			}
		})
	}
}

// Case 3b: no rows and no empty state.
func TestListViewFocusWithNoRowsFocusesTheList(t *testing.T) {
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			if !focusedList(t, emptyView().SetView(m.mode)) {
				t.Error("Focus did not put the focus in the list")
			}
		})
	}
}

// Case 4: a focused list switched to another view keeps the focus.
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

// Case 4b: Focus asked for after a switch from an icon view back to Details.
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

// Case 4c: Focus is asked for once: a later Focus of another control is not
// undone by the list asking again.
func TestListViewFocusIsAskedForOnce(t *testing.T) {
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

// P2g, B: an access key shared by two Fields rows cycles through them and
// wraps, whatever kind of control the row holds.

// focusKind names where the focus is: "list" in a list, "other" on any other
// node, "" nowhere.
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

// cycleF presses Alt+F n times and returns where the focus is after each.
func cycleF(tt *Tester, n int) []string {
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
		t.Skip("on macOS Option+letter types a character in a text editor (ui's editor wants it), so a form's access key never reaches the form from one")
	}
}

// Case 5b, on every OS, with no editor: a list and a button.
func TestSharedKeyOfAListAndAButtonWraps(t *testing.T) {
	calmOff()
	withMac(t, false)
	tt := formOf(t, Fields().Row("&Files", newListBox()).Row("&Find", Button("Go")))
	if got, want := cycleF(tt, 3), []string{"list", "other", "list"}; !slices.Equal(got, want) {
		t.Errorf("Alt+F focused %v, want %v", got, want)
	}
}

// Case 6b, on every OS: a button first, then a list.
func TestSharedKeyOfAButtonAndAListWraps(t *testing.T) {
	calmOff()
	withMac(t, false)
	tt := formOf(t, Fields().Row("&Find", Button("Go")).Row("&Files", newListBox()))
	if got, want := cycleF(tt, 3), []string{"other", "list", "other"}; !slices.Equal(got, want) {
		t.Errorf("Alt+F focused %v, want %v", got, want)
	}
}

// Case 5.
func TestSharedKeyFromAListToATextBoxWraps(t *testing.T) {
	skipWhereOptionTypes(t)
	calmOff()
	withMac(t, false)
	tt := formOf(t, Fields().Row("&Files", newListBox()).Row("&Find", TextBox()))
	if got, want := cycleF(tt, 3), []string{"list", "other", "list"}; !slices.Equal(got, want) {
		t.Errorf("Alt+F focused %v, want %v", got, want)
	}
}

// Case 6.
func TestSharedKeyFromATextBoxToAListWraps(t *testing.T) {
	skipWhereOptionTypes(t)
	calmOff()
	withMac(t, false)
	tt := formOf(t, Fields().Row("&Find", TextBox()).Row("&Files", newListBox()))
	if got, want := cycleF(tt, 3), []string{"other", "list", "other"}; !slices.Equal(got, want) {
		t.Errorf("Alt+F focused %v, want %v", got, want)
	}
}

// Round 2, on every OS: the press from the list to the control (the first
// Alt+F focuses the list, the second the control) puts the focus on it.
func TestSharedKeyFromAListFocusesTheInputKind(t *testing.T) {
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
			if got, want := cycleF(tt, 2), []string{"list", "other"}; !slices.Equal(got, want) {
				t.Errorf("Alt+F focused %v, want %v", got, want)
			}
		})
	}
}

// Round 2: Focus of an editable combo or a date picker focuses its input, with
// the focus in a list first.
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

// Case 7: every kind that wraps an input in a field, in a pair with a list.
// The kinds: TextBox (above), NumericUpDown, an editable ComboBox,
// DateTimePicker. There is no SearchBox in the package.
func TestSharedKeyMovesOnFromEveryInputKind(t *testing.T) {
	skipWhereOptionTypes(t)
	calmOff()
	withMac(t, false)
	kinds := map[string]func() Control{
		"NumericUpDown":  func() Control { return NumericUpDown(0, 10, 1) },
		"ComboBox":       func() Control { return ComboBox(Option{"a", "Apple"}).Editable() },
		"DateTimePicker": func() Control { return DateTimePicker() },
	}
	for name, mk := range kinds {
		t.Run(name, func(t *testing.T) {
			tt := formOf(t, Fields().Row("&Files", newListBox()).Row("&Find", mk()))
			if got, want := cycleF(tt, 3), []string{"list", "other", "list"}; !slices.Equal(got, want) {
				t.Errorf("Alt+F focused %v, want %v", got, want)
			}
		})
	}
}
