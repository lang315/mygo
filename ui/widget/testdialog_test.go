package widget

import "testing"

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
	dt := TestDialog(d.Dialog)
	got := make(chan string, 1)
	go func() { got <- d.run() }()
	focus(dt.Tester, 1)
	dt.Type("Ada")
	dt.Frame()
	if err := dt.Click("OK"); err != nil {
		t.Fatal(err)
	}
	flushEvents(t)
	if v := <-got; v != "Ada" {
		t.Fatalf("Run returned %q", v)
	}
	if !dt.Closed() {
		t.Fatal("not closed")
	}
}

func TestTestDialogCancelAndEscapeClose(t *testing.T) {
	d := newNameDialog()
	dt := TestDialog(d.Dialog)
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
	dt2 := TestDialog(d2.Dialog)
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
	dt := TestDialog(d)
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
	dt := TestDialog(d)
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
	dt := TestDialog(d)
	if err := dt.Click("Hello"); err != nil {
		t.Fatalf("a dialog of height 0 (its natural height) is drawn in no room: %v", err)
	}
}
