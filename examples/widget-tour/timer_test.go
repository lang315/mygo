package main

import (
	"testing"

	"github.com/egoist/mygo/ui/widget"
)

func TestTourTimerStartsAndStops(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 1400)
	tt.Find(widget.ByLabel("Extras")).Click()
	tt.Find(widget.ByLabel("Start timer")).Click()
	if !tt.Has(widget.ByLabel("Stop timer")) {
		t.Fatalf("the button did not turn to Stop: %q", tt.Texts())
	}
	tt.Find(widget.ByLabel("Stop timer")).Click() // so no timer outlives the test
	if !tt.Has(widget.ByLabel("Start timer")) {
		t.Fatalf("texts %q", tt.Texts())
	}
}
