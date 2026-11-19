package skin

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/widget"
)

// roleSentinel is Calm with roles of text no plain theme or Calm has.
func roleSentinel() *Skin {
	l, d := Calm.Tokens(false), Calm.Tokens(true)
	for _, tk := range []*Tokens{&l, &d} {
		tk.Label = Type{Size: 11.5, Weight: 800}
		tk.Caption = Type{Size: 9.5, Weight: 400}
		tk.Heading = Type{Size: 17.5, Weight: 700}
		tk.Title = Type{Size: 23.5, Weight: 700}
	}
	return New("Roles", l, d)
}

// textStyle is the font size and weight the text label draws with, as the
// inspector shows them (the size to one decimal): its own, else the nearest
// ancestor's; -1 where none is set.
func textStyle(t *testing.T, tt *widget.Tester, label string) (size float32, weight int) {
	t.Helper()
	if err := tt.Inspect(label); err != nil {
		t.Fatal(err)
	}
	texts := tt.Texts()
	i := slices.Index(texts, "Styles")
	if i < 0 {
		t.Fatalf("the inspector shows no styles: %q", texts)
	}
	size, weight = -1, -1
	for _, s := range texts[i:] {
		if v, ok := strings.CutPrefix(s, "Value of font-size: "); ok && size < 0 {
			f, err := strconv.ParseFloat(strings.TrimSuffix(v, "px"), 32)
			if err != nil {
				t.Fatalf("font-size %q: %v", v, err)
			}
			size = float32(f)
		}
		if v, ok := strings.CutPrefix(s, "Value of font-weight: "); ok && weight < 0 {
			weight, _ = strconv.Atoi(v)
		}
	}
	return size, weight
}

// near says whether the inspector's size, to one decimal, is want.
func near(got, want float32) bool { return got-want < 0.06 && want-got < 0.06 }

type roleRow struct{ ID, Name string }

// roleCase is a control whose text takes a role: what shows it, and the
// text to inspect.
type roleCase struct {
	name, text string
	show       func(t *testing.T) *widget.Tester
}

func roleCases(s *Skin, dark bool, scale float32) []roleCase {
	in := func(w widget.Control) func(t *testing.T) *widget.Tester {
		return func(t *testing.T) *widget.Tester { return showSkin(t, w, 700, 500, s, dark, scale) }
	}
	return []roleCase{
		{"GroupBox", "Group title", in(widget.GroupBox("Group title").Add(widget.Label("inside")))},
		{"DataGridView", "Name", in(widget.DataGridView(func(r *roleRow) string { return r.ID },
			widget.Column[roleRow]{Title: "Name", Text: func(r *roleRow) string { return r.Name }}).
			SetRows([]roleRow{{"1", "Ada"}}).SetSize(0, 200))},
		{"StatusLabel", "Ready", in(widget.StatusLabel("Ready"))},
		{"Fields", "Section head", in(widget.Fields().Section("Section head").Row("Field", widget.TextBox()))},
		{"Dialog", "Dialog title", func(t *testing.T) *widget.Tester {
			d := widget.NewDialog(nil, "Dialog title", "", 500, 400, widget.Label("body"))
			t.Cleanup(func() { Use(nil) })
			tt := widget.TestForm(t, d.Form)
			Use(s)
			tt.SetDark(dark)
			tt.SetPreferences(ui.Preferences{TextScale: scale})
			tt.Frame()
			return tt
		}},
	}
}

// The titles of group boxes and dialogs, grid headers, status labels and
// form headings take the skin's roles of text, at the text scale; without a
// skin they keep their plain look.
func TestRolesTakeTheSkin(t *testing.T) {
	want := map[string]Type{
		"GroupBox": {Size: 11.5, Weight: 800}, "DataGridView": {Size: 11.5}, "StatusLabel": {Size: 9.5, Weight: 400},
		"Fields": {Size: 17.5, Weight: 700}, "Dialog": {Size: 23.5, Weight: 700},
	}
	for _, dark := range []bool{false, true} {
		for _, scale := range []float32{1, 1.5} {
			for _, c := range roleCases(roleSentinel(), dark, scale) {
				t.Run(fmt.Sprintf("%s-%s-%g", c.name, appearance(dark), scale), func(t *testing.T) {
					tt := c.show(t)
					size, weight := textStyle(t, tt, c.text)
					w := want[c.name]
					if !near(size, w.Size*scale) {
						t.Errorf("the text is %v, want %v", size, w.Size*scale)
					}
					// ui sets a header's weight, which the skin does not reach.
					if w.Weight != 0 && weight != w.Weight {
						t.Errorf("the weight is %d, want %d", weight, w.Weight)
					}
				})
			}
		}
	}
}

// Without a skin, after one: the plain look, a dialog's title LG and bold.
func TestRolesPlainAfterUseNil(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		body := ui.LightTheme().FontSize * scale
		// The theme's size, body, which the window's root sets; -1 is no weight set.
		plain := map[string]Type{
			"GroupBox": {Size: body, Weight: 700}, "DataGridView": {Size: body, Weight: 600}, "StatusLabel": {Size: body, Weight: -1},
			"Fields": {Size: body, Weight: 700}, "Dialog": {Size: body * 1.125, Weight: 700},
		}
		for _, c := range roleCases(roleSentinel(), false, scale) {
			t.Run(fmt.Sprintf("%s-%g", c.name, scale), func(t *testing.T) {
				tt := c.show(t)
				Use(nil)
				tt.Frame()
				size, weight := textStyle(t, tt, c.text)
				w := plain[c.name]
				if !near(size, w.Size) {
					t.Errorf("the text is %v, want %v", size, w.Size)
				}
				if weight != w.Weight {
					t.Errorf("the weight is %d, want %d", weight, w.Weight)
				}
			})
		}
	}
}

// Without a skin, a form's heading keeps the size of the box it is in, as
// before roles: only its weight is its own.
func TestPlainHeadingTakesTheBoxSize(t *testing.T) {
	box := widget.VBox(0).Add(widget.Fields().Section("Section head").Row("Field", widget.TextBox())).TextSize(widget.XL)
	tt := showSkin(t, box, 600, 400, nil, false, 1)
	size, weight := textStyle(t, tt, "Section head")
	if want := ui.LightTheme().FontSize * 1.25; !near(size, want) || weight != 700 {
		t.Errorf("the heading is %v and %d, want the box's XL, %v, and 700", size, weight, want)
	}
}

// A status label's icon keeps the skin's Icon side, at the text scale, while
// its text takes the Caption size: the icon does not shrink to the caption.
func TestStatusLabelIconKeepsIconSide(t *testing.T) {
	square := ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><rect width="16" height="16" fill="currentColor"/></svg>`))
	green := ui.Color{R: iconGreen.R, G: iconGreen.G, B: iconGreen.B, A: 255}
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			label := widget.StatusLabel("Ready").SetIcon(square).Apply(widget.NewStyle().TextColor(widget.Fixed(green)))
			tt := showSkin(t, label, 600, 400, Calm, false, scale)
			if got, want := iconRun(tt, ui.Rect{W: 600, H: 400}), 16*scale; !sideOf(got, want) {
				t.Errorf("the icon is %v wide, want %v", got, want)
			}
			if size, _ := textStyle(t, tt, "Ready"); !near(size, 11*scale) {
				t.Errorf("the text is %v, want the Caption's %v", size, 11*scale)
			}
		})
	}
}

// The app's own text size on a StatusLabel wins over the skin's Caption.
func TestOwnTextSizeWinsOverCaption(t *testing.T) {
	tt := showSkin(t, widget.StatusLabel("Ready").TextSize(widget.XL), 600, 400, roleSentinel(), false, 1)
	size, _ := textStyle(t, tt, "Ready")
	if want := Calm.Tokens(false).FontSize * 1.25; !near(size, want) {
		t.Errorf("the text is %v, want the app's XL, %v", size, want)
	}
}
