package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// The Navigation page's ribbon shows a gallery of styles, three at a time;
// its drop-down holds all six, and a pick shows in the label beside.
func TestTourRibbonGallery(t *testing.T) {
	tt := navigationPage(t)
	if !tt.Has(widget.ByLabel("Styles").Role(ui.RoleGroup)) {
		t.Fatalf("no gallery of styles: %q", tt.Texts())
	}
	if !tt.Has(widget.ByLabel("Heading 2")) || tt.Has(widget.ByLabel("Title")) {
		t.Fatalf("the gallery does not show its first row of three: %q", tt.Texts())
	}
	tt.Find(widget.ByLabel("More").Role(ui.RoleButton)).Click()
	titles := tt.FindAll(widget.ByLabel("Title"))
	if len(titles) != 1 || !tt.Has(widget.ByLabel("Code")) {
		t.Fatalf("the drop-down does not hold all six styles: %q", tt.Texts())
	}
	titles[0].Click()
	tt.Settle()
	if !tt.Has(widget.ByLabel("Style: Title")) {
		t.Fatalf("picking Title does not show in the label: %q", tt.Texts())
	}
}
