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

// BenchmarkUIFrame is the reference for BenchmarkWidgetFrame: the same
// items built with package ui alone, the look set by ui calls inline as
// Style's props.apply makes them. What the widget layer adds to the cost of
// styling is (widget-styled / widget-plain) / (ui-styled / ui-plain).
func BenchmarkUIFrame(b *testing.B) {
	for _, n := range []int{100, 1000} {
		for _, c := range []struct {
			suffix       string
			styled, fade bool
		}{
			{"", false, false},
			{"-styled-no-transition", true, false},
			{"-styled", true, true},
		} {
			b.Run(strconv.Itoa(n)+c.suffix, func(b *testing.B) {
				texts := make([]string, n)
				for i := range texts {
					texts[i] = fmt.Sprintf("Text %d", i)
				}
				look := func(c2 *ui.Context, e ui.Element) {
					if !c.styled {
						return
					}
					t := c2.Theme()
					sp := t.Space(1)
					e.Padding(sp, sp, sp, sp)
					e.Radius(sp, sp, sp, sp)
					e.Shadow(0, 1, 2, 0, ui.RGBA(0, 0, 0, 0.1))
					bg := t.Surface
					if e.Hovered() {
						bg = t.SurfaceHover
					}
					if e.Pressed() {
						bg = t.SurfacePressed
					}
					e.Background(bg)
					if c.fade {
						e.Transition(ui.ElementTransition{Colors: true, Duration: 120 * time.Millisecond})
					}
				}
				tt := ui.NewTester(func(c2 *ui.Context) {
					ui.Column(c2).Fill().Children(func() {
						ui.ScrollBoth(c2).Grow(1).Children(func() {
							ui.Column(c2).Gap(2).Children(func() {
								for i := range n {
									switch i % 3 {
									case 0:
										look(c2, ui.Text(c2, "Label "+strconv.Itoa(i)))
									case 1:
										look(c2, ui.Button(c2, "Button "+strconv.Itoa(i)))
									default:
										look(c2, ui.TextInput(c2, &texts[i]))
									}
								}
							})
						})
					})
				}, 1280, 800)
				for range 10 {
					tt.Frame()
				}
				if r, ok := tt.Find("Label 0"); ok {
					tt.Move(r.X+r.W/2, r.Y+r.H/2)
				}
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

// widgetKinds make a control of each kind a form is made of. own is what
// package ui itself allocates for a control of the kind in each frame (a
// text input, a styled button, a check box's painting): the widget layer adds
// nothing to it.
var widgetKinds = []struct {
	name string
	own  int
	make func(i int) Control
}{
	{"Label", 0, func(i int) Control { return Label(fmt.Sprintf("Label %d", i)) }},
	{"Button", 1, func(i int) Control { return Button(fmt.Sprintf("Button %d", i)).OnClick(func() {}) }},
	{"TextBox", 1, func(i int) Control { return TextBox().SetText(fmt.Sprintf("Text %d", i)) }},
	{"MultilineTextBox", 1, func(i int) Control { return MultilineTextBox().SetText(fmt.Sprintf("Text %d", i)) }},
	{"MaskedTextBox", 1, func(i int) Control { return MaskedTextBox("000-0000") }},
	{"CheckBox", 2, func(i int) Control { return CheckBox(fmt.Sprintf("Check %d", i)) }},
	{"Toggle", 1, func(i int) Control { return Toggle(fmt.Sprintf("Toggle %d", i)) }},
	{"Switch", 1, func(i int) Control { return Switch() }},
	{"NumericUpDown", 8, func(i int) Control { return NumericUpDown(0, 100, 1) }},
	{"TrackBar", 1, func(i int) Control { return TrackBar(0, 100) }},
}

// frameAllocs returns the allocations of a frame of n controls of a kind.
func frameAllocs(make func(i int) Control, n int) float64 {
	box := VBox(2)
	for i := range n {
		box.Add(make(i))
	}
	tt := ui.NewTester(content(ScrollPanel(box)), 1280, 800)
	for range 10 {
		tt.Frame()
	}
	return testing.AllocsPerRun(30, tt.Frame)
}

// TestWidgetFrameAllocatesNoClosuresPerWidget keeps the widget layer from
// allocating for each widget in each frame (a key boxed, a label parsed for
// its mnemonic, a callback made again). It measures the slope, the
// allocations of 200 controls less those of 100, per control: what is fixed
// in a frame, which differs between platforms, is not in it. What is left is
// ui's own, as the table of kinds says.
func TestWidgetFrameAllocatesNoClosuresPerWidget(t *testing.T) {
	for _, k := range widgetKinds {
		t.Run(k.name, func(t *testing.T) {
			slope := (frameAllocs(k.make, 200) - frameAllocs(k.make, 100)) / 100
			if slope > float64(k.own)+0.2 {
				t.Fatalf("%.2f allocations per control in a frame, want %d (ui's own)", slope, k.own)
			}
		})
	}
}
