package widget

import (
	"fmt"
	"image/color"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func pbBounds(tt *Tester) []string {
	var s []string
	for _, n := range tt.Nodes() {
		s = append(s, fmt.Sprintf("%d %q %q %v", n.Role, n.Label, n.Value, n.Bounds))
	}
	return s
}

func pbIsRed(c color.RGBA) bool { return c.R > 180 && c.G < 90 && c.B < 90 }

func pbIn(r ui.Rect, x, y int) bool {
	fx, fy := float32(x)+0.5, float32(y)+0.5
	return fx >= r.X && fx < r.X+r.W && fy >= r.Y && fy < r.Y+r.H
}

// red pixels outside every rect in rs, and inside any.
func pbRed(tt *Tester, rs []ui.Rect) (out, in int) {
	img := tt.Image()
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if !pbIsRed(color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)) {
				continue
			}
			hit := false
			for _, r := range rs {
				if pbIn(r, x, y) {
					hit = true
				}
			}
			if hit {
				in++
			} else {
				out++
			}
		}
	}
	return
}

func pbFieldBounds(tt *Tester, prefix string) []ui.Rect {
	var rs []ui.Rect
	for _, n := range tt.Nodes() {
		if strings.HasPrefix(n.Value, prefix) && n.Bounds.W > 0 {
			rs = append(rs, n.Bounds)
		}
	}
	return rs
}

// Showing the mark moves nothing, in a row, a scroll panel and right to left,
// and stays inside the control (also when it is clipped).
func TestErrorMarkShiftsNothingAndStaysInsideTheControl(t *testing.T) {
	type cs struct {
		name string
		w, h int
		rtl  bool
		root func() (Control, []Control)
	}
	cases := []cs{
		{"fields", 400, 200, false, func() (Control, []Control) {
			a, e, c := TextBox().SetText("aa"), TextBox().SetText("ERR"), TextBox().SetText("cc")
			return Fields().Row("Name", a).Row("Email", e).Row("Age", c), []Control{e}
		}},
		{"hbox", 400, 100, false, func() (Control, []Control) {
			e := TextBox().SetText("ERR")
			return HBox(8).Add(TextBox().SetText("aa"), e, Button("Go")), []Control{e}
		}},
		{"grid", 400, 200, false, func() (Control, []Control) {
			e := TextBox().SetText("ERR")
			return Grid(4, 4).Attach(TextBox().SetText("a"), 0, 0, 1, 1).Attach(e, 1, 0, 1, 1).Attach(Button("b"), 0, 1, 2, 1), []Control{e}
		}},
		{"scroll-clip", 400, 200, false, func() (Control, []Control) {
			v := VBox(8)
			var es []Control
			for i := 0; i < 12; i++ {
				e := TextBox().SetText(fmt.Sprintf("ERR%02d", i))
				es = append(es, e)
				v.Add(e)
			}
			return VBox(0).Add(ScrollPanel(v).SetSize(400, 100), Button("Below")), es
		}},
		{"right-edge", 300, 80, false, func() (Control, []Control) {
			e := TextBox().SetText("ERR").HExpand()
			return HBox(0).Add(e), []Control{e}
		}},
		{"rtl", 400, 100, true, func() (Control, []Control) {
			e := TextBox().SetText("ERR")
			return HBox(8).Add(TextBox().SetText("aa"), e), []Control{e}
		}},
	}
	for _, k := range cases {
		root, es := k.root()
		tt := NewTester(t, root, k.w, k.h)
		if k.rtl {
			tt.SetDirection(ui.RightToLeft)
		}
		tt.Settle()
		before := pbBounds(tt)
		ep := ErrorProvider()
		for _, e := range es {
			ep.SetError(e, "bad")
		}
		tt.Settle()
		after := pbBounds(tt)
		rs := pbFieldBounds(tt, "ERR")
		out, in := pbRed(tt, rs)
		side := ""
		if k.rtl && len(rs) == 1 {
			r := rs[0]
			_, l := pbRed(tt, []ui.Rect{{X: r.X, Y: r.Y, W: r.W / 2, H: r.H}})
			side = fmt.Sprintf(" leftHalfRed=%d", l)
		}
		_ = side
		if !slices.Equal(before, after) || out != 0 || in == 0 {
			t.Errorf("%s: shift or mark outside", k.name)
		}
	}
}
