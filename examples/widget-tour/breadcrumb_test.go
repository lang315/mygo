package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourBreadCrumbTrimsThePath(t *testing.T) {
	tt := navigationPage(t)
	tt.Find(widget.ByLabel("Users").Role(ui.RoleLink)).Click()
	if !tt.Has(widget.ByLabel("At C:/Users")) || tt.Has(widget.ByLabel("ada")) {
		t.Fatalf("a click on Users did not trim the path: %q", tt.Texts())
	}
}
