package widget

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// BenchmarkWidgetFrame builds a frame of n plain widgets in a scrolled
// box, a third each labels, buttons and text boxes, as a long form does.
// It is the baseline the Style work (roadmap 0.5) must stay within 10% of.
//
// The -styled cases give every widget one shared Style with a background
// role, a radius, a shadow, and Hover and Pressed variants; -styled adds a
// transition. In every case the pointer rests on the first widget, so the
// styled ones show their hover.
func BenchmarkWidgetFrame(b *testing.B) {
	look := NewStyle().Padding(1).Radius(1).Background(RoleSurface).
		Shadow(0, 1, 2, 0, Fixed(ui.RGBA(0, 0, 0, 0.1))).
		Hover(NewStyle().Background(RoleSurfaceHover)).
		Pressed(NewStyle().Background(RoleSurfacePressed))
	fading := look.Transition(120 * time.Millisecond)
	for _, n := range []int{100, 1000} {
		for _, c := range []struct {
			suffix string
			style  *Style
		}{
			{"", nil},
			{"-styled-no-transition", &look},
			{"-styled", &fading},
		} {
			style := c.style
			b.Run(strconv.Itoa(n)+c.suffix, func(b *testing.B) {
				box := VBox(2)
				for i := range n {
					var w Control
					switch i % 3 {
					case 0:
						w = Label(fmt.Sprintf("Label %d", i))
					case 1:
						w = Button(fmt.Sprintf("Button %d", i))
					default:
						w = TextBox().SetText(fmt.Sprintf("Text %d", i))
					}
					if style != nil {
						switch w := w.(type) {
						case *LabelWidget:
							w.Apply(*style)
						case *ButtonWidget:
							w.Apply(*style)
						case *TextBoxWidget:
							w.Apply(*style)
						}
					}
					box.Add(w)
				}
				tt := ui.NewTester(content(ScrollPanel(box)), 1280, 800)
				for range 10 {
					tt.Frame()
				}
				if r, ok := tt.Find("Label 0"); ok {
					tt.Move(r.X+r.W/2, r.Y+r.H/2)
				}
				// The tester's clock stands still while the loop runs: let
				// the hover's transition finish, so the pointer rests.
				tt.Advance(time.Second)
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					tt.Frame()
				}
			})
		}
	}
}
