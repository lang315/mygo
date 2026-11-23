package widget

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// Case 5: Parent is NewDialog's parent, nil for none.
func TestDialogParent(t *testing.T) {
	f := NewForm("P", 600, 400)
	if p := NewDialog(f, "T", "", 300, 200, Label("x")).Parent(); p != f {
		t.Errorf("Parent() is %p, want the form %p", p, f)
	}
	if p := NewDialog(nil, "T", "", 300, 200, Label("x")).Parent(); p != nil {
		t.Errorf("Parent() of a dialog over none is %p, want nil", p)
	}
}

// Case 6: after the dialog closes, Parent is still the form, and a handler
// that ran the dialog shows a toast on it once Run returns.
func TestDialogParentAfterCloseShowsAToast(t *testing.T) {
	f, tt := toastForm(t, 600, 400, nil)
	var d *Dialog
	d = NewDialog(f, "Ask", "", 300, 200, Button("Done").OnClick(func() { d.Close() }))
	dt := TestDialog(t, d)
	ran := runAsync(func() bool {
		Update(func() {
			d.Run()
			if p := d.Parent(); p != nil {
				p.ShowToast(Toast{Status: StatusSuccess, Title: "Saved"})
			}
		})
		return true
	})
	if err := dt.Click("Done"); err != nil {
		t.Fatal(err)
	}
	received(t, ran)
	if !dt.Closed() {
		t.Fatal("the dialog did not close")
	}
	if p := d.Parent(); p != f {
		t.Errorf("Parent() after close is %p, want the form %p", p, f)
	}
	tt.Frame()
	c := toastCard(t, tt)
	if c.Role != ui.RoleStatus {
		t.Errorf("the toast's role is %v, want RoleStatus", c.Role)
	}
	if r := tt.Find(ByLabel("Saved")).Bounds(); !inside(r, c.Bounds) {
		t.Errorf("%q at %v is not inside the toast %v", "Saved", r, c.Bounds)
	}
}

// Case 7: Parent from another goroutine while the dialog runs does not
// race (run with -race).
func TestDialogParentWhileRunning(t *testing.T) {
	f := NewForm("P", 600, 400)
	started := make(chan bool, 1)
	d := NewDialog(f, "Ask", "", 300, 200, Label("x")).OnBeforeRun(func() { started <- true })
	dt := TestDialog(t, d)
	ran := runAsync(func() bool { d.Run(); return true })
	received(t, started)
	got := runAsync(func() *Form { return d.Parent() })
	if p := received(t, got); p != f {
		t.Errorf("Parent() while running is %p, want %p", p, f)
	}
	if dt.Closed() {
		t.Error("the dialog closed before Close")
	}
	d.Close()
	received(t, ran)
}
