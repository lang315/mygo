package tw

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui/widget"
)

// edges reads a button's top edge (its middle), its left edge (its middle)
// and its padding, at scale 1.
func edges(t *testing.T, b *widget.ButtonWidget) (top, left, fill color.RGBA) {
	t.Helper()
	tt := widget.NewTester(t, b, 400, 200)
	tt.SetDark(false)
	tt.Frame()
	r := tt.Find(widget.ByLabel("Go")).Bounds()
	img := tt.Image()
	return img.RGBAAt(int(r.X+r.W/2), int(r.Y)), img.RGBAAt(int(r.X), int(r.Y+r.H/2)), img.RGBAAt(int(r.X)+6, int(r.Y+r.H/2))
}

// Case 6: an applied tw border class loses to Suggested's own border, as an
// applied background does; border-l-0 still leaves no left border.
func TestSuggestedOwnBorderBeatsTwBorderClasses(t *testing.T) {
	top, left, fill := edges(t, widget.Button("Go").Suggested().Apply(S("border-red-500 border-l-0 rounded-l-none")))
	if top != fill {
		t.Errorf("top edge %v, want the fill %v: the own border wins", top, fill)
	}
	if left != fill {
		t.Errorf("left edge %v with border-l-0, want the fill %v", left, fill)
	}
}

// tw's suggested-action and destructive-action draw the border in the fill
// too, and the login form's classes leave no left border.
func TestTwActionsDrawTheBorderInTheFill(t *testing.T) {
	for _, class := range []string{"suggested-action", "destructive-action", "suggested-action rounded-l-none border-l-0"} {
		top, left, fill := edges(t, widget.Button("Go").Apply(S(class)))
		if top != fill {
			t.Errorf("%q: top edge %v, want the fill %v", class, top, fill)
		}
		if left != fill {
			t.Errorf("%q: left edge %v, want the fill %v", class, left, fill)
		}
	}
}
