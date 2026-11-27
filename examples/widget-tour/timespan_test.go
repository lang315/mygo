package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourTimeSpanEdit(t *testing.T) {
	tt := widget.NewTester(t, tour(nil), 1100, 1800)
	tt.Find(widget.ByLabel("Choices")).Click()
	var in *widget.Node
	for _, n := range tt.FindAll(widget.ByRole(ui.RoleTextField)) {
		if n.Value() == "00:01:30" {
			in = n
		}
	}
	if in == nil {
		t.Fatalf("no field shows 00:01:30 on the Choices tab: %q", tt.Texts())
	}
	in.Focus()
	in.Key(0, ui.KeyEnd) // the seconds
	in.Key(0, ui.KeyUp)
	if !tt.Has(widget.ByLabel("Timeout 1m31s")) {
		t.Fatalf("after Up on the seconds: %q", tt.Texts())
	}
	if got := in.Value(); got != "00:01:31" {
		t.Fatalf("after Up the field shows %q", got)
	}
}
