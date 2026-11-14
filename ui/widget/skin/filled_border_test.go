package skin

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// calmButton shows w in Calm, light, with a probe that keeps the theme of
// the last frame in *th.
func calmButton(t *testing.T, w widget.Control, th *ui.Theme) *widget.Tester {
	t.Helper()
	t.Cleanup(func() { Use(nil) })
	probe := widget.Custom(func(c *ui.Context) ui.Element {
		*th = *c.Theme()
		return ui.Text(c, "probe")
	})
	tt := widget.NewTester(t, widget.VBox(8).Add(w, probe), 400, 200)
	Use(Calm)
	tt.SetDark(false)
	tt.Frame()
	return tt
}

// outline is a button's outline row, its fill and its top left corner, at
// scale 1.
type outline struct {
	row    []color.RGBA // the top row, edge to edge
	border color.RGBA   // its middle
	fill   color.RGBA   // the padding left of the text
}

func outlineOf(t *testing.T, tt *widget.Tester, label string) outline {
	t.Helper()
	b := tt.Find(widget.ByLabel(label)).Bounds()
	img := tt.Image()
	var o outline
	for x := int(b.X); x < int(b.X+b.W); x++ {
		o.row = append(o.row, img.RGBAAt(x, int(b.Y)))
	}
	o.border = img.RGBAAt(int(b.X+b.W/2), int(b.Y))
	o.fill = img.RGBAAt(int(b.X)+6, int(b.Y+b.H/2))
	return o
}

func states(t *testing.T, tt *widget.Tester, label string) (rest, hover, pressed outline) {
	t.Helper()
	rest = outlineOf(t, tt, label)
	b := tt.Find(widget.ByLabel(label)).Bounds()
	x, y := b.X+b.W/2, b.Y+b.H/2
	tt.Move(x, y)
	hover = outlineOf(t, tt, label)
	tt.Press(x, y)
	pressed = outlineOf(t, tt, label)
	return rest, hover, pressed
}

func isColor(c color.RGBA, want ui.Color) bool {
	d := func(a, b uint8) bool { x := int(a) - int(b); return x > -3 && x < 3 }
	return d(c.R, want.R) && d(c.G, want.G) && d(c.B, want.B)
}

// Cases 1 and 2 under Calm: border = fill = the accent's Base, hover and pressed.
func TestCalmSuggestedBorderIsItsFill(t *testing.T) {
	var th ui.Theme
	tt := calmButton(t, widget.Button("Start").Suggested(), &th)
	rest, hover, pressed := states(t, tt, "Start")
	for _, c := range []struct {
		state string
		o     outline
		want  ui.Color
	}{{"rest", rest, ui.Hex("#1c71d8")}, {"hover", hover, th.AccentHover}, {"pressed", pressed, th.AccentPressed}} {
		if c.o.border != c.o.fill {
			t.Errorf("%s: border %v, fill %v: want the same", c.state, c.o.border, c.o.fill)
		}
		if !isColor(c.o.border, c.want) {
			t.Errorf("%s: border %v, want %v", c.state, c.o.border, c.want)
		}
	}
}

// Case 3 under Calm.
func TestCalmDestructiveBorderIsItsFill(t *testing.T) {
	var th ui.Theme
	tt := calmButton(t, widget.Button("Delete").Destructive(), &th)
	rest, hover, pressed := states(t, tt, "Delete")
	for state, o := range map[string]outline{"rest": rest, "hover": hover, "pressed": pressed} {
		if o.border != o.fill {
			t.Errorf("%s: border %v, fill %v: want the same", state, o.border, o.fill)
		}
		if isColor(o.border, th.ButtonBorder) {
			t.Errorf("%s: border %v is the plain gray", state, o.border)
		}
	}
}

// Case 5 under Calm: the same size as a plain button.
func TestCalmFilledBorderKeepsTheSize(t *testing.T) {
	size := func(b *widget.ButtonWidget) (w, h float32) {
		var th ui.Theme
		r := calmButton(t, b, &th).Find(widget.ByLabel("Go")).Bounds()
		return r.W, r.H
	}
	pw, ph := size(widget.Button("Go"))
	for name, b := range map[string]*widget.ButtonWidget{"suggested": widget.Button("Go").Suggested(), "destructive": widget.Button("Go").Destructive()} {
		if w, h := size(b); w != pw || h != ph {
			t.Errorf("%s %v x %v, plain %v x %v", name, w, h, pw, ph)
		}
	}
}

// Case 7: the outline row of a Suggested button has the fill at its middle and
// no pixel of the plain gray.
func TestCalmSuggestedOutlineRowHasNoGray(t *testing.T) {
	var th ui.Theme
	tt := calmButton(t, widget.Button("Start").Suggested(), &th)
	o := outlineOf(t, tt, "Start")
	if o.border != o.fill || !isColor(o.border, ui.Hex("#1c71d8")) {
		t.Errorf("middle of the top edge %v, fill %v, want #1c71d8", o.border, o.fill)
	}
	for x, c := range o.row {
		if isColor(c, th.ButtonBorder) {
			t.Errorf("pixel %d of the outline row is the gray %v", x, c)
			break
		}
	}
}

// Case 6 under Calm: a later own call wins, an applied color loses.
func TestCalmFilledBorderIsTheOwnStyle(t *testing.T) {
	red := ui.RGB(255, 0, 0)
	accent := ui.Hex("#1c71d8")
	for name, c := range map[string]struct {
		b    *widget.ButtonWidget
		want ui.Color
	}{
		"Border":              {widget.Button("Go").Suggested().Border(1, red), red},
		"applied BorderColor": {widget.Button("Go").Suggested().Apply(widget.NewStyle().BorderColor(widget.Fixed(red))), accent},
	} {
		var th ui.Theme
		if o := outlineOf(t, calmButton(t, c.b, &th), "Go"); !isColor(o.border, c.want) {
			t.Errorf("%s: border %v, want %v", name, o.border, c.want)
		}
	}
}

// Case 8 under Calm: a tool strip's Suggested and Destructive tool buttons.
func TestCalmToolButtonFilledBorder(t *testing.T) {
	for name, b := range map[string]*widget.ToolButtonWidget{
		"suggested":   widget.ToolButton("Go").Suggested(),
		"destructive": widget.ToolButton("Go").Destructive(),
	} {
		var th ui.Theme
		tt := calmButton(t, widget.ToolStrip(b), &th)
		rest, hover, pressed := states(t, tt, "Go")
		for state, o := range map[string]outline{"rest": rest, "hover": hover, "pressed": pressed} {
			if o.border != o.fill {
				t.Errorf("%s %s: border %v, fill %v: want the same", name, state, o.border, o.fill)
			}
		}
	}
}
