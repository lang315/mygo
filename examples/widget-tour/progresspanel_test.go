package main

import (
	"testing"

	"github.com/egoist/mygo/ui/widget"
)

func TestTourProgressPanelAndWaitForm(t *testing.T) {
	f := widget.NewForm("Widget Tour", 1000, 1400)
	f.SetContent(tour(f))
	tt := widget.TestForm(t, f)
	tt.Find(widget.ByLabel("Dialogs")).Click()
	if !tt.Has(widget.ByLabel("An inline ProgressPanel")) {
		t.Fatalf("no inline ProgressPanel: %q", tt.Texts())
	}
	tt.Find(widget.ByLabel("Show wait form")).Click()
	if !tt.Has(widget.ByLabel("This closes in 3 seconds")) {
		t.Fatalf("the wait form does not show: %q", tt.Texts())
	}
	f.CloseWait() // the tour's timer closes it later; the test does not wait for it
	tt.Settle()
	if tt.Has(widget.ByLabel("This closes in 3 seconds")) {
		t.Fatal("CloseWait left the wait form")
	}
}
