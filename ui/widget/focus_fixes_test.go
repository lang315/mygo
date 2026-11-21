package widget

import "testing"

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
