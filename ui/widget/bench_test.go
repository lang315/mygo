package widget

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
)

// BenchmarkWidgetFrame builds a frame of n plain widgets in a scrolled
// box, a third each labels, buttons and text boxes, as a long form does.
// It is the baseline the Style work (roadmap 0.5) must stay within 10% of.
func BenchmarkWidgetFrame(b *testing.B) {
	for _, n := range []int{100, 1000} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			box := VBox(2)
			for i := range n {
				switch i % 3 {
				case 0:
					box.Add(Label(fmt.Sprintf("Label %d", i)))
				case 1:
					box.Add(Button(fmt.Sprintf("Button %d", i)))
				default:
					box.Add(Entry().SetText(fmt.Sprintf("Text %d", i)))
				}
			}
			tt := ui.NewTester(content(Scrolled(box)), 1280, 800)
			for range 10 {
				tt.Frame()
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				tt.Frame()
			}
		})
	}
}
