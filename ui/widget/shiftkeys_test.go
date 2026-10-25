package widget

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Going back with Cmd+Shift takes out the rows the extension chose in the
// controls built on a list or a grid, which keep the choice in their own
// Selection: nothing they do between two keys makes the list forget it.
func TestCmdShiftKeysGoingBackTakesOutTheRowsInAListBox(t *testing.T) {
	lb := fruitBox().Multi()
	tt := tester(t, lb)
	click(t, tt, "Banana")
	if err := tt.ClickWith(ui.Cmd, "Apple"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd|ui.Shift, ui.KeyDown)
	tt.Frame()
	tt.Key(ui.Cmd|ui.Shift, ui.KeyDown)
	tt.Frame()
	if got := lb.Selected(); !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Fatalf("Cmd+Shift+Down twice from Apple chose %v", got)
	}
	tt.Key(ui.Cmd|ui.Shift, ui.KeyUp)
	tt.Frame()
	if got := lb.Selected(); !slices.Equal(got, []string{"a", "b"}) {
		t.Fatalf("Cmd+Shift+Up chose %v, want Cherry out and Banana, chosen before, kept", got)
	}
}

func TestCmdShiftKeysGoingBackTakesOutTheRowsInEveryListView(t *testing.T) {
	for _, m := range allModes {
		t.Run(m.name, func(t *testing.T) {
			lv := newView().SetView(m.mode)
			next := ui.KeyRight
			if m.mode == ViewDetails {
				next = ui.KeyDown
			}
			back := ui.KeyLeft
			if m.mode == ViewDetails {
				back = ui.KeyUp
			}
			tt := tester(t, lv)
			click(t, tt, "Banana")
			if err := tt.ClickWith(ui.Cmd, "Apple"); err != nil {
				t.Fatal(err)
			}
			for range 2 {
				tt.Key(ui.Cmd|ui.Shift, next)
				tt.Frame()
			}
			if got := lv.Selected(); !slices.Equal(got, []string{"a", "b", "c"}) {
				t.Fatalf("Cmd+Shift twice from Apple chose %v", got)
			}
			tt.Key(ui.Cmd|ui.Shift, back)
			tt.Frame()
			if got := lv.Selected(); !slices.Equal(got, []string{"a", "b"}) {
				t.Fatalf("Cmd+Shift back chose %v, want Cherry out and Banana, chosen before, kept", got)
			}
		})
	}
}
