package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourTextBoxSuggestsCities(t *testing.T) {
	tt := navigationPage(t)
	tt.Find(widget.ByLabel("City")).Type("Ha")
	if !tt.HasText("Hanoi") {
		t.Fatalf("typing Ha did not suggest Hanoi: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	tt.Settle()
	if v := tt.Find(widget.ByLabel("City")).Value(); v != "Hanoi" {
		t.Fatalf("Down, Enter took %q, want Hanoi", v)
	}
}
