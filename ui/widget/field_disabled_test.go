package widget

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// The date part of a disabled DateTimePicker is drawn as a disabled TextBox
// is, in both appearances, and hovering it changes nothing.
func TestDisabledDateTimePickerLooksLikeADisabledTextBox(t *testing.T) {
	for _, dark := range []bool{false, true} {
		sample := func(w Control, role ui.Role) (edge [3]ui.Color, hoverChanged bool) {
			tt := tester(t, w)
			tt.SetDark(dark)
			tt.Frame()
			for _, n := range tt.Nodes() {
				if n.Role != role || n.Bounds.W == 0 {
					continue
				}
				img := tt.Image()
				y := int(n.Bounds.Y + n.Bounds.H/2)
				for i, dx := range []int{0, 1, 4} {
					p := img.RGBAAt(int(n.Bounds.X)+dx, y)
					edge[i] = ui.Color{R: p.R, G: p.G, B: p.B}
				}
				before := append([]byte(nil), img.Pix...)
				tt.Move(n.Bounds.X+n.Bounds.W/2, n.Bounds.Y+n.Bounds.H/2)
				tt.Frame()
				return edge, string(tt.Image().Pix) != string(before)
			}
			t.Fatalf("dark=%v: no node with role %v", dark, role)
			return
		}
		textEdge, textHover := sample(TextBox().SetText("2026-01-05").SetDisabled(true), ui.RoleTextField)
		date := DateTimePicker().SetFormat(DateOnly).SetValue(time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)).SetDisabled(true)
		dateEdge, dateHover := sample(date, ui.RolePopUpButton)
		if dateEdge != textEdge {
			t.Errorf("dark=%v: the date field's edge is %v, a disabled TextBox's %v", dark, dateEdge, textEdge)
		}
		if textHover || dateHover {
			t.Errorf("dark=%v: hovering changed what is drawn (TextBox %v, date %v)", dark, textHover, dateHover)
		}
	}
}
