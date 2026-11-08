package ui

import (
	"image"
	"testing"
	"time"
)

// A disabled DateInput is drawn as a disabled text input is, in both
// appearances: the same border and fill at its edge, and hovering it
// changes nothing.
func TestDisabledDateInputLooksLikeADisabledTextInput(t *testing.T) {
	for _, dark := range []bool{false, true} {
		th := LightTheme()
		if dark {
			th = DarkTheme()
		}
		var text string
		date := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
		tt := coreNewTester(func(c *context) {
			c.SetTheme(th)
			coreColumn(c).Gap(10).Padding(20).Children(func() {
				coreTextInput(c, &text).Width(200).Label("Text").Disabled(true)
				coreDateInput(c, &date).Label("Date").Disabled(true)
			})
		}, 400, 200)
		tt.Frame()
		// edge returns the first two pixels of a control's left edge, the
		// border and the fill inside it, at the middle of its height.
		edge := func(img *image.RGBA, label string) [2]Color {
			t.Helper()
			for _, n := range tt.Nodes() {
				if n.Label != label || n.Bounds.W == 0 || n.Role == RoleText {
					continue
				}
				y := int(n.Bounds.Y + n.Bounds.H/2)
				bg := img.RGBAAt(2, 198)
				for x := int(n.Bounds.X) - 16; x <= int(n.Bounds.X)+2; x++ {
					if p := img.RGBAAt(x, y); p != bg {
						a, b := img.RGBAAt(x, y), img.RGBAAt(x+2, y)
						return [2]Color{{R: a.R, G: a.G, B: a.B}, {R: b.R, G: b.G, B: b.B}}
					}
				}
			}
			t.Fatalf("dark=%v: no edge of %q", dark, label)
			return [2]Color{}
		}
		img := tt.Image()
		text0, date0 := edge(img, "Text"), edge(img, "Date")
		if text0 != date0 {
			t.Errorf("dark=%v: a disabled DateInput has border and fill %v, a disabled TextInput %v", dark, date0, text0)
		}
		before := *img
		before.Pix = append([]byte(nil), img.Pix...)
		r, ok := tt.Find("Date")
		if !ok {
			// the label names the control, not its text: hover its node
			for _, n := range tt.Nodes() {
				if n.Label == "Date" && n.Role != RoleText {
					r = Rect{X: n.Bounds.X, Y: n.Bounds.Y, W: n.Bounds.W, H: n.Bounds.H}
				}
			}
		}
		tt.Move(r.X+r.W/2, r.Y+r.H/2)
		tt.Frame()
		after := tt.Image()
		if string(after.Pix) != string(before.Pix) {
			t.Errorf("dark=%v: hovering a disabled DateInput changed what is drawn", dark)
		}
	}
}
