package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourAccordionOpensASection(t *testing.T) {
	tt := navigationPage(t)
	tt.Find(widget.ByRole(ui.RoleDisclosure).Label("Privacy")).Click()
	tt.Advance(time.Second) // the sections' animation, on the tester's clock
	tt.Settle()
	if !tt.Has(widget.ByLabel("Open section: 1")) || !tt.Has(widget.ByLabel("Send usage data")) {
		t.Fatalf("a click on Privacy did not open it: %q", tt.Texts())
	}
	if tt.Has(widget.ByLabel("Start at login")) {
		t.Fatalf("General stayed open: %q", tt.Texts())
	}
}
