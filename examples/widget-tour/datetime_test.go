package main

import (
	"testing"

	"github.com/egoist/mygo/ui/widget"
)

func TestTourShowsDateTimePickerAndMonthCalendar(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1000, 1400)
	tt.Find(widget.ByLabel("Choices")).Click()
	if !tt.Has(widget.ByLabel("hours")) {
		t.Errorf("no time field: %+v", tt.Nodes())
	}
	if !tt.Has(widget.ByLabel("Next month")) {
		t.Error("no month calendar")
	}
}
