package widget

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// Closing a dialog with the focus in a field runs the field's OnFocusOut
// before Run returns, as GTK and WinForms tell the focused control; a show
// again starts with no stale focus-out.
func TestDialogCloseRunsFocusOutBeforeRunReturns(t *testing.T) {
	var outs []string // appended on the event goroutine, read once Run returned
	tb := TextBox().SetText("k1").Focus()
	tb.OnFocusOut(func() { outs = append(outs, tb.Text()) })
	d := NewDialog(nil, "Leave", "", 300, 200, VBox(4).Add(tb, Button("Done")))
	focused := func() bool { mu.Lock(); defer mu.Unlock(); return tb.focusIn != noView }

	ran := runAsync(func() []string { d.Run(); return slices.Clone(outs) })
	fw := waitFake(t, "Leave")
	frame(fw)
	frame(fw)
	wait.Until(t, "the field to take the focus", focused)
	d.Close()
	if got := received(t, ran); !slices.Equal(got, []string{"k1"}) {
		t.Fatalf("focus-outs when Run returned: %q, want [k1]", got)
	}

	ran = runAsync(func() []string { d.Run(); return slices.Clone(outs) })
	waitFake(t, "Leave")
	d.Close()
	if got := received(t, ran); len(got) != 1 {
		t.Fatalf("focus-outs after a show without the focus: %q, want only the first", got)
	}
}

// Headless, the same: Escape from the field runs its OnFocusOut before
// Run returns.
func TestTestDialogCloseRunsFocusOutBeforeRunReturns(t *testing.T) {
	var outs []string
	tb := TextBox()
	tb.OnFocusOut(func() { outs = append(outs, tb.Text()) })
	d := NewDialog(nil, "LeaveT", "", 300, 200, tb)
	dt := TestDialog(t, d)
	ran := runAsync(func() []string { d.Run(); return slices.Clone(outs) })
	dt.Find(ByRole(ui.RoleTextField)).Type("k1")
	dt.Escape()
	if got := received(t, ran); !slices.Equal(got, []string{"k1"}) {
		t.Fatalf("focus-outs when Run returned: %q, want [k1]", got)
	}
}
