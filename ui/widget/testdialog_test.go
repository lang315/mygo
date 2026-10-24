package widget

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
)

type nameDialog struct {
	*Dialog
	name   *TextBoxWidget
	result string
}

func newNameDialog() *nameDialog {
	d := &nameDialog{name: TextBox()}
	var ok, cancel *ButtonWidget
	d.Dialog = NewDialog(nil, "Name", "", 300, 160, VBox(8).Add(
		d.name,
		HBox(8).Add(
			Button("Cancel").AssignTo(&cancel).OnClick(func() { d.Close() }),
			Button("OK").AssignTo(&ok).OnClick(func() { d.result = d.name.Text(); d.Close() }),
		),
	))
	return d
}

// run is a dialog's Run that returns a value, as gota's do.
func (d *nameDialog) run() string { d.Dialog.Run(); return d.result }

func TestTestDialogAcceptReturnsTheValue(t *testing.T) {
	d := newNameDialog()
	dt := TestDialog(t, d.Dialog)
	got := make(chan string, 1)
	go func() { got <- d.run() }()
	dt.Find(ByRole(ui.RoleTextField)).Type("Ada")
	dt.Find(ByLabel("OK")).Click()
	if v := <-got; v != "Ada" {
		t.Fatalf("Run returned %q", v)
	}
	if !dt.Closed() {
		t.Fatal("not closed")
	}
}

func TestTestDialogCancelAndEscapeClose(t *testing.T) {
	d := newNameDialog()
	dt := TestDialog(t, d.Dialog)
	got := make(chan string, 1)
	go func() { got <- d.run() }()
	if dt.Closed() {
		t.Fatal("closed before any action")
	}
	dt.Escape()
	flushEvents(t)
	if v := <-got; v != "" || !dt.Closed() {
		t.Fatalf("Escape: result %q, closed %v", v, dt.Closed())
	}

	d2 := newNameDialog()
	dt2 := TestDialog(t, d2.Dialog)
	done := make(chan struct{})
	go func() { d2.Dialog.Run(); close(done) }()
	if err := dt2.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	flushEvents(t)
	<-done
	if !dt2.Closed() {
		t.Fatal("Cancel did not close")
	}
}

func TestTestDialogEscapeClosesAPopupFirstAndRespectsSetDeletable(t *testing.T) {
	combo := ComboBox(Option{"a", "Apple"}, Option{"b", "Banana"}).SetValue("a")
	d := NewDialog(nil, "Pick", "", 300, 200, combo)
	dt := TestDialog(t, d)
	if err := dt.Click("Apple"); err != nil { // opens the popup
		t.Fatal(err)
	}
	dt.Frame()
	if !dt.HasText("Banana") {
		t.Fatalf("no popup in %q", dt.Texts())
	}
	dt.Escape() // closes the popup, not the dialog
	dt.Frame()
	if dt.Closed() || dt.HasText("Banana") {
		t.Fatalf("closed %v, texts %q: the first Escape must only close the popup", dt.Closed(), dt.Texts())
	}
	d.SetDeletable(false)
	dt.Escape()
	if dt.Closed() {
		t.Fatal("Escape closed a dialog that is not deletable")
	}
	d.Close() // Close still works
	if !dt.Closed() {
		t.Fatal("Close did not close")
	}
}

func TestTestDialogRunsOnBeforeRun(t *testing.T) {
	ran := make(chan struct{}, 1)
	d := NewDialog(nil, "X", "", 200, 100, Label("hi")).OnBeforeRun(func() { ran <- struct{}{} })
	dt := TestDialog(t, d)
	go d.Run()
	flushEvents(t)
	<-ran
	d.Close()
	if !dt.HasText("hi") {
		t.Fatalf("texts %q", dt.Texts())
	}
}

// flushEvents waits for what was posted to the event goroutine.
func flushEvents(t *testing.T) { t.Helper(); <-posted() }

func TestTestDialogOfNaturalHeightIsDrawn(t *testing.T) {
	d := NewDialog(nil, "Auto", "", 400, 0, Button("Hello"))
	dt := TestDialog(t, d)
	if err := dt.Click("Hello"); err != nil {
		t.Fatalf("a dialog of height 0 (its natural height) is drawn in no room: %v", err)
	}
}

func TestTestDialogEscapeAsksOnClose(t *testing.T) {
	asked := make(chan struct{}, 4)
	answer := false
	d := NewDialog(nil, "Ask", "", 300, 200, Label("hi"))
	d.OnClose(func() bool { asked <- struct{}{}; return answer })
	dt := TestDialog(t, d)
	dt.Escape()
	flushEvents(t)
	select {
	case <-asked:
	default:
		t.Fatal("Escape did not ask OnClose, as a window's close request does")
	}
	if dt.Closed() {
		t.Fatal("OnClose said no, and the dialog closed")
	}
	answer = true
	dt.Escape()
	flushEvents(t)
	if !dt.Closed() {
		t.Fatal("OnClose said yes, and the dialog stayed open")
	}
}

func TestTestDialogEscapeTakenByAShortcutKeepsTheDialog(t *testing.T) {
	took := 0
	c := Custom(func(c *ui.Context) ui.Element {
		if c.Shortcut(0, ui.KeyEscape) {
			took++
		}
		return ui.Text(c, "static")
	})
	d := NewDialog(nil, "Esc", "", 300, 200, c)
	dt := TestDialog(t, d)
	dt.Escape()
	if dt.Closed() || took != 1 {
		t.Fatalf("closed %v after the content took %d Escapes: a key something routed closes nothing", dt.Closed(), took)
	}
}

func TestTestDialogEscapeClosesWhenTextsChangeForAnotherReason(t *testing.T) {
	frames := 0
	c := Custom(func(c *ui.Context) ui.Element { frames++; return ui.Text(c, fmt.Sprint("frame ", frames)) })
	d := NewDialog(nil, "Esc", "", 300, 200, c)
	dt := TestDialog(t, d)
	dt.Escape()
	if !dt.Closed() {
		t.Fatal("texts changed on their own, and an Escape nothing took kept the dialog open")
	}
}

func TestTestDialogRunsAgain(t *testing.T) {
	started := make(chan struct{}, 2)
	d := NewDialog(nil, "Again", "", 300, 200, Label("hi")).OnBeforeRun(func() { started <- struct{}{} })
	dt := TestDialog(t, d)
	for run := 1; run <= 2; run++ {
		done := make(chan struct{})
		go func() { d.Run(); close(done) }()
		<-started
		if dt.Closed() {
			t.Fatalf("run %d starts with the dialog closed, from the run before", run)
		}
		d.Close()
		<-done
		if !dt.Closed() {
			t.Fatalf("run %d ended with the dialog open", run)
		}
	}
}

func TestTestDialogShowsTheFormsStatusStrip(t *testing.T) {
	d := NewDialog(nil, "S", "", 300, 200, Label("body"))
	d.StatusStrip(StatusStrip(Label("ready")))
	dt := TestDialog(t, d)
	if !dt.HasText("ready") || !dt.HasText("body") {
		t.Fatalf("texts %q: the form's status strip is not drawn", dt.Texts())
	}
}
