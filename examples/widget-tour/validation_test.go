package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

func TestTourValidatesAnEmail(t *testing.T) {
	f := widget.NewForm("tour", 1100, 1800)
	f.SetContent(tour(f))
	tt := widget.TestForm(t, f)
	tt.Find(widget.ByLabel("Inputs")).Click()
	tt.Find(widget.ByLabel("Validate")).Click()
	if !tt.Has(widget.ByLabel("Not valid")) {
		t.Fatalf("an empty email validated: %q", tt.Texts())
	}
	var email *widget.Node // the text field in the row of the button, before it
	btn := tt.Find(widget.ByLabel("Validate")).Bounds()
	for _, n := range tt.FindAll(widget.ByRole(ui.RoleTextField)) {
		if b := n.Bounds(); b.X < btn.X && b.Y < btn.Y+btn.H && b.Y+b.H > btn.Y {
			email = n
		}
	}
	if email == nil {
		t.Fatalf("no email box among %q", tt.Texts())
	}
	email.Focus().Type("ann@example.com")
	tt.Find(widget.ByLabel("Validate")).Click()
	if !tt.Has(widget.ByLabel("Valid")) {
		t.Fatalf("a good email did not validate: %q", tt.Texts())
	}
}
