package main

import (
	"image/color"
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

// The email's rules: text that is not an email fails the regexp rule, and
// leaving the field marks it before any button is pressed.
func TestTourEmailRules(t *testing.T) {
	f := widget.NewForm("tour", 1100, 1800)
	f.SetContent(tour(f))
	tt := widget.TestForm(t, f)
	tt.Find(widget.ByLabel("Inputs")).Click()
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
	box := email.Bounds()
	before := redPixels(tt, box)
	email.Focus().Type("ann@example")
	tt.Find(widget.ByLabel("Validate")).Focus() // the focus leaves the email
	tt.Settle()
	if redPixels(tt, box) <= before {
		t.Error("leaving the email with no domain did not mark it")
	}
	tt.Find(widget.ByLabel("Validate")).Click()
	if !tt.Has(widget.ByLabel("Not valid")) {
		t.Fatalf("an email with no domain validated: %q", tt.Texts())
	}
	email.Focus().Type(".com")
	tt.Find(widget.ByLabel("Validate")).Click()
	if !tt.Has(widget.ByLabel("Valid")) || redPixels(tt, box) > before {
		t.Fatalf("a good email did not validate, or kept its mark: %q", tt.Texts())
	}
}

// redPixels counts the pixels of a clearly red hue inside r of the frame.
func redPixels(tt *widget.Tester, r ui.Rect) int {
	img := tt.Image()
	n := 0
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
			if c.R > 180 && c.G < 90 && c.B < 90 {
				n++
			}
		}
	}
	return n
}
